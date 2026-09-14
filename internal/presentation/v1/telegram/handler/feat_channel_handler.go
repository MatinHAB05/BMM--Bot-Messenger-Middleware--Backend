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

type FeatChannelHandler struct {
	chatService           service_contract.ChatService
	otpService            service_contract.OTPService
	channelPendingService service_contract.ChannelPendingService
	log                   logger.Logger
	errLog                logger.Logger
}

func NewFeatChannelHandler(
	chatService service_contract.ChatService,
	otpService service_contract.OTPService,
	channelPendingService service_contract.ChannelPendingService,
	log logger.Logger,
	errLog logger.Logger,
) *FeatChannelHandler {
	return &FeatChannelHandler{
		chatService:           chatService,
		otpService:            otpService,
		channelPendingService: channelPendingService,
		log:                   log.With(logger.String("component", "FeatChannelHandler_Telegram")),
		errLog:                errLog.With(logger.String("component", "FeatChannelHandler_Telegram")),
	}
}

func (h *FeatChannelHandler) SetChannelPendingDeepStartChannelCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	info, ok := telegrammiddleware.GetDeepStartChannelOTP(ctx)
	if !ok {
		h.errLog.Error(
			exception.ErrMiddlewareTokensAssetion,
			"failed to retrieve deep start channel OTP info from middleware context",
			logger.Any("otp_info", info),
		)
		return
	}

	rawPayload, err := h.otpService.GetAndInvalidateOTP(ctx, "", otp.TypeLink, info.Code)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to retrieve or invalidate OTP",
			logger.String("otp_code", info.Code),
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

	err = h.channelPendingService.SetPendingChannel(ctx, info.TargetUserID, uint(companyID))
	if err != nil {
		h.errLog.Error(
			err,
			"failed to set pending channel state",
			logger.String("target_user_id", info.TargetUserID),
			logger.Uint64("company_id", companyID),
		)
		return
	}

	h.log.Info(
		"successfully set pending channel state",
		logger.String("platform_target_user_id", info.TargetUserID),
		logger.Uint64("company_id", companyID),
	)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: info.PlatformChatID,
		Text:   "Welcome!\nWe're ready to connect your Telegram channel to your company dashboard.\n\nClick the button below to select your channel and grant the bot administrative access.",
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "📢 Select & Add Channel", CallbackData: "select_channel"},
				},
			},
		},
	})
	if err != nil {
		h.errLog.Error(
			err,
			"failed to send channel setup message",
			logger.Any("chat_id", info.PlatformChatID),
		)
		return
	}
}

func (h *FeatChannelHandler) RegisterChannelAcceptance(ctx context.Context, b *bot.Bot, update *models.Update) {
	joinedChatID, approverUserID, ok := telegrammiddleware.GetBotJoinedChannelInfo(ctx)
	if !ok {
		h.errLog.Error(
			exception.ErrMiddlewareTokensAssetion,
			"failed to retrieve bot joined channel info from middleware context",
		)
		return
	}

	companyID, err := h.channelPendingService.GetAndRemovePendingChannel(ctx, strconv.FormatInt(approverUserID, 10))
	if err != nil {
		h.errLog.Error(
			err,
			"failed to retrieve pending channel record for approver",
			logger.Int64("approver_user_id", approverUserID),
		)
		return
	}

	targetChat, err := h.chatService.GetByPlatformID(ctx, string(entity.PlatformTelegram), strconv.FormatInt(joinedChatID, 10))
	if err != nil {
		h.errLog.Error(
			err,
			"failed to fetch chat by platform ID",
			logger.Int64("platform_joined_chat_id", joinedChatID),
		)
		return
	}

	updatedChat, err := h.chatService.UpdateCompanyID(ctx, uint(*companyID), targetChat.ID)
	if err != nil {
		h.errLog.Error(
			err,
			"failed to update chat company ID",
			logger.Uint("company_id", *companyID),
			logger.Any("chat_id", targetChat.ID),
		)
		return
	}

	h.log.Info(
		"successfully linked channel to company",
		logger.Any("chat_id", updatedChat.ID),
		logger.Uint("company_id", *companyID),
	)

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: joinedChatID,
		Text:   "✅ Channel connected successfully!",
	})
	if err != nil {
		h.errLog.Error(
			err,
			"failed to send channel connection confirmation message",
			logger.Int64("chat_id", joinedChatID),
		)
	}
}
