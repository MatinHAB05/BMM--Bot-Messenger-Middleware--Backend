// internal/infrastructure/repository/redis_token_repository.go
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	redisAdp "messenger-backend/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

type redisTokenRepository struct {
	client redisAdp.Cache
}

func NewRedisTokenRepository(client redisAdp.Cache) repository_contract.AuthnTokenRepository {
	return &redisTokenRepository{
		client: client,
	}
}

func (r *redisTokenRepository) SaveRefreshSession(ctx context.Context, userID, tokenID, refreshToken string, ttl time.Duration) error {
	key := refreshSessionKey(userID, tokenID)
	if err := r.rdb(ctx).Set(ctx, key, refreshToken, ttl).Err(); err != nil {
		return fmt.Errorf("redis set refresh session: %w", err)
	}
	return nil
}

func (r *redisTokenRepository) GetRefreshSession(ctx context.Context, userID, tokenID string) (string, error) {
	key := refreshSessionKey(userID, tokenID)
	val, err := r.rdb(ctx).Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", exception.ErrTokenNotFound
		}
		return "", fmt.Errorf("redis get refresh session: %w", err)
	}
	return val, nil
}

func (r *redisTokenRepository) DeleteRefreshSession(ctx context.Context, userID, tokenID string) (string, error) {
	key := refreshSessionKey(userID, tokenID)
	if err := r.rdb(ctx).Del(ctx, key).Err(); err != nil {
		return key, fmt.Errorf("redis delete refresh session: %w", err)
	}
	return key, nil
}

func (r *redisTokenRepository) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := blacklistKey(jti)
	if err := r.rdb(ctx).Set(ctx, key, "revoked", ttl).Err(); err != nil {
		return fmt.Errorf("redis blacklist access token: %w", err)
	}
	return nil
}

func (r *redisTokenRepository) DeleteUserRefreshSessions(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("refresh_token:%s:*", userID)
	iter := r.rdb(ctx).Scan(ctx, 0, pattern, 0).Iterator()

	var deleteErr error
	for iter.Next(ctx) {
		if err := r.rdb(ctx).Del(ctx, iter.Val()).Err(); err != nil {
			deleteErr = err
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis scan refresh sessions: %w", err)
	}
	if deleteErr != nil {
		return fmt.Errorf("redis delete user refresh session: %w", deleteErr)
	}

	return nil
}

func (r *redisTokenRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := blacklistKey(jti)
	count, err := r.rdb(ctx).Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists check failed: %w", err)
	}
	return count > 0, nil
}

func refreshSessionKey(userID, tokenID string) string {
	return fmt.Sprintf("refresh_token:%s:%s", userID, tokenID)
}

func blacklistKey(jti string) string {
	return fmt.Sprintf("blacklist:access_token:%s", jti)
}

// rdb returns the trx-aware Redis client (see ChannelPendingRepository.rdb).
// NOTE: DeleteUserRefreshSessions uses SCAN, which needs real replies, so
// call it OUTSIDE WithTransaction / the write phase of WithWatch.
func (r *redisTokenRepository) rdb(ctx context.Context) redis.Cmdable {
	return redisAdp.ExtractTrxOrCache(ctx, r.client).GetRDB()
}

func (r *redisTokenRepository) RefreshSessionKey(userID, tokenID string) string {
	return refreshSessionKey(userID, tokenID)
}

func (r *redisTokenRepository) BlacklistKey(jti string) string {
	return blacklistKey(jti)
}
