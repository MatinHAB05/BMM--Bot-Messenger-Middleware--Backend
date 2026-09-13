package repository_contract

import (
	"context"
	"time"
)

type AuthnTokenRepository interface {
	SaveRefreshSession(ctx context.Context, userID, tokenID, refreshToken string, ttl time.Duration) error
	GetRefreshSession(ctx context.Context, userID, tokenID string) (string, error)
	DeleteRefreshSession(ctx context.Context, userID, tokenID string) (string, error)
	BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	DeleteUserRefreshSessions(ctx context.Context, userID string) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}
