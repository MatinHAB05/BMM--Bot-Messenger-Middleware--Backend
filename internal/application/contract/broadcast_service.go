package service_contract

import (
	"context"

	"github.com/google/uuid"
)

type BroadcastRequest struct {
	Message   string   `json:"message" binding:"required"`
	Platforms []string `json:"platforms" binding:"required"`
}

type DeleteBroadcastRequest struct {
	Platforms []string `json:"platforms" binding:"required"`
}

type UpdateBroadcastRequest struct {
	Platforms []string             `json:"platforms" binding:"required"`
	NewMessge UpdateChatHisRequest `json:"message" binding:"required"`
}

type BroadcastResult struct {
	Platform string   `json:"platform"`
	Success  bool     `json:"success"`
	Error    []string `json:"error,omitempty"`
}

type BroadcastTarget struct {
	Name      string `json:"name"`
	Platform  string `json:"platform"`
	IsPrivate bool   `json:"is_private"`
	Type      string `json:"type"`
}

type BroadcastResponse struct {
	Targets []BroadcastTarget `json:"targets"`
	Results []BroadcastResult `json:"results"`
}

// BroadcastService fans messages out to configured messenger clients concurrently.
type BroadcastService interface {
	Broadcast(ctx context.Context, companyID uint, req BroadcastRequest) (*BroadcastResponse, error)
	DeleteBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req DeleteBroadcastRequest) error
	UpdateBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req UpdateBroadcastRequest) error
}
