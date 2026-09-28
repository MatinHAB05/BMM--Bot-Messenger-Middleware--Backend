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
	if err := r.rdb(ctx).Set(ctx, key, companyID, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}

func (r *ChannelPendingRepository) Get(ctx context.Context, userID string) (uint, error) {
	key := keyFunc(userID)
	val, err := r.rdb(ctx).Get(ctx, key).Result()
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
	count, err := r.rdb(ctx).Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return count > 0, nil
}

func (r *ChannelPendingRepository) Delete(ctx context.Context, userID string) error {
	key := keyFunc(userID)
	res := r.rdb(ctx).Del(ctx, key)
	if err := res.Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}

	if res.Val() == 0 {
		return exception.ErrOTPNotFound
	}

	return nil
}

// rdb returns the trx-aware Redis client: inside TrxManager.WithWatch /
// WithTransaction it is the *redis.Tx / Pipeliner carried by ctx, otherwise
// the plain client.
func (r *ChannelPendingRepository) rdb(ctx context.Context) redis.Cmdable {
	return redisApt.ExtractTrxOrCache(ctx, r.redisClient).GetRDB()
}

// Key exposes the key this repo uses for userID (for TrxManager.WithWatch).
func (r *ChannelPendingRepository) Key(userID string) string {
	return keyFunc(userID)
}

// Remove deletes the key WITHOUT inspecting the reply, so it is safe inside
// the write phase of a transaction (where replies are not available yet).
func (r *ChannelPendingRepository) Remove(ctx context.Context, userID string) error {
	if err := r.rdb(ctx).Del(ctx, keyFunc(userID)).Err(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrCacheOperation, err)
	}
	return nil
}
