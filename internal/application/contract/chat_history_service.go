package service_contract

import (
	"context"
	"encoding/json"
	"messenger-backend/internal/domain/entity"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ChatHistoryService interface {
	List(ctx context.Context, companyID, chatID uint, query ChatHistoryListQuery) (*ChatHistoryListResponse, error)
	GetByIDInCompany(ctx context.Context, companyID, chatID, messageID uint) (*ChatHistoryDetailResponse, error)
	GetByID(ctx context.Context, chatID, messageID uint) (*ChatHistoryDetailResponse, error)
	Delete(ctx context.Context, companyID, chatID, messageID uint) error
	Create(ctx context.Context, message *CreateMessageRequest) (*ChatHistoryDetailResponse, error)
	Upsert(ctx context.Context, message *CreateMessageRequest) (*ChatHistoryDetailResponse, error)

	ListByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, query ChatHistoryListQuery) (*ChatHistoryListResponse, error)
	GetByIDInCompanyByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, messageID uint) (*ChatHistoryDetailResponse, error)
	GetByIDByPlatformID(ctx context.Context, platform, platformChatID string, messageID uint) (*ChatHistoryDetailResponse, error)
	DeleteByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, messageID uint) error
}

type CreateMessageRequest struct {
	ChatID            uint           ` json:"chat_id"`
	PlatformMessageID int64          ` json:"platform_message_id"`
	SenderID          string         ` json:"sender_id,omitempty"`
	SenderName        string         ` json:"sender_name,omitempty"`
	Content           string         ` json:"content,omitempty"`
	MediaType         string         ` json:"media_type"`
	RawPayload        datatypes.JSON ` json:"raw_payload,omitempty"`
	MessageTimestamp  time.Time      ` json:"message_timestamp"`
	HasAttachments    bool           ` json:"has_attachments"`
}

type UpdateChatHisRequest struct {
	Content string ` json:"content,omitempty"`
}

// ChatHistoryListQuery carries GET /api/v1/chats/:id/history's pagination
// and filters.
type ChatHistoryListQuery struct {
	Search    string
	MediaType string
	FromDate  *time.Time
	ToDate    *time.Time
	Page      int
	PageSize  int
}

// ChatHistoryResponse is the public representation of a single message,
// as returned in a list. RawPayload is deliberately omitted here -- it's
// only surfaced via ChatHistoryDetailResponse (the single-message GET) to
// keep list responses light.
type ChatHistoryResponse struct {
	ID                uint   `json:"id"`
	ChatID            uint   `json:"chat_id"`
	PlatformMessageID int64  `json:"platform_message_id"`
	SenderID          string `json:"sender_id,omitempty"`
	SenderName        string `json:"sender_name,omitempty"`
	Content           string `json:"content,omitempty"`
	MediaType         string `json:"media_type"`
	MessageTimestamp  string `json:"message_timestamp"`
	CreatedAt         string `json:"created_at"`

	IsBroadcast    bool       `json:"is_broadcast"`
	BroadcastUUID  *uuid.UUID `json:"broadcast_uuid"`
	HasAttachments bool       `json:"has_attachments"`
}

// ChatHistoryDetailResponse is returned by GET .../history/:message_id.
type ChatHistoryDetailResponse struct {
	ChatHistoryResponse
	RawPayload json.RawMessage `json:"raw_payload,omitempty"`
}

// ChatHistoryListResponse is returned by GET /api/v1/chats/:id/history.
type ChatHistoryListResponse struct {
	Messages []ChatHistoryResponse `json:"messages"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// IngestedUpdate is the platform-neutral shape the bot listeners (via
// ChatService.IngestUpdate) use to hand off one incoming message. Building
// this struct is the ONLY thing pkg/messenger/{telegram,bale} do with an
// update -- everything past that point (finding/creating the Chat row,
// writing the ChatHistory row) is business logic that lives in the
// service layer, not the listener.
type IngestedUpdate struct {
	Platform          string    `json:"platform"`
	PlatformChatID    string    `json:"platform_chat_id"`
	ChatTitle         string    `json:"chat_title,omitempty"`
	ChatUsername      string    `json:"chat_username,omitempty"`
	ChatType          string    `json:"chat_type"`
	IsPrivateChat     bool      `json:"is_private_chat"`
	PlatformMessageID int64     `json:"platform_message_id"`
	SenderID          string    `json:"sender_id,omitempty"`
	SenderName        string    `json:"sender_name,omitempty"`
	Content           string    `json:"content,omitempty"`
	MediaType         string    `json:"media_type"`
	RawPayload        []byte    `json:"raw_payload,omitempty"`
	MessageTimestamp  time.Time `json:"message_timestamp"`
	HasAttachments    bool      `json:"has_attachments"`
}

// ToChatHistoryResponse maps a persisted ChatHistory row into the
// list-safe API representation (no RawPayload -- see
// ToChatHistoryDetailResponse for that).
func ToChatHistoryResponse(message *entity.ChatHistory) ChatHistoryResponse {
	return ChatHistoryResponse{
		ID:                message.ID,
		ChatID:            message.ChatID,
		PlatformMessageID: message.PlatformMessageID,
		SenderID:          message.SenderID,
		SenderName:        message.SenderName,
		Content:           message.Content,
		MediaType:         message.MediaType,
		MessageTimestamp:  message.MessageTimestamp.Format(timeLayout),
		CreatedAt:         message.CreatedAt.Format(timeLayout),
		IsBroadcast:       message.IsBroadcast,
		BroadcastUUID:     message.BroadcastUUID,
		HasAttachments:    message.HasAttachments,
	}
}

// ToChatHistoryDetailResponse additionally includes RawPayload, for the
// single-message GET endpoint only.
func ToChatHistoryDetailResponse(message *entity.ChatHistory) ChatHistoryDetailResponse {
	return ChatHistoryDetailResponse{
		ChatHistoryResponse: ToChatHistoryResponse(message),
		RawPayload:          json.RawMessage(message.RawPayload),
	}
}
