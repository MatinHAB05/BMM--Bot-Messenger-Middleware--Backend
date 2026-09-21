package service

import (
	"context"
	"errors"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
)

type MediaGroupServiceConfig struct {
	TTL time.Duration
}

type MediaGroupService struct {
	repo repository_contract.MediaGroupRepository
	log  logger.Logger

	cfg MediaGroupServiceConfig
}

func NewMediaGroupService(
	repo repository_contract.MediaGroupRepository,
	log logger.Logger,
	cfg MediaGroupServiceConfig,
) service_contract.MediaGroupService {
	return &MediaGroupService{
		repo: repo,
		log:  log.With(logger.String("component", "media_group_service")),
		cfg:  cfg,
	}
}

func (s *MediaGroupService) SetMediaGroup(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint) error {
	if err := s.repo.Set(ctx, mediaGroupID, chatID, chatHistoryID, s.cfg.TTL); err != nil {
		s.log.Error(err, "failed to set media group", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID), logger.Uint("chat_history_id", chatHistoryID))
		return err
	}
	return nil
}

func (s *MediaGroupService) GetMediaGroup(ctx context.Context, mediaGroupID string, chatID string) (*uint, error) {
	chatHistoryID, err := s.repo.Get(ctx, mediaGroupID, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrMediaGroupMsgNotFound) {
			return nil, err
		}
		s.log.Error(err, "failed to get media group", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID))
		return nil, err
	}
	return &chatHistoryID, nil
}

func (s *MediaGroupService) HasMediaGroup(ctx context.Context, mediaGroupID string, chatID string) (*bool, error) {
	exists, err := s.repo.Exists(ctx, mediaGroupID, chatID)
	if err != nil {
		s.log.Error(err, "failed to check media group existence", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID))
		return nil, err
	}
	return &exists, nil
}

func (s *MediaGroupService) RemoveMediaGroup(ctx context.Context, mediaGroupID string, chatID string) error {
	if err := s.repo.Delete(ctx, mediaGroupID, chatID); err != nil {
		s.log.Error(err, "failed to delete media group", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID))
		return err
	}
	return nil
}

func (s *MediaGroupService) GetAndRemoveMediaGroup(ctx context.Context, mediaGroupID string, chatID string) (*uint, error) {
	chatHistoryID, err := s.repo.GetAndDelete(ctx, mediaGroupID, chatID)
	if err != nil {
		s.log.Error(err, "failed to get and delete media group", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID))
		return nil, err
	}
	return &chatHistoryID, nil
}

func (s *MediaGroupService) ForceGetMediaGroup(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint) (*uint, error) {
	result, err := s.repo.ForceGet(ctx, mediaGroupID, chatID, chatHistoryID, s.cfg.TTL)
	if err != nil {
		s.log.Error(err, "failed to force get media group", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID))
		return nil, err
	}
	return &result, nil
}

func (s *MediaGroupService) GetMediaGroupTTL(ctx context.Context, mediaGroupID string, chatID string) (*time.Duration, error) {
	ttl, err := s.repo.TTL(ctx, mediaGroupID, chatID)
	if err != nil {
		s.log.Error(err, "failed to get media group ttl", logger.String("media_group_id", mediaGroupID), logger.String("chat_id", chatID))
		return nil, err
	}
	return &ttl, nil
}
