// service/chat_link_service_impl.go
package service

import (
	"context"
	"strconv"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/otp"
	"messenger-backend/pkg/logger"
)

type chatLinkService struct {
	chatService           service_contract.ChatService
	otpService            service_contract.OTPService
	channelPendingService service_contract.ChannelPendingService

	log    logger.Logger
	errLog logger.Logger
}

func NewChatLinkService(
	chatService service_contract.ChatService,
	otpService service_contract.OTPService,
	channelPendingService service_contract.ChannelPendingService,
	log logger.Logger,
	errLog logger.Logger,
) service_contract.ChatLinkService {
	return &chatLinkService{
		chatService:           chatService,
		otpService:            otpService,
		channelPendingService: channelPendingService,
		log:                   log.With(logger.String("component", "chat_link_service")),
		errLog:                errLog.With(logger.String("component", "chat_link_service")),
	}
}

// resolveLinkCompanyID validates and invalidates otpCode as a TypeLink OTP
// and returns the companyID it carries. Shared by every flow below.
//
// NOTE: both the direct-link/group flow AND the channel flow currently use
// the same otp.TypeLink -- see the earlier review note about that.
func (s *chatLinkService) resolveLinkCompanyID(ctx context.Context, otpCode string) (uint, error) {
	rawPayload, err := s.otpService.GetAndInvalidateOTP(ctx, "==dose not matter==", otp.TypeLink, otpCode)
	if err != nil {
		s.errLog.Error(err, "failed to retrieve or invalidate OTP", logger.String("otp_code", otpCode))
		return 0, err
	}

	payload, ok := rawPayload.(*otp.LinkChatCompanyPayload)
	if !ok {
		s.errLog.Error(exception.ErrInvalidPayload, "failed to type-assert OTP payload to LinkChatCompanyPayload", logger.Any("raw_payload", rawPayload))
		return 0, exception.ErrInvalidPayload
	}

	companyID, err := strconv.ParseUint(payload.CompanyID, 10, 64)
	if err != nil {
		s.errLog.Error(err, "failed to parse company ID from OTP payload", logger.String("company_id_raw", payload.CompanyID))
		return 0, err
	}

	return uint(companyID), nil
}

func (s *chatLinkService) LinkChatToCompany(ctx context.Context, platform, platformChatID, otpCode string) (*service_contract.ChatResponse, error) {
	companyID, err := s.resolveLinkCompanyID(ctx, otpCode)
	if err != nil {
		return nil, err
	}

	targetChat, err := s.chatService.GetByPlatformID(ctx, platform, platformChatID)
	if err != nil {
		s.errLog.Error(err, "failed to fetch chat by platform ID", logger.String("platform_chat_id", platformChatID))
		return nil, err
	}

	updatedChat, err := s.chatService.UpdateCompanyID(ctx, companyID, targetChat.ID)
	if err != nil {
		s.errLog.Error(err, "failed to update chat company ID", logger.Uint("company_id", companyID), logger.Any("chat_id", targetChat.ID))
		return nil, err
	}

	s.log.Info("successfully linked chat to company", logger.Any("chat_id", updatedChat.ID), logger.Uint("company_id", companyID))
	return updatedChat, nil
}

func (s *chatLinkService) PrepareChannelLink(ctx context.Context, otpCode, targetUserID string) error {
	companyID, err := s.resolveLinkCompanyID(ctx, otpCode)
	if err != nil {
		return err
	}

	if err := s.channelPendingService.SetPendingChannel(ctx, targetUserID, companyID); err != nil {
		s.errLog.Error(err, "failed to set pending channel state", logger.String("target_user_id", targetUserID), logger.Uint("company_id", companyID))
		return err
	}

	s.log.Info("successfully set pending channel state", logger.String("target_user_id", targetUserID), logger.Uint("company_id", companyID))
	return nil
}

func (s *chatLinkService) ConfirmChannelLink(ctx context.Context, platform, platformChatID, approverUserID string) (*service_contract.ChatResponse, error) {
	companyID, err := s.channelPendingService.GetAndRemovePendingChannel(ctx, approverUserID)
	if err != nil {
		s.errLog.Error(err, "failed to retrieve pending channel record for approver", logger.String("approver_user_id", approverUserID))
		return nil, err
	}

	targetChat, err := s.chatService.GetByPlatformID(ctx, platform, platformChatID)
	if err != nil {
		s.errLog.Error(err, "failed to fetch chat by platform ID", logger.String("platform_chat_id", platformChatID))
		return nil, err
	}

	updatedChat, err := s.chatService.UpdateCompanyID(ctx, *companyID, targetChat.ID)
	if err != nil {
		s.errLog.Error(err, "failed to update chat company ID", logger.Uint("company_id", *companyID), logger.Any("chat_id", targetChat.ID))
		return nil, err
	}

	s.log.Info("successfully linked channel to company", logger.Any("chat_id", updatedChat.ID), logger.Uint("company_id", *companyID))
	return updatedChat, nil
}
