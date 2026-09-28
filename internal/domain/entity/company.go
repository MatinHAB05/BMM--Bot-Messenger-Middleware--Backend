package entity

import (
	"time"

	"gorm.io/gorm"
)

// Company is the tenant boundary for the whole system: every User, Chat,
// and ChatHistory row belongs to exactly one Company, and Casbin
// authorization is scoped per-company (domain == CompanyID) rather than
// having any cross-company superuser.
type Company struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Description string         `json:"description"`
	Code        string         `gorm:"column:code;size:64;not null;uniqueIndex" json:"code"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Company) TableName() string {
	return "companies"
}
