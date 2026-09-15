package repository_contract

import (
	"context"
	"time"
)

type SentBaleMsgRepository interface {
	Set(ctx context.Context, baleChatID string, contentHash string, ttl time.Duration) error
	Get(ctx context.Context, baleChatID string, contentHash string) (bool, error)
	Exists(ctx context.Context, baleChatID string, contentHash string) (bool, error)
	Delete(ctx context.Context, baleChatID string, contentHash string) error
	TTL(ctx context.Context, baleChatID string, contentHash string) (time.Duration, error)
}
