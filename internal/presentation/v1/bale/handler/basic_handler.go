package balehandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger/telegram"

	"github.com/go-telegram/bot"
)

type BasicHandler struct {
	chatService        service_contract.ChatService
	chatHistoryService service_contract.ChatHistoryService
	attachmentService  service_contract.AttachmentService
	sentbalemsgService service_contract.SentBaleMsgService
	storageRepo        repository_contract.StorageRepository
	bucketName         string
	log                logger.Logger
	errLog             logger.Logger
}

func NewBasicHandler(
	chatService service_contract.ChatService,
	chatHistoryService service_contract.ChatHistoryService,
	attachmentService service_contract.AttachmentService,
	sentbalemsgService service_contract.SentBaleMsgService,
	storageRepo repository_contract.StorageRepository,
	bucketName string,
	log logger.Logger,
	errLog logger.Logger,
) *BasicHandler {
	return &BasicHandler{
		chatService:        chatService,
		chatHistoryService: chatHistoryService,
		attachmentService:  attachmentService,
		sentbalemsgService: sentbalemsgService,
		storageRepo:        storageRepo,
		bucketName:         bucketName,
		log:                log.With(logger.String("component", "BasicHandler_Bale")),
		errLog:             errLog.With(logger.String("component", "BasicHandler_Bale")),
	}
}

// OnUpdate persists chat identity. If the chat isn't known yet
// (no row for this platform+platformChatID), it's half-created -- saved
// with no companyID/plan, since ingestion happens outside the tenant scope
// and company assignment is a separate step (see ChatService.HalfCreate).
//
// Unchanged from the previous version of this file.
func (h *BasicHandler) OnUpdate(ctx context.Context, u telegram.ChatUpdate) {
	h.log.Debug("bale onUpdate entry", logger.String("target_id", u.TargetID))

	if _, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformBale), u.TargetID); err == nil {
		// Chat already exists. TODO: diff title/username/etc and Update() if changed.
		h.log.Debug("bale chat already exists", logger.String("target_id", u.TargetID))
		return
	} else if !errors.Is(err, exception.ErrChatNotFound) {
		h.errLog.Error(
			err,
			"failed to lookup chat by platform ID",
			logger.String("platform_chat_id", u.TargetID),
		)
		return
	}

	if _, err := h.chatService.HalfCreate(ctx, service_contract.CreateChatRequest{
		Platform:       string(entity.PlatformBale),
		PlatformChatID: u.TargetID,
		Title:          u.Title,
		Username:       u.Username,
		ChatType:       normalizeChatType(u.ChatType),
		IsPrivate:      u.ChatType == "private",
	}); err != nil {
		h.errLog.Error(
			err,
			"failed to half-create chat on update",
			logger.String("platform_chat_id", u.TargetID),
		)
	}
}

// OnMessage persists an incoming message -- and any media it carries --
// into the chat's history, atomically, via
// AttachmentService.IngestMessageWithAttachments, the same as the
// Telegram handler now does. It does NOT assume OnUpdate already ran for
// this same update, so it resolves-or-half-creates the chat itself
// first.
//
// Bale-specific, and deliberately NOT ported from Telegram: right after
// content is resolved, we still run it through
// sentbalemsgService.HasSentBaleMsg. Bale's webhook occasionally echoes
// back messages the bot itself just sent, as if they were new incoming
// updates from the user; HasSentBaleMsg is how we tell a genuine incoming
// message apart from such an echo. That check runs BEFORE any attachment
// download/upload happens, on purpose -- an echo of our own outgoing
// message would otherwise cost a wasted Bale file fetch + MinIO upload
// for media we already have.
//
// New this round (mirroring the Telegram handler): any attachment's
// actual bytes are fetched (via telegram.FetchFile -- Bale's Bot API is
// wire-compatible with Telegram's, so the same adapter package is reused)
// and streamed into MinIO (via storageRepo.UploadFile) before the DB
// write, so CreateAttachmentRequest.StoragePath is already populated by
// the time IngestMessageWithAttachments runs. A media fetch/upload
// failure never blocks the message itself from being saved -- see
// uploadAttachmentMedia's doc comment.
func (h *BasicHandler) OnMessage(ctx context.Context, bot *bot.Bot, update *telegram.Update) {
	msg := telegram.ExtractRawMessage(update)
	if msg == nil {
		return
	}

	platformChatID := strconv.FormatInt(msg.Chat.ID, 10)

	chat, err := h.findOrHalfCreateChat(ctx, platformChatID, telegram.ChatTitle(msg.Chat), string(msg.Chat.Type))
	if err != nil {
		h.errLog.Error(
			err,
			"failed to resolve or half-create chat on message",
			logger.String("platform_chat_id", platformChatID),
		)
		return
	}

	tgAttachments, caption := telegram.ExtractAttachments(msg)

	content := msg.Text
	if content == "" {
		content = caption
	}

	ex, err := h.sentbalemsgService.HasSentBaleMsg(ctx, platformChatID, content) // ? : FUCK BALE!
	if err != nil {
		h.log.Error(err, "failed to check sent bale msg", logger.String("chat_id", platformChatID))
		return
	}
	if ex == nil || *ex {
		if ex == nil {
			h.log.Error(errors.New("has sent bale msg returned nil result"), "unexpected nil result", logger.String("chat_id", platformChatID))
		} else {
			h.log.Info("bale message already sent, skipping", logger.String("chat_id", platformChatID), logger.String("content", content))
		}
		return
	}

	attachments := toCreateAttachmentRequests(tgAttachments)
	for i := range attachments {
		h.uploadAttachmentMedia(ctx, bot, chat.CompanyID, chat.ID, &attachments[i])
	}

	rawPayload, err := json.Marshal(update)
	if err != nil {
		h.errLog.Error(err, "failed to marshal raw update payload", logger.String("platform_chat_id", platformChatID))
		rawPayload = []byte("{}")
	}

	if err := h.attachmentService.IngestMessageWithAttachments(ctx, service_contract.IngestMessageWithAttachmentsRequest{
		ChatID:            chat.ID,
		PlatformMessageID: int64(msg.ID),
		SenderID:          telegram.SenderID(msg.From),
		SenderName:        telegram.SenderDisplayName(msg.From),
		Content:           content,
		MediaType:         telegram.MediaTypeFromAttachments(tgAttachments),
		RawPayload:        rawPayload,
		MessageTimestamp:  time.Unix(int64(msg.Date), 0).UTC(),
		Attachments:       attachments,
		MediaGroupID:      msg.MediaGroupID,
	}); err != nil {
		h.errLog.Error(
			err,
			"failed to ingest message with attachments",
			logger.String("platform_chat_id", platformChatID),
		)
	}
}

// uploadAttachmentMedia fetches att's file (and thumbnail, if any) via
// Bale's Bot API and streams them into MinIO, populating
// att.StoragePath/ThumbnailStoragePath on success. Failures are logged,
// not propagated: a message whose media didn't upload is still worth
// saving (PlatformFileID stays on the row as a fallback a retry job could
// use to fetch the bytes again later) -- losing the whole message over a
// transient MinIO or Bale hiccup would be worse than a temporarily empty
// StoragePath. Same reasoning as the Telegram handler's
// uploadAttachmentMedia, which this mirrors.
func (h *BasicHandler) uploadAttachmentMedia(ctx context.Context, bot *bot.Bot, companyID *uint, chatID uint, att *service_contract.CreateAttachmentRequest) {
	if att.PlatformFileID != "" {
		objectKey := buildObjectKey(companyID, chatID, att.FileType, time.Now(), att.PlatformFileID)
		if path, err := h.fetchAndUpload(ctx, bot, att.PlatformFileID, objectKey, att.MimeType); err != nil {
			h.errLog.Error(err, "failed to fetch/upload attachment media",
				logger.String("platform_file_id", att.PlatformFileID), logger.Uint("chat_id", chatID))
		} else {
			att.StoragePath = path
		}
	}

	if att.ThumbnailPlatformFileID != "" {
		objectKey := buildObjectKey(companyID, chatID, "thumbnail", time.Now(), att.ThumbnailPlatformFileID)
		if path, err := h.fetchAndUpload(ctx, bot, att.ThumbnailPlatformFileID, objectKey, "image/jpeg"); err != nil {
			h.errLog.Error(err, "failed to fetch/upload attachment thumbnail",
				logger.String("thumbnail_platform_file_id", att.ThumbnailPlatformFileID), logger.Uint("chat_id", chatID))
		} else {
			att.ThumbnailStoragePath = path
		}
	}
}

// fetchAndUpload resolves platformFileID to bytes via the getFile +
// download-link flow (shared telegram adapter, since Bale's Bot API is
// wire-compatible with Telegram's), then streams those bytes straight
// into MinIO under objectKey -- the response body is never fully
// buffered in memory.
func (h *BasicHandler) fetchAndUpload(ctx context.Context, bot *bot.Bot, platformFileID, objectKey, contentType string) (string, error) {
	fetched, err := telegram.FetchFile(ctx, bot, platformFileID)
	if err != nil {
		return "", fmt.Errorf("fetch from telegram: %w", err)
	}
	defer fetched.Body.Close()

	storagePath, err := h.storageRepo.UploadFile(ctx, h.bucketName, objectKey, fetched.Body, fetched.FileSize, contentType)
	if err != nil {
		return "", fmt.Errorf("upload to storage: %w", err)
	}
	return storagePath, nil
}

// buildObjectKey is the structured MinIO path every bot-ingested file
// (and its thumbnail) is stored under:
// company_<id>/chats/chat_<id>/<file_type>/<year>/<month>/<file_id>.
// companyID is nil for a chat that hasn't been claimed by a company yet
// (see ChatService.HalfCreate) -- that's a normal, expected state, not an
// error, so it gets its own stable segment rather than failing the
// upload.
func buildObjectKey(companyID *uint, chatID uint, fileType string, when time.Time, fileID string) string {
	companySegment := "company_unclaimed"
	if companyID != nil {
		companySegment = fmt.Sprintf("company_%d", *companyID)
	}
	if fileType == "" {
		fileType = "other"
	}
	return fmt.Sprintf("%s/chats/chat_%d/%s/%04d/%02d/%s",
		companySegment, chatID, fileType, when.Year(), int(when.Month()), fileID)
}

// findOrHalfCreateChat centralizes the "get by platform id, half-create if
// missing" pattern shared by both handlers above.
func (h *BasicHandler) findOrHalfCreateChat(ctx context.Context, platformChatID, title, chatType string) (*service_contract.ChatResponse, error) {
	chat, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformBale), platformChatID)
	if err == nil {
		return chat, nil
	}
	if !errors.Is(err, exception.ErrChatNotFound) {
		return nil, err
	}

	return h.chatService.HalfCreate(ctx, service_contract.CreateChatRequest{
		Platform:       string(entity.PlatformBale),
		PlatformChatID: platformChatID,
		Title:          title,
		ChatType:       normalizeChatType(chatType),
		IsPrivate:      chatType == "private",
	})
}

func normalizeChatType(baleChatType string) string {
	switch baleChatType {
	case "group", "supergroup", "channel":
		return baleChatType
	default:
		return "group"
	}
}
