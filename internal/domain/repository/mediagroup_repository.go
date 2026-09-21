package repository_contract

import (
	"context"
	"time"
)

type MediaGroupRepository interface {
	Set(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint, ttl time.Duration) error
	Get(ctx context.Context, mediaGroupID string, chatID string) (uint, error)
	Exists(ctx context.Context, mediaGroupID string, chatID string) (bool, error)
	Delete(ctx context.Context, mediaGroupID string, chatID string) error
	GetAndDelete(ctx context.Context, mediaGroupID string, chatID string) (uint, error)
	ForceGet(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint, ttl time.Duration) (uint, error)
	TTL(ctx context.Context, mediaGroupID string, chatID string) (time.Duration, error)
}
