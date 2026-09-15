package balehandlers

import (
	"context"
	"errors"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger/telegram"

	"gorm.io/datatypes"
)

type BasicHandler struct {
	chatService        service_contract.ChatService
	chatHistoryService service_contract.ChatHistoryService
	sentbalemsgService service_contract.SentBaleMsgService
	log                logger.Logger
	errLog             logger.Logger
}

func NewBasicHandler(
	chatService service_contract.ChatService,
	chatHistoryService service_contract.ChatHistoryService,
	sentbalemsgService service_contract.SentBaleMsgService,
	log logger.Logger,
	errLog logger.Logger,
) *BasicHandler {
	return &BasicHandler{
		chatService:        chatService,
		chatHistoryService: chatHistoryService,
		sentbalemsgService: sentbalemsgService,
		log:                log.With(logger.String("component", "BasicHandler_Bale")),
		errLog:             errLog.With(logger.String("component", "BasicHandler_Bale")),
	}
}

// OnUpdate persists chat identity. If the chat isn't known yet
// (no row for this platform+platformChatID), it's half-created -- saved
// with no companyID/plan, since ingestion happens outside the tenant scope
// and company assignment is a separate step (see ChatService.HalfCreate).
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

// OnMessage persists an incoming message into the chat's history.
// It does NOT assume OnUpdate already ran for this same update
// (the two callbacks are independent), so it resolves-or-half-creates the
// chat itself first.
func (h *BasicHandler) OnMessage(ctx context.Context, msg telegram.MessageUpdate) {
	chat, err := h.findOrHalfCreateChat(ctx, msg.TargetID, msg.ChatTitle, msg.ChatType)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to resolve or half-create chat on message",
			logger.String("platform_chat_id", msg.TargetID),
		)
		return
	}

	ex, err := h.sentbalemsgService.HasSentBaleMsg(ctx, msg.TargetID, msg.Content) // ? : FUCK BALE!
	if err != nil {
		h.log.Error(err, "failed to check sent bale msg", logger.String("chat_id", msg.TargetID))
		return
	}
	if ex == nil || *ex {
		if ex == nil {
			h.log.Error(errors.New("has sent bale msg returned nil result"), "unexpected nil result", logger.String("chat_id", msg.TargetID))
		} else {
			h.log.Info("bale message already sent, skipping", logger.String("chat_id", msg.TargetID), logger.String("content", msg.Content))
		}
		return
	}

	if _, err := h.chatHistoryService.Upsert(ctx, &service_contract.CreateMessageRequest{
		ChatID:            chat.ID,
		PlatformMessageID: msg.PlatformMessageID,
		SenderID:          msg.SenderID,
		SenderName:        msg.SenderName,
		Content:           msg.Content,
		MediaType:         msg.MediaType,
		RawPayload:        datatypes.JSON(msg.RawPayload),
		MessageTimestamp:  msg.Timestamp,
	}); err != nil {
		h.errLog.Error(
			err,
			"failed to create history for chat",
			logger.String("platform_chat_id", msg.TargetID),
		)
	}
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
