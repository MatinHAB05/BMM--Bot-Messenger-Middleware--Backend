package repository_contract

import (
	"context"

	"messenger-backend/internal/domain/entity"
)

// AttachmentRepository abstracts persistence for Attachment rows.
//
// No method here accepts a *gorm.DB or transaction handle. Every
// implementation resolves its own database handle via
// database.ExtractTrxOrDB(ctx, r.db) internally, so a caller joins an
// ongoing transaction purely by calling through a context that
// database.TrxManager.WithTransaction has already injected one into.
//
// Not-found reads (GetByID, GetByPlatformFileID) return
// exception.ErrAttachmentNotFound directly, matching this codebase's
// other repositories -- callers check with errors.Is.
type AttachmentRepository interface {
	// --- Create ---
	Create(ctx context.Context, attachment *entity.Attachment) error
	CreateBatch(ctx context.Context, attachments []entity.Attachment) error

	// --- Read & query ---
	GetByID(ctx context.Context, id uint) (*entity.Attachment, error)
	GetByIDs(ctx context.Context, ids []uint) ([]entity.Attachment, error)
	GetByPlatformFileID(ctx context.Context, platformFileID string) (*entity.Attachment, error)
	GetByThumbnailPlatformFileID(ctx context.Context, thumbID string) ([]entity.Attachment, error)
	GetByChatHistoryID(ctx context.Context, chatHistoryID uint) ([]entity.Attachment, error)
	GetByChatHistoryIDs(ctx context.Context, chatHistoryIDs []uint) (map[uint][]entity.Attachment, error)
	// List supports an optional fileType filter (pass "" for none) so
	// AttachmentService.ListAttachments' file_type query param can be
	// applied at the DB layer, before LIMIT/OFFSET -- filtering after
	// the fact in the service would silently break pagination counts.
	List(ctx context.Context, limit, offset int, sort, fileType string) ([]entity.Attachment, int64, error)

	// --- Update ---
	Update(ctx context.Context, attachment *entity.Attachment) error
	UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error
	UpdateFieldsBatch(ctx context.Context, ids []uint, fields map[string]interface{}) error

	// --- Delete & restore (soft, via entity.Attachment's gorm.DeletedAt) ---
	DeleteByID(ctx context.Context, id uint) error
	DeleteByIDs(ctx context.Context, ids []uint) error
	DeleteByChatHistoryID(ctx context.Context, chatHistoryID uint) error
	RestoreByID(ctx context.Context, id uint) error
}
