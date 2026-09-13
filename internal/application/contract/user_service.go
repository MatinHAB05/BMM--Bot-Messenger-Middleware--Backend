package service_contract

import (
	"context"
	"messenger-backend/internal/domain/entity"
)

// UserService manages user identity CRUD operations and Casbin RBAC role sync.
type UserService interface {
	Me(ctx context.Context, companyID uint, userID string) (*UserResponse, error)
	List(ctx context.Context, companyID uint, page, pageSize int) (*UserListResponse, error)
	// Create(ctx context.Context, companyID uint, req CreateUserRequest) (*UserResponse, error) ?!??!
	UpdateRoles(context.Context, uint, string, []string) (*UserResponse, error)
	UpdateEmail(ctx context.Context, companyID uint, userID uint, req UpdateUserEmailRequest) (*UserResponse, error)
	UpdatePhone(ctx context.Context, companyID uint, userID uint, req UpdateUserPhoneRequest) (*UserResponse, error)
	UpdateUsername(ctx context.Context, companyID uint, userID uint, req UpdateUserUsernameRequest) (*UserResponse, error)
	Delete(ctx context.Context, companyID uint, userID string) error
	GetByID(ctx context.Context, companyID uint, userID string) (*UserResponse, error)
	Update(ctx context.Context, companyID uint, userID string, req UpdateUserRequest) (*UserResponse, error)
}

type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=64"`
	Password string   `json:"password" binding:"omitempty,min=6"`
	Phone    string   `json:"phone"`
	Email    string   `json:"email" binding:"required,email"`
	Roles    []string `json:"roles"`
}

type CreateMeUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"omitempty,min=6"`
	Phone    string `json:"phone" binding:"omitempty"`
	Email    string `json:"email" binding:"omitempty,email"`
}

type UpdateUserRolesRequest struct {
	Roles []string `json:"roles" binding:"required,min=1"`
}

type UpdateUserRequest struct {
	IsActive *bool `json:"is_active"`
}

type UpdateUserEmailRequest struct {
	Email string `json:"email" binding:"email"`
}

type UpdateUserPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type UpdateUserUsernameRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
}

type UserResponse struct {
	ID        uint     `json:"id"`
	CompanyID uint     `json:"company_id"`
	Username  string   `json:"username"`
	Phone     *string  `json:"phone,omitempty"`
	Email     *string  `json:"email"`
	IsActive  bool     `json:"is_active"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type UserListResponse struct {
	Users    []UserResponse `json:"users"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

func ToUserResponse(user *entity.User, roles []string) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		IsActive:  user.IsActive,
		Roles:     roles,
		CreatedAt: user.CreatedAt.Format(timeLayout),
		UpdatedAt: user.UpdatedAt.Format(timeLayout),
		CompanyID: user.CompanyID,
		Phone:     user.Phone,
	}
}
