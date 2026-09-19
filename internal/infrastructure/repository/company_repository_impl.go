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

type companyRepository struct {
	db database.Database
}

func NewCompanyRepository(db database.Database) repository_contract.CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) Create(ctx context.Context, company *entity.Company) error {
	 db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	if err := db.GetGormDB().WithContext(ctx).Create(company).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return exception.ErrCompanyAlreadyExists
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return nil
}

func (r *companyRepository) FindByID(ctx context.Context, id uint) (*entity.Company, error) {
	var company entity.Company
	 db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("id = ?", id).First(&company).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrCompanyNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &company, nil
}

func (r *companyRepository) FindByCode(ctx context.Context, code string) (*entity.Company, error) {
	var company entity.Company
	 db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	err := db.GetGormDB().WithContext(ctx).Where("code = ?", code).First(&company).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, exception.ErrCompanyNotFound
		}
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	return &company, nil
}

func (r *companyRepository) Update(ctx context.Context, company *entity.Company) error {
	 db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	res := db.GetGormDB().WithContext(ctx).Save(company)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return exception.ErrCompanyAlreadyExists
		}
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return exception.ErrCompanyNotFound
		}
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}

	if res.RowsAffected == 0 {
		return exception.ErrCompanyNotFound
	}

	return nil
}

func (r *companyRepository) Delete(ctx context.Context, id uint) error {
	 db := database.ExtractTrxOrDB(ctx, r.db)
	// db := r.db

	res := db.GetGormDB().WithContext(ctx).Delete(&entity.Company{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, res.Error)
	}
	if res.RowsAffected == 0 {
		return exception.ErrCompanyNotFound
	}

	return nil
}
