package repository_contract

import (
	"context"

	"messenger-backend/internal/domain/entity"
)

// CompanyRepository abstracts persistence for Company aggregates -- the
// tenant boundary for the whole system.
type CompanyRepository interface {
	Create(ctx context.Context, company *entity.Company) error
	FindByID(ctx context.Context, id uint) (*entity.Company, error)
	FindByCode(ctx context.Context, code string) (*entity.Company, error)

	Update(ctx context.Context, company *entity.Company) error
	Delete(ctx context.Context, id uint) error
}
