package balehandlers

import (
	"context"
	"fmt"
	"strconv"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	telegrammiddleware "messenger-backend/internal/presentation/middleware/telegram"
	"messenger-backend/pkg/logger"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type FeatChannelHandler struct {
	chatLinkService service_contract.ChatLinkService
	botUsername     string
	errLog          logger.Logger
}

func NewFeatChannelHandler(
	chatLinkService service_contract.ChatLinkService,
	botUsername string,
	errLog logger.Logger,
) *FeatChannelHandler {
	return &FeatChannelHandler{
		chatLinkService: chatLinkService,
		botUsername:     botUsername,
		errLog:          errLog.With(logger.String("component", "FeatChannelHandler_Telegram")),
	}
}

func (h *FeatChannelHandler) SetChannelPendingDeepStartChannelCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	info, ok := telegrammiddleware.GetDeepStartChannelOTP(ctx)
	if !ok {
		h.errLog.Error(exception.ErrMiddlewareTokensAssetion, "failed to retrieve deep start channel OTP info from middleware context", logger.Any("otp_info", info))
		return
	}

	if err := h.chatLinkService.PrepareChannelLink(ctx, info.Code, info.TargetUserID); err != nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: info.PlatformChatID,
		Text:   "Welcome!\nWe're ready to connect your Telegram channel to your company dashboard.\n\nClick the button below to select your channel and grant the bot administrative access.",
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "📢 Select & Add Channel", URL: fmt.Sprintf("https://t.me/%s?startchannel&admin=post_messages+edit_messages+delete_messages", h.botUsername)},
				},
			},
		},
	})
	if err != nil {
		h.errLog.Error(err, "failed to send channel setup message", logger.Any("chat_id", info.PlatformChatID))
	}
}

func (h *FeatChannelHandler) RegisterChannelAcceptance(ctx context.Context, b *bot.Bot, update *models.Update) {
	joinedChatID, approverUserID, ok := telegrammiddleware.GetBotJoinedChannelInfo(ctx)
	if !ok {
		h.errLog.Error(exception.ErrMiddlewareTokensAssetion, "failed to retrieve bot joined channel info from middleware context")
		return
	}

	platformChatID := strconv.FormatInt(joinedChatID, 10)
	approverID := strconv.FormatInt(approverUserID, 10)

	if _, err := h.chatLinkService.ConfirmChannelLink(ctx, string(entity.PlatformTelegram), platformChatID, approverID); err != nil {
		return
	}

	sendText(ctx, b, h.errLog, joinedChatID, "✅ Channel connected successfully!")
}
