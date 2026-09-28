package repository_contract

import (
	"context"

	"messenger-backend/internal/domain/entity"
)

// UserRepository abstracts persistence for User aggregates. Every method
// that reads or lists users is scoped to a companyID -- there is no
// cross-company lookup anywhere in this interface, which is what makes
// tenant isolation structural rather than something callers have to
// remember to apply.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByIDInCompany(ctx context.Context, companyID, id uint) (*entity.User, error)
	// known from a token -- callers must independently confirm the
	// returned user's IsActive/CompanyID are acceptable for the request
	// being made once other checks in the auth flow may require it.
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByUsernameInCompany(ctx context.Context, companyID uint, username string) (*entity.User, error)
	FindByIdentifier(ctx context.Context, identifier string) (*entity.User, error)
	FindByIdentifierInCompany(ctx context.Context, companyID uint, identifier string) (*entity.User, error)
	List(ctx context.Context, companyID uint, offset, limit int) ([]entity.User, int64, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, companyID, id uint) error
	ExistsByUsername(ctx context.Context, companyID uint, username string) (*bool, error)
}
