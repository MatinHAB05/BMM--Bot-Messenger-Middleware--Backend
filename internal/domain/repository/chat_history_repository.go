package repository_contract

import (
	"context"
	"time"

	"messenger-backend/internal/domain/entity"

	"github.com/google/uuid"
)

// ChatHistoryFilter narrows a paginated history query. Zero-valued fields
// are treated as "no filter" for that field.
type ChatHistoryFilter struct {
	Search    string // matched against Content, case-insensitive substring
	MediaType string
	FromDate  *time.Time
	ToDate    *time.Time
}

// ChatHistoryRepository abstracts persistence for ChatHistory (message)
// rows. Reads take chatID (already confirmed to belong to the caller's
// company by the service layer) rather than companyID directly, since a
// message's tenant is implied by its parent Chat.
type ChatHistoryRepository interface {
	Create(ctx context.Context, message *entity.ChatHistory) error
	Upsert(ctx context.Context, message *entity.ChatHistory) error
	FindByID(ctx context.Context, chatID, messageID uint) (*entity.ChatHistory, error)
	FindByPlatformMessgeID(ctx context.Context, chatID, platformMessageID uint) (*entity.ChatHistory, error)
	List(ctx context.Context, chatID uint, filter ChatHistoryFilter, offset, limit int) ([]entity.ChatHistory, *int64, error)
	Delete(ctx context.Context, chatID, messageID uint) error
	GetByBroadcastMsgID(ctx context.Context, broadcastMsgUUID uuid.UUID, chatID uint) (*entity.ChatHistory, error)
	GetAllByBroadcastMsgID(ctx context.Context, broadcastMsgUUID uuid.UUID, chatID uint) ([]entity.ChatHistory, error)
}
