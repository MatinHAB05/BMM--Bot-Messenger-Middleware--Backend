package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
)

type chatHistoryRepository struct {
	db database.Database
}

func NewChatHistoryRepository(db database.Database) repository_contract.ChatHistoryRepository {
	return &chatHistoryRepository{db: db}
}

func (r *chatHistoryRepository) Create(ctx context.Context, message *entity.ChatHistory) error {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	if err := db.GetGormDB().WithContext(ctx).Create(message).Error; err != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return nil
}

func (r *chatHistoryRepository) Upsert(ctx context.Context, message *entity.ChatHistory) error {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	if err := db.GetGormDB().WithContext(ctx).Save(message).Error; err != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return nil
}

func (r *chatHistoryRepository) FindByID(ctx context.Context, chatID, messageID uint) (*entity.ChatHistory, error) {
	var message entity.ChatHistory
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("id = ? AND chat_id = ?", messageID, chatID).First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrMessageNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &message, nil
}

func (r *chatHistoryRepository) FindByPlatformMessgeID(ctx context.Context, chatID, platformMessageID uint) (*entity.ChatHistory, error) {
	var message entity.ChatHistory
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("platform_message_id = ? AND chat_id = ?", platformMessageID, chatID).First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrMessageNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &message, nil
}

func (r *chatHistoryRepository) List(ctx context.Context, chatID uint, filter repository_contract.ChatHistoryFilter, offset, limit int) ([]entity.ChatHistory, *int64, error) {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	baseQuery := func() *gorm.DB {
		q := db.GetGormDB().WithContext(ctx).Model(&entity.ChatHistory{}).Where("chat_id = ?", chatID)
		if filter.Search != "" {
			q = q.Where("content ILIKE ?", "%"+filter.Search+"%")
		}
		if filter.MediaType != "" {
			q = q.Where("media_type = ?", filter.MediaType)
		}
		if filter.FromDate != nil {
			q = q.Where("message_timestamp >= ?", *filter.FromDate)
		}
		if filter.ToDate != nil {
			q = q.Where("message_timestamp <= ?", *filter.ToDate)
		}
		return q
	}

	var total int64
	if err := baseQuery().Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	var messages []entity.ChatHistory
	if err := baseQuery().Order("message_timestamp DESC").Offset(offset).Limit(limit).Find(&messages).Error; err != nil {
		return nil, nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return messages, &total, nil
}

func (r *chatHistoryRepository) Delete(ctx context.Context, chatID, messageID uint) error {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	res := db.GetGormDB().WithContext(ctx).Where("chat_id = ?", chatID).Delete(&entity.ChatHistory{}, "id = ?", messageID)
	if res.Error != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.ErrMessageNotFound
	}

	return nil
}

func (r *chatHistoryRepository) GetByBroadcastMsgID(ctx context.Context, broadcastMsgUUID uuid.UUID, chatID uint) (*entity.ChatHistory, error) {
	var message entity.ChatHistory
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("broadcast_uuid = ? AND chat_id = ?", broadcastMsgUUID, chatID).First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || message.BroadcastUUID == nil || message.IsBroadcast == false {
			return nil, exception.ErrMessageNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &message, nil
}
