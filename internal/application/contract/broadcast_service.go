package service_contract

import "context"

type BroadcastRequest struct {
	Message   string   `json:"message" binding:"required"`
	Platforms []string `json:"platforms" binding:"required"`
}

type BroadcastResult struct {
	Platform string `json:"platform"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

type BroadcastResponse struct {
	Targets []struct {
		Name      string
		Platform  string
		IsPrivate bool
		Type      string
	} `json:"targets"`
	Results []BroadcastResult `json:"results"`
}

// BroadcastService fans messages out to configured messenger clients concurrently.
type BroadcastService interface {
	Broadcast(ctx context.Context, companyID uint, req BroadcastRequest) (*BroadcastResponse, error)
}
