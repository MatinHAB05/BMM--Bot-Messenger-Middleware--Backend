package service

import (
	"context"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
)

type ChannelPendingServiceConfig struct {
	TTL time.Duration
}

type ChannelPendingService struct {
	repo repository_contract.ChannelPendingRepository
	log  logger.Logger

	cfg ChannelPendingServiceConfig
}

func NewChannelPendingService(
	repo repository_contract.ChannelPendingRepository,
	log logger.Logger,
	cfg ChannelPendingServiceConfig,
) service_contract.ChannelPendingService {
	return &ChannelPendingService{
		repo: repo,
		log:  log.With(logger.String("component", "pending_channel_service")),
		cfg:  cfg,
	}
}

func (s *ChannelPendingService) SetPendingChannel(ctx context.Context, userID string, companyID uint) error {
	if err := s.repo.Set(ctx, userID, companyID, s.cfg.TTL); err != nil {
		s.log.Error(err, "failed to set pending channel", logger.String("user_id", userID), logger.Uint("company_id", companyID))
		return err
	}
	return nil
}

func (s *ChannelPendingService) GetPendingChannel(ctx context.Context, userID string) (*uint, error) {
	companyID, err := s.repo.Get(ctx, userID)
	if err != nil {
		s.log.Error(err, "failed to get pending channel", logger.String("user_id", userID))
		return nil, err
	}
	return &companyID, nil
}

func (s *ChannelPendingService) HasPendingChannel(ctx context.Context, userID string) (*bool, error) {
	exists, err := s.repo.Exists(ctx, userID)
	if err != nil {
		s.log.Error(err, "failed to check pending channel existence", logger.String("user_id", userID))
		return nil, err
	}
	return &exists, nil
}

func (s *ChannelPendingService) RemovePendingChannel(ctx context.Context, userID string) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		s.log.Error(err, "failed to delete pending channel", logger.String("user_id", userID))
		return err
	}
	return nil
}

func (s *ChannelPendingService) GetAndRemovePendingChannel(ctx context.Context, userTelegramID string) (*uint, error) {
	companyID, err := s.GetPendingChannel(ctx, userTelegramID)
	if err != nil {
		return nil, err
	}
	err = s.RemovePendingChannel(ctx, userTelegramID)
	if err != nil {
		return nil, err
	}

	return companyID, nil
}
