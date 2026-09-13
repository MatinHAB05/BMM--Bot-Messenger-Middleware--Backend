package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChatHistory is a single ingested message belonging to a Chat. RawPayload
// preserves the full, platform-native update JSON for forensics/replay;
// Content/SenderName/MediaType are extracted onto plain columns for quick
// display and filtering (search, media_type) without touching the JSONB
// blob on every query.
type ChatHistory struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ChatID            uint           `gorm:"column:chat_id;not null;index:idx_chat_histories_chat_timestamp,priority:1" json:"chat_id"`
	PlatformMessageID int64          `gorm:"column:platform_message_id" json:"platform_message_id"`
	SenderID          string         `gorm:"column:sender_id;size:128" json:"sender_id,omitempty"`
	SenderName        string         `gorm:"column:sender_name;size:255" json:"sender_name,omitempty"`
	Content           string         `gorm:"type:text" json:"content,omitempty"`
	MediaType         string         `gorm:"column:media_type;size:32;default:text" json:"media_type"`
	RawPayload        datatypes.JSON `gorm:"column:raw_payload;type:jsonb" json:"raw_payload,omitempty"`
	MessageTimestamp  time.Time      `gorm:"column:message_timestamp;index:idx_chat_histories_chat_timestamp,priority:2" json:"message_timestamp"`
	CreatedAt         time.Time      `json:"created_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ChatHistory) TableName() string {
	return "chat_histories"
}
