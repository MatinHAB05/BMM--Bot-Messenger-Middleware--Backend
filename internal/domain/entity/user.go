package entity

import (
	"time"

	"gorm.io/gorm"
)

// User represents an authenticated principal of the system. Every user
// belongs to exactly one Company (CompanyID) -- there is no global
// superuser, so all administrative privileges are bounded to that
// company via Casbin's domain-scoped RBAC (domain == CompanyID). Roles
// are not stored on this struct directly -- they live as Casbin grouping
// policies (see internal/infrastructure/repository/rbac) keyed by the
// User's ID within their company's domain.
//
// Username is scoped per-company (the same handle can exist in two
// different companies). Email and Phone are kept globally unique instead:
// they double as OTP identifiers (see OTPRepository), and an OTP-based
// login has no company context to disambiguate against until *after* the
// matching user is found, so the lookup has to be unambiguous.
type User struct {
	ID              uint `gorm:"primaryKey" json:"id"`
	Firstname       string
	Lastname        string
	CompanyID       uint           `gorm:"column:company_id;not null;index" json:"company_id"`
	Username        string         `gorm:"size:64;not null;uniqueIndex:idx_users_company_username" json:"username"`
	Email           *string        `gorm:"size:128;uniqueIndex" json:"email,omitempty"`
	Phone           *string        `gorm:"size:32;uniqueIndex" json:"phone,omitempty"`
	PasswordHash    string         `gorm:"column:password_hash;size:255" json:"-"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	IsVerifiedPhone bool           `gorm:"default:false" json:"is_verified_phone"`
	IsVerifiedEmail bool           `gorm:"default:false" json:"is_verified_email"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
