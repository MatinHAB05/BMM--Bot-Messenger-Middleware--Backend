// internal/infrastructure/repository/redis_rate_limiter_repository.go
package repository

import (
	"context"
	"fmt"
	repository_contract "messenger-backend/internal/domain/repository"
	redis_adapter "messenger-backend/internal/infrastructure/redis"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisRateLimiterRepository struct {
	client redis_adapter.Cache
}

func NewRedisRateLimiterRepository(client redis_adapter.Cache) repository_contract.RateLimiterRepository {
	return &redisRateLimiterRepository{
		client: client,
	}
}

func (r *redisRateLimiterRepository) RecordAndCount(
	ctx context.Context,
	key string,
	member string,
	windowStart int64,
	now time.Time,
	window time.Duration,
) (int64, error) {
	pipe := r.client.GetRDB().TxPipeline()

	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: member})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("redis rate limiter pipeline execution failed: %w", err)
	}

	count, err := countCmd.Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get count from rate limiter pipeline: %w", err)
	}

	return count, nil
}
