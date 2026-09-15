package service_contract

import (
	"context"
	"time"
)

type SentBaleMsgService interface {
	SetSentBaleMsg(ctx context.Context, baleChatID string, content string) error
	GetSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error)
	GetAndRemoveSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error)
	HasSentBaleMsg(ctx context.Context, chatID string, content string) (*bool, error)
	RemoveSentBaleMsg(ctx context.Context, baleChatID string, content string) error
	ForceGetSentBaleMsg(ctx context.Context, baleChatID string, content string) (*bool, error)
	GetSentBaleMsgTTL(ctx context.Context, baleChatID string, content string) (time.Duration, error)
}
