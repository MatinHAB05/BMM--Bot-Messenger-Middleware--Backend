package service_contract

import (
	"context"
	"messenger-backend/internal/domain/entity"
)

type ChatService interface {
	List(ctx context.Context, companyID uint, query ChatListQuery) (*ChatListResponse, error)
	GetByIDInCompany(ctx context.Context, companyID, chatID uint) (*ChatResponse, error)
	GetByID(ctx context.Context, chatID uint) (*ChatResponse, error)
	Create(ctx context.Context, companyID uint, req CreateChatRequest) (*ChatResponse, error)
	HalfCreate(ctx context.Context, req CreateChatRequest) (*ChatResponse, error)
	Update(ctx context.Context, companyID, chatID uint, req UpdateChatRequest) (*ChatResponse, error)
	UpdateCompanyID(ctx context.Context, companyID, chatID uint) (*ChatResponse, error)
	Delete(ctx context.Context, companyID, chatID uint) error
	IngestUpdate(ctx context.Context, companyID uint, update IngestedUpdate) error

	GetByPlatformIDInCompany(ctx context.Context, companyID uint, platform, platformChatID string) (*ChatResponse, error)
	GetByPlatformID(ctx context.Context, platform, platformChatID string) (*ChatResponse, error)
	UpdateByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, req UpdateChatRequest) (*ChatResponse, error)
	DeleteByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string) error

	SendOTP(ctx context.Context, companyID uint) (*SendLinkChatOTPResponse, error)
	FeatChatWithOTP(ctx context.Context, code, platformChatID, platform, otpchatType string) (*ChatResponse, error)
}

type SendLinkChatOTPResponse struct {
	Message         string `json:"message"`
	ExpiresInSecond int    `json:"expires_in_seconds"`
	Code            string `json:"code,omitempty"`
}

// CreateChatRequest is the payload for POST /api/v1/chats -- manually
// registering/linking an existing platform chat to the caller's company.
type CreateChatRequest struct {
	Platform       string `json:"platform" binding:"required,oneof=telegram bale"`
	PlatformChatID string `json:"platform_chat_id" binding:"required"`
	Title          string `json:"title"`
	Username       string `json:"username"`
	ChatType       string `json:"chat_type" binding:"required,oneof=channel group supergroup"`
	IsPrivate      bool   `json:"is_private"`
}

// UpdateChatRequest is the payload for PUT /api/v1/chats/:id.
type UpdateChatRequest struct {
	Title    string `json:"title"`
	Username string `json:"username"`
	IsActive *bool  `json:"is_active"`
}

// ChatListQuery carries GET /api/v1/chats' pagination and filters.
type ChatListQuery struct {
	Platform string `json:"platform,omitempty" form:"platform"`
	ChatType string `json:"chat_type,omitempty" form:"chat_type"`
	IsActive *bool  `json:"is_active,omitempty" form:"is_active"`
	Page     int    `json:"page,omitempty" form:"page"`
	PageSize int    `json:"page_size,omitempty" form:"page_size"`
}

// ChatResponse is the public representation of a Chat.
type ChatResponse struct {
	ID             uint   `json:"id"`
	CompanyID      *uint  `json:"company_id"`
	Platform       string `json:"platform"`
	PlatformChatID string `json:"platform_chat_id"`
	Title          string `json:"title"`
	Username       string `json:"username,omitempty"`
	ChatType       string `json:"chat_type"`
	IsPrivate      bool   `json:"is_private"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
}

// ChatListResponse is returned by GET /api/v1/chats.
type ChatListResponse struct {
	Chats    []ChatResponse `json:"chats"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// ToChatResponse maps a persisted Chat into the API-facing representation.
func ToChatResponse(chat *entity.Chat) ChatResponse {
	return ChatResponse{
		ID:             chat.ID,
		CompanyID:      chat.CompanyID,
		Platform:       string(chat.Platform),
		PlatformChatID: chat.PlatformChatID,
		Title:          chat.Title,
		Username:       chat.Username,
		ChatType:       string(chat.ChatType),
		IsPrivate:      chat.IsPrivate,
		IsActive:       chat.IsActive,
		CreatedAt:      chat.CreatedAt.Format(timeLayout),
	}
}
