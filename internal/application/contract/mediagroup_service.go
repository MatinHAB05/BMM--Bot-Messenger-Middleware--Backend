package service_contract

import (
	"context"
	"time"
)

type MediaGroupService interface {
	SetMediaGroup(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint) error
	GetMediaGroup(ctx context.Context, mediaGroupID string, chatID string) (*uint, error)
	GetAndRemoveMediaGroup(ctx context.Context, mediaGroupID string, chatID string) (*uint, error)
	HasMediaGroup(ctx context.Context, mediaGroupID string, chatID string) (*bool, error)
	RemoveMediaGroup(ctx context.Context, mediaGroupID string, chatID string) error
	ForceGetMediaGroup(ctx context.Context, mediaGroupID string, chatID string, chatHistoryID uint) (*uint, error)
	GetMediaGroupTTL(ctx context.Context, mediaGroupID string, chatID string) (*time.Duration, error)
}
