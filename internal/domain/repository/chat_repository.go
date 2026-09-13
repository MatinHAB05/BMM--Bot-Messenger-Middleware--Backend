package repository_contract

import (
	"context"

	"messenger-backend/internal/domain/entity"
)

// ChatFilter narrows a company's chat list. Zero-valued fields are
// treated as "no filter" for that field.
type ChatFilter struct {
	Platform entity.MessengerPlatform
	ChatType entity.ChatType
	IsActive *bool
}

// ChatRepository abstracts persistence for Chat aggregates. Every method
// is scoped to a companyID, matching UserRepository's tenant-isolation
// approach.
type ChatRepository interface {
	Create(ctx context.Context, chat *entity.Chat) error
	HalfCreate(ctx context.Context, chat *entity.Chat) error
	FindByIDInCompany(ctx context.Context, companyID, id uint) (*entity.Chat, error)
	FindByID(ctx context.Context, id uint) (*entity.Chat, error)
	// FindByPlatformChatID is the upsert lookup used by the bot ingestion
	// flow to find (or learn there isn't yet) a Chat row for an incoming
	// update.
	FindByPlatformChatIDInCompany(ctx context.Context, companyID uint, platform entity.MessengerPlatform, platformChatID string) (*entity.Chat, error)
	FindByPlatformChatID(ctx context.Context, platform entity.MessengerPlatform, platformChatID string) (*entity.Chat, error)

	List(ctx context.Context, companyID uint, filter ChatFilter, offset, limit int) ([]entity.Chat, *int64, error)
	ListAll(ctx context.Context, companyID uint, platforms []string) (map[string][]entity.Chat, *int64, error)
	Update(ctx context.Context, chat *entity.Chat) error
	Delete(ctx context.Context, companyID, id uint) error
}
