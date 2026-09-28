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
	redisApt "messenger-backend/internal/infrastructure/redis"
	"messenger-backend/pkg/logger"
)

type SentBaleMsgServiceConfig struct {
	TTL time.Duration
}

type sentBaleMsgService struct {
	repo repository_contract.SentBaleMsgRepository
	trx  redisApt.TrxManager
	log  logger.Logger

	cfg SentBaleMsgServiceConfig
}

func NewSentBaleMsgService(
	repo repository_contract.SentBaleMsgRepository,
	trx redisApt.TrxManager,
	log logger.Logger,
	cfg SentBaleMsgServiceConfig,
) service_contract.SentBaleMsgService {
	return &sentBaleMsgService{
		repo: repo,
		trx:  trx,
		log:  log.With(logger.String("component", "sent_bale_msg_service")),
		cfg:  cfg,
	}
}

func (s *sentBaleMsgService) SetSentBaleMsg(ctx context.Context, baleChatID string, content string) error {
	h := s.hash(content)
	if err := s.repo.Set(ctx, baleChatID, h, s.cfg.TTL); err != nil {
		s.log.Error(err, "failed to set sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		return err
	}
	return nil
}

func (s *sentBaleMsgService) GetSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	h := s.hash(content)
	val, err := s.repo.Get(ctx, baleChatID, h)
	if err != nil {
		s.log.Error(err, "failed to get sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		return nil, err
	}
	return &val, nil
}

func (s *sentBaleMsgService) HasSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	h := s.hash(content)
	exists, err := s.repo.Exists(ctx, baleChatID, h)
	if err != nil {
		s.log.Error(err, "failed to check sent bale msg existence", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		return nil, err
	}
	return &exists, nil
}

func (s *sentBaleMsgService) RemoveSentBaleMsg(ctx context.Context, baleChatID string, content string) error {
	h := s.hash(content)
	if err := s.repo.Delete(ctx, baleChatID, h); err != nil {
		s.log.Error(err, "failed to delete sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		return err
	}
	return nil
}

// GetAndRemoveSentBaleMsg reads and deletes the record atomically (WATCH on
// the key). It also fixes the old double-hash bug: the previous version passed
// s.hash(content) into methods that hash again, so it looked up a different key.
func (s *sentBaleMsgService) GetAndRemoveSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	h := s.hash(content)
	var val bool

	err := s.trx.WithWatch(ctx, []string{s.repo.Key(baleChatID, h)},
		func(trxCtx context.Context) error {
			v, err := s.repo.Get(trxCtx, baleChatID, h)
			if err != nil {
				return err
			}
			val = v
			return nil
		},
		func(trxCtx context.Context) error {
			return s.repo.Remove(trxCtx, baleChatID, h)
		},
	)
	if err != nil {
		if !errors.Is(err, exception.ErrSentBaleMsgNotFound) {
			s.log.Error(err, "failed to get and remove sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		}
		return nil, err
	}

	return &val, nil
}

// ForceGetSentBaleMsg is "get or claim": if the record exists its value is
// returned, otherwise it is created (with TTL) and true is returned. WATCH
// guarantees two concurrent callers can't both "claim" it.
func (s *sentBaleMsgService) ForceGetSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error) {
	h := s.hash(content)
	var (
		val   bool
		found bool
	)

	err := s.trx.WithWatch(ctx, []string{s.repo.Key(baleChatID, h)},
		func(trxCtx context.Context) error {
			found = false
			v, err := s.repo.Get(trxCtx, baleChatID, h)
			if err != nil {
				if errors.Is(err, exception.ErrSentBaleMsgNotFound) {
					return nil
				}
				return err
			}
			found, val = true, v
			return nil
		},
		func(trxCtx context.Context) error {
			if found {
				return nil // nothing to write
			}
			val = true
			return s.repo.Set(trxCtx, baleChatID, h, s.cfg.TTL)
		},
	)
	if err != nil {
		s.log.Error(err, "failed to force get sent bale msg", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		return nil, err
	}

	return &val, nil
}

func (s *sentBaleMsgService) GetSentBaleMsgTTL(ctx context.Context, baleChatID string, content string) (time.Duration, error) {
	h := s.hash(content)
	ttl, err := s.repo.TTL(ctx, baleChatID, h)
	if err != nil {
		s.log.Error(err, "failed to get sent bale msg ttl", logger.String("chat_id", baleChatID), logger.String("content_hash", h))
		return 0, err
	}
	return ttl, nil
}

func (s *sentBaleMsgService) hash(content string) string {
	hashBytes := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hashBytes[:])
}
