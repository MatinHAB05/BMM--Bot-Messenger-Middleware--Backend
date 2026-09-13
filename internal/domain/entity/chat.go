package entity

import (
	"time"

	"gorm.io/gorm"
)

// MessengerPlatform enumerates the messenger engines the system
// integrates with.
type MessengerPlatform string

const (
	PlatformTelegram MessengerPlatform = "telegram"
	PlatformBale     MessengerPlatform = "bale"
)

// ChatType mirrors the native chat/channel/group distinction Telegram and
// Bale both expose on their chat objects.
type ChatType string

const (
	ChatTypeChannel    ChatType = "channel"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSupergroup ChatType = "supergroup"
)

// Chat is a company-owned messenger chat/channel/group, identified by its
// native platform chat id. It replaces the earlier, tenant-unaware
// Channel entity. A Company can own multiple Chats across both
// platforms; (CompanyID, Platform, PlatformChatID) is unique so the same
// native chat can't be registered twice under one company (but the same
// native chat CAN be registered independently by two different
// companies, e.g. a bot shared across tenants).
type Chat struct {
	ID             uint              `gorm:"primaryKey" json:"id"`
	CompanyID      *uint             `gorm:"column:company_id;null;uniqueIndex:idx_chats_company_platform_chatid" json:"company_id"`
	Platform       MessengerPlatform `gorm:"column:platform;size:16;not null;uniqueIndex:idx_chats_company_platform_chatid" json:"platform"`
	PlatformChatID string            `gorm:"column:platform_chat_id;size:128;not null;uniqueIndex:idx_chats_company_platform_chatid" json:"platform_chat_id"`
	Title          string            `gorm:"size:255" json:"title"`
	Username       string            `gorm:"size:128" json:"username,omitempty"`
	ChatType       ChatType          `gorm:"column:chat_type;size:32" json:"chat_type"`
	IsPrivate      bool              `gorm:"column:is_private;default:false" json:"is_private"`
	IsActive       bool              `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	DeletedAt      gorm.DeletedAt    `gorm:"index" json:"-"`
}

func (Chat) TableName() string {
	return "chats"
}
