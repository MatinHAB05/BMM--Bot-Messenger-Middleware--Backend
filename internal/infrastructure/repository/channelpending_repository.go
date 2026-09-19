package repository

import (
	"context"
	"errors"
	"fmt"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	redisApt "messenger-backend/internal/infrastructure/redis"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// todo : redis trx manager for all redis base repos
type ChannelPendingRepository struct {
	redisClient redisApt.Cache
}

func NewChannelPendingRepository(redisClient redisApt.Cache) repository_contract.ChannelPendingRepository {
	return &ChannelPendingRepository{redisClient: redisClient}
}

func keyFunc(userID string) string {
	return fmt.Sprintf("otp:channel:pending:%s", userID)
}

func (r *ChannelPendingRepository) Set(ctx context.Context, userID string, companyID uint, ttl time.Duration) error {
	key := keyFunc(userID)
	if err := r.redisClient.GetRDB().Set(ctx, key, companyID, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

func (r *ChannelPendingRepository) Get(ctx context.Context, userID string) (uint, error) {
	key := keyFunc(userID)
	val, err := r.redisClient.GetRDB().Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, exception.ErrOTPNotFound
		}
		return 0, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	companyID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid company_id type: %v", exception.ErrCacheOperation, err)
	}

	return uint(companyID), nil
}

func (r *ChannelPendingRepository) Exists(ctx context.Context, userID string) (bool, error) {
	key := keyFunc(userID)
	count, err := r.redisClient.GetRDB().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return count > 0, nil
}

func (r *ChannelPendingRepository) Delete(ctx context.Context, userID string) error {
	key := keyFunc(userID)
	res := r.redisClient.GetRDB().Del(ctx, key)
	if err := res.Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	if res.Val() == 0 {
		return exception.ErrOTPNotFound
	}

	return nil
}
