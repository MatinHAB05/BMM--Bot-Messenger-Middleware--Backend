package balehandlers

import (
	"context"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	telegrammiddleware "messenger-backend/internal/presentation/middleware/telegram"
	"messenger-backend/pkg/logger"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type DirectFeatChatCommandHandler struct {
	chatLinkService service_contract.ChatLinkService
	errLog          logger.Logger
}

func NewDirectFeatChatCommandHandler(
	chatLinkService service_contract.ChatLinkService,
	errLog logger.Logger,
) *DirectFeatChatCommandHandler {
	return &DirectFeatChatCommandHandler{
		chatLinkService: chatLinkService,
		errLog:          errLog.With(logger.String("component", "DirectFeatChatCommandHandler_Telegram")),
	}
}

func (h *DirectFeatChatCommandHandler) FeatChatsLinkCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	info, ok := telegrammiddleware.GetLinkOTP(ctx)
	if !ok {
		h.errLog.Error(exception.ErrMiddlewareTokensAssetion, "failed to retrieve link OTP info from middleware context", logger.Any("link_otp_info", info))
		return
	}

	h.linkAndReply(ctx, b, info.PlatformChatID, info.Code)
}

func (h *DirectFeatChatCommandHandler) FeatDeepStartGroupCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	info, ok := telegrammiddleware.GetDeepStartGroupOTP(ctx)
	if !ok {
		h.errLog.Error(exception.ErrMiddlewareTokensAssetion, "failed to retrieve deep start group OTP info from middleware context", logger.Any("deep_start_otp_info", info))
		return
	}

	h.linkAndReply(ctx, b, info.PlatformChatID, info.Code)
}

// linkAndReply is the "call one service, then reply" tail shared by both
// commands above.
func (h *DirectFeatChatCommandHandler) linkAndReply(ctx context.Context, b *bot.Bot, platformChatID, otpCode string) {
	if _, err := h.chatLinkService.LinkChatToCompany(ctx, string(entity.PlatformTelegram), platformChatID, otpCode); err != nil {
		// service already logged the failure; original behavior is to
		// stay silent to the user on internal errors, so we do too.
		return
	}

	sendText(ctx, b, h.errLog, platformChatID, "✅ Done ✅")
}
