package repository_contract

import (
	"context"
	"time"
)

type ChannelPendingRepository interface {
	Set(ctx context.Context, userID string, companyID uint, ttl time.Duration) error
	Get(ctx context.Context, userID string) (uint, error)
	Exists(ctx context.Context, userID string) (bool, error)
	Delete(ctx context.Context, userID string) error
	Key(userID string) string
	Remove(ctx context.Context, userID string) error
}
