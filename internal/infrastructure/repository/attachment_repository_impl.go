package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
)

type attachmentRepository struct {
	db database.Database
}

func NewAttachmentRepository(db database.Database) repository_contract.AttachmentRepository {
	return &attachmentRepository{db: db}
}

// --- Create ---

func (r *attachmentRepository) Create(ctx context.Context, attachment *entity.Attachment) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Create(attachment).Error
}

func (r *attachmentRepository) CreateBatch(ctx context.Context, attachments []entity.Attachment) error {
	if len(attachments) == 0 {
		return nil
	}
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Create(&attachments).Error
}

// --- Read & query ---

func (r *attachmentRepository) GetByID(ctx context.Context, id uint) (*entity.Attachment, error) {
	db := database.ExtractTrxOrDB(ctx, r.db)

	var attachment entity.Attachment
	err := db.GetGormDB().WithContext(ctx).First(&attachment, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exception.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (r *attachmentRepository) GetByIDs(ctx context.Context, ids []uint) ([]entity.Attachment, error) {
	if len(ids) == 0 {
		return []entity.Attachment{}, nil
	}
	db := database.ExtractTrxOrDB(ctx, r.db)

	var attachments []entity.Attachment
	if err := db.GetGormDB().WithContext(ctx).Where("id IN ?", ids).Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *attachmentRepository) GetByPlatformFileID(ctx context.Context, platformFileID string) (*entity.Attachment, error) {
	db := database.ExtractTrxOrDB(ctx, r.db)

	var attachment entity.Attachment
	err := db.GetGormDB().WithContext(ctx).First(&attachment, "platform_file_id = ?", platformFileID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exception.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (r *attachmentRepository) GetByThumbnailPlatformFileID(ctx context.Context, thumbID string) ([]entity.Attachment, error) {
	db := database.ExtractTrxOrDB(ctx, r.db)

	var attachments []entity.Attachment
	if err := db.GetGormDB().WithContext(ctx).Where("thumbnail_platform_file_id = ?", thumbID).Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *attachmentRepository) GetByChatHistoryID(ctx context.Context, chatHistoryID uint) ([]entity.Attachment, error) {
	db := database.ExtractTrxOrDB(ctx, r.db)

	var attachments []entity.Attachment
	if err := db.GetGormDB().WithContext(ctx).
		Where("chat_history_id = ?", chatHistoryID).
		Order("id ASC").
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *attachmentRepository) GetByChatHistoryIDs(ctx context.Context, chatHistoryIDs []uint) (map[uint][]entity.Attachment, error) {
	result := make(map[uint][]entity.Attachment, len(chatHistoryIDs))
	if len(chatHistoryIDs) == 0 {
		return result, nil
	}
	db := database.ExtractTrxOrDB(ctx, r.db)

	var attachments []entity.Attachment
	if err := db.GetGormDB().WithContext(ctx).
		Where("chat_history_id IN ?", chatHistoryIDs).
		Order("id ASC").
		Find(&attachments).Error; err != nil {
		return nil, err
	}

	for i := range attachments {
		key := attachments[i].ChatHistoryID
		result[key] = append(result[key], attachments[i])
	}
	return result, nil
}

func (r *attachmentRepository) List(ctx context.Context, limit, offset int, sort string) ([]entity.Attachment, int64, error) {
	db := database.ExtractTrxOrDB(ctx, r.db)
	if sort == "" {
		sort = "id DESC"
	}

	var total int64
	if err := db.GetGormDB().WithContext(ctx).Model(&entity.Attachment{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var attachments []entity.Attachment
	if err := db.GetGormDB().WithContext(ctx).
		Order(sort).
		Offset(offset).
		Limit(limit).
		Find(&attachments).Error; err != nil {
		return nil, 0, err
	}

	return attachments, total, nil
}

// --- Update ---

func (r *attachmentRepository) Update(ctx context.Context, attachment *entity.Attachment) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Save(attachment).Error
}

func (r *attachmentRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Model(&entity.Attachment{}).Where("id = ?", id).Updates(fields).Error
}

func (r *attachmentRepository) UpdateFieldsBatch(ctx context.Context, ids []uint, fields map[string]interface{}) error {
	if len(ids) == 0 || len(fields) == 0 {
		return nil
	}
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Model(&entity.Attachment{}).Where("id IN ?", ids).Updates(fields).Error
}

// --- Delete & restore (soft) ---

func (r *attachmentRepository) DeleteByID(ctx context.Context, id uint) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Delete(&entity.Attachment{}, "id = ?", id).Error
}

func (r *attachmentRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Delete(&entity.Attachment{}, "id IN ?", ids).Error
}

func (r *attachmentRepository) DeleteByChatHistoryID(ctx context.Context, chatHistoryID uint) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).Delete(&entity.Attachment{}, "chat_history_id = ?", chatHistoryID).Error
}

func (r *attachmentRepository) RestoreByID(ctx context.Context, id uint) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	return db.GetGormDB().WithContext(ctx).
		Unscoped().
		Model(&entity.Attachment{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}

// --- Hard delete (permanent) ---

func (r *attachmentRepository) HardDeleteByID(ctx context.Context, id uint) error {
	db := database.ExtractTrxOrDB(ctx, r.db)

	return db.GetGormDB().Unscoped().Delete(&entity.Attachment{}, "id = ?", id).Error
}

func (r *attachmentRepository) HardDeleteByChatHistoryID(ctx context.Context, chatHistoryID uint) error {
	db := database.ExtractTrxOrDB(ctx, r.db)

	return db.GetGormDB().Unscoped().Delete(&entity.Attachment{}, "chat_history_id = ?", chatHistoryID).Error
}
