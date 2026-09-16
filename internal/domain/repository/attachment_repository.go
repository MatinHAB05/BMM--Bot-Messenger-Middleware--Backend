package repository_contract

import (
	"context"

	"messenger-backend/internal/domain/entity"
)

type AttachmentRepository interface {
	// Create
	Create(ctx context.Context, attachment *entity.Attachment) error
	CreateBatch(ctx context.Context, attachments []entity.Attachment) error

	// Read & query
	GetByID(ctx context.Context, id uint) (*entity.Attachment, error)
	GetByIDs(ctx context.Context, ids []uint) ([]entity.Attachment, error)
	GetByPlatformFileID(ctx context.Context, platformFileID string) (*entity.Attachment, error)
	GetByThumbnailPlatformFileID(ctx context.Context, thumbID string) ([]entity.Attachment, error)
	GetByChatHistoryID(ctx context.Context, chatHistoryID uint) ([]entity.Attachment, error)
	GetByChatHistoryIDs(ctx context.Context, chatHistoryIDs []uint) (map[uint][]entity.Attachment, error)
	List(ctx context.Context, limit, offset int, sort string) ([]entity.Attachment, int64, error)

	// Update
	Update(ctx context.Context, attachment *entity.Attachment) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	UpdateFieldsBatch(ctx context.Context, ids []uint, fields map[string]interface{}) error

	// Delete & restore (soft)
	DeleteByID(ctx context.Context, id uint) error
	DeleteByIDs(ctx context.Context, ids []uint) error
	DeleteByChatHistoryID(ctx context.Context, chatHistoryID uint) error
	RestoreByID(ctx context.Context, id uint) error

	// Hard delete (permanent)
	HardDeleteByID(ctx context.Context, id uint) error
	HardDeleteByChatHistoryID(ctx context.Context, chatHistoryID uint) error
}
