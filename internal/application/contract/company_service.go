package service_contract

import (
	"context"
	"messenger-backend/internal/domain/entity"
)

type CompanyService interface {
	Create(ctx context.Context, req CreateCompanyRequest) (*CompanyResponse, error)
	Me(ctx context.Context, companyID uint) (*CompanyResponse, error)
	Update(ctx context.Context, callerCompanyID, targetCompanyID uint, req UpdateCompanyRequest) (*CompanyResponse, error)
	Delete(ctx context.Context, callerCompanyID, targetCompanyID uint) error
	SendRegistionrWithOTP(ctx context.Context, companyID uint, req SendRegistionrWithOTPRequest) (*SendRegistionrWithOTPResponse, error)
}

type SendRegistionrWithOTPRequest struct {
	Roles []string `json:"roles" binding:"required"`
}

type SendRegistionrWithOTPResponse struct {
	Message         string `json:"message"`
	ExpiresInSecond int    `json:"expires_in_seconds"`
	Code            string `json:"code,omitempty"`
}

type CreateCompanyRequest struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=128"`
	Code        string `json:"code" binding:"omitempty,min=2,max=64"`
	Description string `json:"description" binding:"omitempty"`
}

// UpdateCompanyRequest is the payload for PUT /api/v1/companies/:id. All
// fields are optional -- only those provided are changed.
type UpdateCompanyRequest struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=128"`
	Code        string `json:"code" binding:"omitempty,min=2,max=64"`
	IsActive    *bool  `json:"is_active"`
	Description string `json:"description" binding:"omitempty"`
}

// CompanyResponse is the public representation of a Company.
type CompanyResponse struct {
	ID          uint   `json:"id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ToCompanyResponse maps a persisted Company into the API-facing
// representation.
func ToCompanyResponse(company *entity.Company) CompanyResponse {
	return CompanyResponse{
		ID:          company.ID,
		Name:        company.Name,
		Code:        company.Code,
		IsActive:    company.IsActive,
		CreatedAt:   company.CreatedAt.Format(timeLayout),
		Description: company.Description,
		UpdatedAt:   company.UpdatedAt.Format(timeLayout),
	}
}
