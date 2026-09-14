package service_contract

import (
	"context"
)

type ChannelPendingService interface {
	SetPendingChannel(ctx context.Context, userTelegramID string, companyID uint) error
	GetPendingChannel(ctx context.Context, userTelegramID string) (*uint, error)
	GetAndRemovePendingChannel(ctx context.Context, userTelegramID string) (*uint, error)
	HasPendingChannel(ctx context.Context, userTelegramID string) (*bool, error)
	RemovePendingChannel(ctx context.Context, userTelegramID string) error
}
