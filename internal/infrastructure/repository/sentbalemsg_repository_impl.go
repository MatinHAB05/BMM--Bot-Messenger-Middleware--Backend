package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	redisApt "messenger-backend/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

type sentBaleMsgRepository struct {
	redisClient redisApt.Cache
}

func NewSentBaleMsgRepository(redisClient redisApt.Cache) repository_contract.SentBaleMsgRepository {
	return &sentBaleMsgRepository{redisClient: redisClient}
}

func sentBaleMsgKeyFunc(baleChatID string, contentHash string) string {
	return fmt.Sprintf("sent_bale_msg:%s:%s", baleChatID, contentHash)
}

func (r *sentBaleMsgRepository) Set(ctx context.Context, baleChatID string, contentHash string, ttl time.Duration) error {
	key := sentBaleMsgKeyFunc(baleChatID, contentHash)
	if err := r.redisClient.GetRDB().Set(ctx, key, true, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

func (r *sentBaleMsgRepository) Get(ctx context.Context, baleChatID string, contentHash string) (bool, error) {
	key := sentBaleMsgKeyFunc(baleChatID, contentHash)
	val, err := r.redisClient.GetRDB().Get(ctx, key).Bool()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, exception.ErrSentBaleMsgNotFound
		}
		return false, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	return val, nil
}

func (r *sentBaleMsgRepository) Exists(ctx context.Context, baleChatID string, contentHash string) (bool, error) {
	key := sentBaleMsgKeyFunc(baleChatID, contentHash)
	count, err := r.redisClient.GetRDB().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return count > 0, nil
}

func (r *sentBaleMsgRepository) Delete(ctx context.Context, baleChatID string, contentHash string) error {
	key := sentBaleMsgKeyFunc(baleChatID, contentHash)
	res := r.redisClient.GetRDB().Del(ctx, key)
	if err := res.Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	if res.Val() == 0 {
		return exception.ErrSentBaleMsgNotFound
	}

	return nil
}

func (r *sentBaleMsgRepository) TTL(ctx context.Context, baleChatID string, contentHash string) (time.Duration, error) {
	key := sentBaleMsgKeyFunc(baleChatID, contentHash)
	ttl, err := r.redisClient.GetRDB().TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	// -2 = nt found
	if ttl == -2 {
		return 0, exception.ErrSentBaleMsgNotFound
	}

	return ttl, nil
}
