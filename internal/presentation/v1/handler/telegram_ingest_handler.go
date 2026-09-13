package handler

import (
	"context"
	"errors"
	"log"
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger/telegram"
	"messenger-backend/pkg/tellogger"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gorm.io/datatypes"
)

type TelegramIngestHandler struct {
	chatService        service_contract.ChatService
	chatHistoryService service_contract.ChatHistoryService
	log                logger.Logger
	tellog             tellogger.Logger
}

func NewTelegramIngestHandler(
	chatService service_contract.ChatService,
	chatHistoryService service_contract.ChatHistoryService,
	log logger.Logger,
	tellog tellogger.Logger,
) *TelegramIngestHandler {
	return &TelegramIngestHandler{
		chatService:        chatService,
		chatHistoryService: chatHistoryService,
		log:                log,
		tellog:             tellog,
	}
}

// onUpdateTelegram persists chat identity. If the chat isn't known yet
// (no row for this platform+platformChatID), it's half-created -- saved
// with no companyID/plan, since ingestion happens outside the tenant scope
// and company assignment is a separate step (see ChatService.HalfCreate).
func (h *TelegramIngestHandler) OnUpdateTelegram(ctx context.Context, u telegram.ChatUpdate) {
	log.Printf("telegram: onUpdate: im heare2!")
	if _, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformTelegram), u.TargetID); err == nil {
		// Chat already exists. TODO: diff title/username/etc and Update()
		// if changed -- out of scope for now.
		log.Printf("telegram: onUpdate: im heare!")
		return
	} else if !errors.Is(err, exception.ErrChatNotFound) {
		log.Printf("telegram: onUpdate: lookup chat %s failed: %v", u.TargetID, err)
		return
	}

	if _, err := h.chatService.HalfCreate(ctx, service_contract.CreateChatRequest{
		Platform:       string(entity.PlatformTelegram),
		PlatformChatID: u.TargetID,
		Title:          u.Title,
		Username:       u.Username,
		ChatType:       normalizeChatType(u.ChatType),
		IsPrivate:      u.ChatType == "private",
	}); err != nil {
		log.Printf("telegram: onUpdate: half-create chat %s failed: %v", u.TargetID, err)
	}
}

// onMessageTelegram persists an incoming message into the chat's history.
// It does NOT assume onUpdateTelegram already ran for this same update
// (the two callbacks are independent), so it resolves-or-half-creates the
// chat itself first.
func (h *TelegramIngestHandler) OnMessageTelegram(ctx context.Context, msg telegram.MessageUpdate) {
	chat, err := h.findOrHalfCreateChat(ctx, msg.TargetID, msg.ChatTitle, msg.ChatType)
	if err != nil {
		log.Printf("telegram: onMessage: resolve chat %s failed: %v", msg.TargetID, err)
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
		log.Printf("telegram: onMessage: create history for chat %s failed: %v", msg.TargetID, err)
	}
}

// findOrHalfCreateChat centralizes the "get by platform id, half-create if
// missing" pattern shared by both handlers above.
func (h *TelegramIngestHandler) findOrHalfCreateChat(ctx context.Context, platformChatID, title, chatType string) (*service_contract.ChatResponse, error) {
	chat, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformTelegram), platformChatID)
	if err == nil {
		return chat, nil
	}
	if !errors.Is(err, exception.ErrChatNotFound) {
		return nil, err
	}

	return h.chatService.HalfCreate(ctx, service_contract.CreateChatRequest{
		Platform:       string(entity.PlatformTelegram),
		PlatformChatID: platformChatID,
		Title:          title,
		ChatType:       normalizeChatType(chatType),
		IsPrivate:      chatType == "private",
	})
}

// normalizeChatType maps Telegram's chat types onto CreateChatRequest.ChatType,
// whose binding only accepts channel/group/supergroup -- "private" has no
// slot there currently. Defaulting to "group" is a MOCK choice; needs a
// real decision (extend the binding rule? separate IsPrivate handling?).
func normalizeChatType(telegramChatType string) string {
	switch telegramChatType {
	case "group", "supergroup", "channel":
		return telegramChatType
	default:
		return "group" // TODO: "private" isn't representable yet, see above
	}
}

func (h *TelegramIngestHandler) FeatChatWithOTP(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg, _ := telegram.ExtractMessage(update)
	text := msg.Content

	telChatID := msg.TargetID

	text = strings.Trim(text, " ")
	args := strings.Split(text, " ")

	h.log.Debug("args input", logger.Any("args", args))

	if len(args) != 3 || args[0] != "/link" {
		h.log.Error(exception.ErrBadRequest, "somthing in input text of feat chat with otp is not rights", logger.Any("args", args))
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: telChatID,
			Text:   "⚠️bad request⚠️",
		})
		if err != nil {
			h.log.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
		}
		return
	}

	// we expect : /link <company-code> <otp-code>
	companyCode := args[1]
	otpCode := args[2]

	new_chat, err := h.chatService.FeatChatWithOTP(ctx, companyCode, otpCode, telChatID, string(entity.PlatformTelegram))
	if err != nil || new_chat.CompanyID == nil {
		h.log.Error(err, "err from feat chat with otp", logger.Any("new_chat", new_chat))
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: telChatID,
			Text:   "❌request failed❌",
		})
		if err != nil {
			h.log.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
		}

		return
	}

	h.log.Debug("new_chat_resp", logger.Any("new_chat", new_chat))
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: telChatID,
		Text:   "✅Done✅",
	})
	if err != nil {
		h.log.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat-id", telChatID))
	}

}
