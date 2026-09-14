package telegramhandlers

import (
	"context"
	"strconv"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/otp"
	telegrammiddleware "messenger-backend/internal/presentation/middleware/telegram"
	"messenger-backend/pkg/logger"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type DirectFeatChatCommandHandler struct {
	chatService service_contract.ChatService
	otpService  service_contract.OTPService

	log    logger.Logger
	errLog logger.Logger
}

func NewDirectFeatChatCommandHandler(
	chatService service_contract.ChatService,
	otpService service_contract.OTPService,
	log logger.Logger,
	errLog logger.Logger,
) *DirectFeatChatCommandHandler {
	return &DirectFeatChatCommandHandler{
		chatService: chatService,
		otpService:  otpService,
		log:         log.With(logger.String("component", "DirectFeatChatCommandHandler_Telegram")),
		errLog:      errLog.With(logger.String("component", "DirectFeatChatCommandHandler_Telegram")),
	}
}

func (h *DirectFeatChatCommandHandler) FeatChatsLinkCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	info, ok := telegrammiddleware.GetLinkOTP(ctx)
	if !ok {
		h.errLog.Error(
			exception.ErrMiddlewareTokensAssetion,
			"failed to retrieve link OTP info from middleware context",
			logger.Any("link_otp_info", info),
		)
		return
	}

	h.linkChatToCompany(ctx, b, info.PlatformChatID, info.Code)
}

func (h *DirectFeatChatCommandHandler) FeatDeepStartGroupCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	info, ok := telegrammiddleware.GetDeepStartGroupOTP(ctx)
	if !ok {
		h.errLog.Error(
			exception.ErrMiddlewareTokensAssetion,
			"failed to retrieve deep start group OTP info from middleware context",
			logger.Any("deep_start_otp_info", info),
		)
		return
	}

	h.linkChatToCompany(ctx, b, info.PlatformChatID, info.Code)
}

// linkChatToCompany encapsulates the shared OTP validation, chat resolution,
// company ID updates, and Telegram response dispatch for both link commands.
func (h *DirectFeatChatCommandHandler) linkChatToCompany(ctx context.Context, b *bot.Bot, platformChatID string, otpCode string) {
	rawPayload, err := h.otpService.GetAndInvalidateOTP(ctx, "", otp.TypeLink, otpCode)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to retrieve or invalidate OTP",
			logger.String("otp_code", otpCode),
		)
		return
	}

	payload, ok := rawPayload.(*otp.LinkChatCompanyPayload)
	if !ok {
		h.errLog.Error(
			exception.ErrInvalidPayload,
			"failed to type-assert OTP payload to LinkChatCompanyPayload",
			logger.Any("raw_payload", rawPayload),
		)
		return
	}

	companyID, err := strconv.ParseUint(payload.CompanyID, 10, 64)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to parse company ID from OTP payload",
			logger.String("company_id_raw", payload.CompanyID),
		)
		return
	}

	targetChat, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformTelegram), platformChatID)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to fetch chat by platform ID",
			logger.String("platform_chat_id", platformChatID),
		)
		return
	}

	updatedChat, err := h.chatService.UpdateCompanyID(ctx, uint(companyID), targetChat.ID)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to update chat company ID",
			logger.Uint64("company_id", companyID),
			logger.Any("chat_id", targetChat.ID),
		)
		return
	}

	h.log.Info(
		"successfully linked chat to company",
		logger.Any("chat_id", updatedChat.ID),
		logger.Uint64("company_id", companyID),
	)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: platformChatID,
		Text:   "✅ Done ✅",
	})
	if err != nil {
		h.errLog.Error(
			err,
			exception.ErrSendMessagePlatform.Error(),
			logger.String("platform_chat_id", platformChatID),
			logger.String("platform", "telegram"),
		)
	}
}
