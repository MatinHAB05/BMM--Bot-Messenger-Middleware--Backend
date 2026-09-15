package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
)

type SentBaleMsgServiceConfig struct {
	TTL time.Duration
}

type sentBaleMsgService struct {
	repo repository_contract.SentBaleMsgRepository
	log  logger.Logger

	cfg SentBaleMsgServiceConfig
}

func NewSentBaleMsgService(
	repo repository_contract.SentBaleMsgRepository,
	log logger.Logger,
	cfg SentBaleMsgServiceConfig,
) service_contract.SentBaleMsgService {
	return &sentBaleMsgService{
		repo: repo,
		log:  log.With(logger.String("component", "sent_bale_msg_service")),
		cfg:  cfg,
	}
}

func (s *sentBaleMsgService) SetSentBaleMsg(ctx context.Context, baleChatID string, content string) error {
	if err := s.repo.Set(ctx, baleChatID, s.hash(content), s.cfg.TTL); err != nil {
		s.log.Error(err, "failed to set sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
		return err
	}
	return nil
}

func (s *sentBaleMsgService) GetSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	val, err := s.repo.Get(ctx, baleChatID, s.hash(content))
	if err != nil {
		s.log.Error(err, "failed to get sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
		return nil, err
	}
	return &val, nil
}

func (s *sentBaleMsgService) HasSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	exists, err := s.repo.Exists(ctx, baleChatID, s.hash(content))
	if err != nil {
		s.log.Error(err, "failed to check sent bale msg existence", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
		return nil, err
	}
	return &exists, nil
}

func (s *sentBaleMsgService) RemoveSentBaleMsg(ctx context.Context, baleChatID string, content string) error {
	if err := s.repo.Delete(ctx, baleChatID, s.hash(content)); err != nil {
		s.log.Error(err, "failed to delete sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
		return err
	}
	return nil
}

func (s *sentBaleMsgService) GetAndRemoveSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	val, err := s.GetSentBaleMsg(ctx, baleChatID, s.hash(content))
	if err != nil {
		return nil, err
	}

	if err := s.RemoveSentBaleMsg(ctx, baleChatID, s.hash(content)); err != nil {
		return nil, err
	}

	return val, nil
}

func (s *sentBaleMsgService) ForceGetSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	val, err := s.repo.Get(ctx, baleChatID, s.hash(content))
	if err != nil {
		if !errors.Is(err, exception.ErrSentBaleMsgNotFound) {
			s.log.Error(err, "failed to force get sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
			return nil, err
		}

		if err := s.repo.Set(ctx, baleChatID, s.hash(content), s.cfg.TTL); err != nil {
			s.log.Error(err, "failed to set sent bale msg on force get", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
			return nil, err
		}
		val = true
	}

	return &val, nil
}

func (s *sentBaleMsgService) GetSentBaleMsgTTL(ctx context.Context, baleChatID string, content string) (time.Duration, error) {
	ttl, err := s.repo.TTL(ctx, baleChatID, s.hash(content))
	if err != nil {
		s.log.Error(err, "failed to get sent bale msg ttl", logger.String("chat_id", baleChatID), logger.String("content_hash", s.hash(content)))
		return 0, err
	}
	return ttl, nil
}

func (s *sentBaleMsgService) hash(content string) string {
	hashBytes := sha256.Sum256([]byte(content))
	contentHash := hex.EncodeToString(hashBytes[:])
	return contentHash
}
