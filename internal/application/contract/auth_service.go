package service_contract

import "context"

// Auth DTOs
type TokenPairResponse struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
}

type DefaultLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type EmailOption struct {
	Email string `json:"email" binding:"required,email"`
}

type PhoneOption struct {
	Phone string `json:"phone" binding:"required"`
}

type LoginRequest struct {
	Default     *DefaultLogin `json:"default" binding:"omitempty"`
	EmailOption *EmailOption  `json:"email_option" binding:"omitempty"`
	PhoneOption *PhoneOption  `json:"phone_option" binding:"omitempty"`
}

type RegisterWithCompanyRequest struct {
	Company CreateCompanyRequest `json:"company" binding:"required"`
	Me      CreateMeUserRequest  `json:"me" binding:"required"`
}

type VerifyRegisterWithOTPRequest struct {
	Me   CreateMeUserRequest `json:"me" biding:"true"`
	Code string              `json:"code" binding:"required"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// SendOTPRequest is the payload for POST /api/v1/auth/otp/send.
type SendAuthOTPRequest struct {
	Identifier string `json:"identifier" binding:"required"` // phone - email - company-id
	Type       string `json:"type" binding:"required,oneof=email phone link"`
}

// SendOTPResponse acknowledges an OTP dispatch. Code is only populated
// outside of production -- see AuthService.SendOTP -- since no real
// email/SMS delivery provider is wired into this backend yet; treat that
// integration as the next step for production use.
type SendAuthOTPResponse struct {
	Message         string `json:"message"`
	ExpiresInSecond int    `json:"expires_in_seconds"`
	Code            string `json:"code,omitempty"`
}

// VerifyOTPRequest is the payload for POST /api/v1/auth/otp/verify.
type VerifyAuthOTPRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=email phone link"`
	Code       string `json:"code" binding:"required"`
}

// AuthService owns token lifecycle management, PASETO issuance, and session revocation.
type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*TokenPairResponse, error)
	RegisterMeWithCompany(ctx context.Context, req RegisterWithCompanyRequest) (*UserResponse, *CompanyResponse, error)
	RegisterWithOTP(ctx context.Context, req VerifyRegisterWithOTPRequest) (*UserResponse, *CompanyResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPairResponse, error)
	Logout(ctx context.Context, accessToken string) error

	SendOTP(ctx context.Context, identifier, otpType string) (*SendAuthOTPResponse, error)
	VerifyOTP(ctx context.Context, identifier, otpType, code string) (*bool, error)
}
