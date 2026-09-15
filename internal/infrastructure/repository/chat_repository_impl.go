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

type chatRepository struct {
	db database.Database
}

func NewChatRepository(db database.Database) repository_contract.ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) Create(ctx context.Context, chat *entity.Chat) error {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	if err := db.GetGormDB().WithContext(ctx).Create(chat).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return exception.ErrChatAlreadyExists
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return nil
}

// companyID => nil
func (r *chatRepository) HalfCreate(ctx context.Context, chat *entity.Chat) error {
	chat.CompanyID = nil

	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	if err := db.GetGormDB().WithContext(ctx).Create(chat).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return exception.ErrChatAlreadyExists
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return nil
}

func (r *chatRepository) FindByIDInCompany(ctx context.Context, companyID, id uint) (*entity.Chat, error) {
	var chat entity.Chat

	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("id = ? AND company_id = ?", id, companyID).First(&chat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &chat, nil
}

func (r *chatRepository) FindByPlatformChatIDInCompany(ctx context.Context, companyID uint, platform entity.MessengerPlatform, platformChatID string) (*entity.Chat, error) {
	var chat entity.Chat

	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("company_id = ? AND platform = ? AND platform_chat_id = ?", companyID, platform, platformChatID).First(&chat).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &chat, nil
}

func (r *chatRepository) FindByID(ctx context.Context, id uint) (*entity.Chat, error) {
	var chat entity.Chat

	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("id = ?", id).First(&chat).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &chat, nil
}

func (r *chatRepository) FindByPlatformChatID(ctx context.Context, platform entity.MessengerPlatform, platformChatID string) (*entity.Chat, error) {
	var chat entity.Chat

	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	err := db.GetGormDB().WithContext(ctx).
		Where("platform = ? AND platform_chat_id = ?", platform, platformChatID).
		First(&chat).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &chat, nil
}

func (r *chatRepository) List(ctx context.Context, companyID uint, filter repository_contract.ChatFilter, offset, limit int) ([]entity.Chat, *int64, error) {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	baseQuery := func() *gorm.DB {
		q := db.GetGormDB().WithContext(ctx).Model(&entity.Chat{}).Where("company_id = ?", companyID)
		if filter.Platform != "" {
			q = q.Where("platform = ?", filter.Platform)
		}
		if filter.ChatType != "" {
			q = q.Where("chat_type = ?", filter.ChatType)
		}
		if filter.IsActive != nil {
			q = q.Where("is_active = ?", *filter.IsActive)
		}
		return q
	}

	var total int64
	if err := baseQuery().Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	var chats []entity.Chat
	if err := baseQuery().Order("id DESC").Offset(offset).Limit(limit).Find(&chats).Error; err != nil {
		return nil, nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return chats, &total, nil
}

func (r *chatRepository) ListAll(ctx context.Context, companyID uint, platforms []string) (map[string][]entity.Chat, *int64, error) {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	var chats []entity.Chat

	query := db.GetGormDB().WithContext(ctx).Where("company_id = ? AND is_active = ?", companyID, true)

	if len(platforms) > 0 {
		query = query.Where("platform IN ?", platforms)
	}

	if err := query.Find(&chats).Error; err != nil {
		return nil, nil, err
	}

	total := int64(len(chats))
	result := make(map[string][]entity.Chat)

	for _, chat := range chats {
		platformKey := string(chat.Platform)
		result[platformKey] = append(result[platformKey], chat)
	}

	return result, &total, nil
}

func (r *chatRepository) Update(ctx context.Context, chat *entity.Chat) error {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	res := db.GetGormDB().WithContext(ctx).Save(chat)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return exception.ErrChatAlreadyExists
		}
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return exception.ErrChatNotFound
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}

	if res.RowsAffected == 0 {
		return exception.ErrChatNotFound
	}

	return nil
}

func (r *chatRepository) Delete(ctx context.Context, companyID, id uint) error {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	res := db.GetGormDB().WithContext(ctx).Where("company_id = ?", companyID).Delete(&entity.Chat{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.ErrChatNotFound
	}

	return nil
}

func (r *chatRepository) GetAllChatsContainsBroadcastMsgUUID(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, platforms []string) (map[string][]entity.Chat, *int64, error) {
	// db := database.ExtractTrxOrDB(ctx, r.db)
	db := r.db

	query := db.GetGormDB().WithContext(ctx).
		Model(&entity.Chat{}).
		Joins("JOIN chat_histories ON chat_histories.chat_id = chats.id").
		Where("chat_histories.broadcast_uuid = ?", broadcastMsgUUID).
		Where("chat_histories.is_broadcast = ?", true).
		Where("chats.company_id = ?", companyID)

	if len(platforms) > 0 {
		query = query.Where("chats.platform IN ?", platforms)
	}

	var chats []entity.Chat
	if err := query.Find(&chats).Error; err != nil {
		return nil, nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	total := int64(len(chats))

	platformChats := make(map[string][]entity.Chat, len(platforms))
	for _, chat := range chats {
		platform := string(chat.Platform)
		platformChats[platform] = append(platformChats[platform], chat)
	}

	return platformChats, &total, nil
}
