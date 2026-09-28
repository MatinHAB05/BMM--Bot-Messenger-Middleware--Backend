package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
)

type userRepository struct {
	db database.Database
}

func NewUserRepository(db database.Database) repository_contract.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	if err := db.GetGormDB().WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return exception.ErrUserAlreadyExists
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &user, nil
}

func (r *userRepository) FindByIDInCompany(ctx context.Context, companyID, id uint) (*entity.User, error) {
	var user entity.User
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("id = ? AND company_id = ?", id, companyID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &user, nil
}

func (r *userRepository) FindByUsernameInCompany(ctx context.Context, companyID uint, username string) (*entity.User, error) {
	var user entity.User
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("username = ? AND company_id = ?", username, companyID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &user, nil
}

// FindByIdentifier looks a user up by email or phone -- both globally
// unique -- for OTP-based passwordless login.
func (r *userRepository) FindByIdentifier(ctx context.Context, identifier string) (*entity.User, error) {
	var user entity.User
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("email = ? OR phone = ? OR username = ?", identifier, identifier, identifier).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &user, nil
}

func (r *userRepository) FindByIdentifierInCompany(ctx context.Context, companyID uint, identifier string) (*entity.User, error) {
	var user entity.User
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("(email = ? OR phone = ?) AND company_id = ?", identifier, identifier, companyID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &user, nil
}

func (r *userRepository) List(ctx context.Context, companyID uint, offset, limit int) ([]entity.User, int64, error) {
	var (
		users []entity.User
		total int64
	)

	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	if err := db.GetGormDB().WithContext(ctx).Model(&entity.User{}).Where("company_id = ?", companyID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	if err := db.GetGormDB().WithContext(ctx).Where("company_id = ?", companyID).Order("id ASC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	res := db.GetGormDB().WithContext(ctx).Save(user)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return exception.ErrUserAlreadyExists
		}
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return exception.ErrUserNotFound
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}

	if res.RowsAffected == 0 {
		return exception.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, companyID, id uint) error {
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	res := db.GetGormDB().WithContext(ctx).Where("company_id = ?", companyID).Delete(&entity.User{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) ExistsByUsername(ctx context.Context, companyID uint, username string) (*bool, error) {
	var count int64
	db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	if err := db.GetGormDB().WithContext(ctx).Model(&entity.User{}).Where("company_id = ? AND username = ?", companyID, username).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	ok := count > 0
	return &ok, nil
}
