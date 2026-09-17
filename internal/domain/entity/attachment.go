package entity

import (
	"time"

	"gorm.io/gorm"
)

// AttachmentFileType enumerates the kinds of media an Attachment can
// represent -- the same vocabulary ChatHistory.MediaType draws its
// single-file values from (see that entity's doc comment).
type AttachmentFileType string

const (
	AttachmentTypePhoto     AttachmentFileType = "photo"
	AttachmentTypeVideo     AttachmentFileType = "video"
	AttachmentTypeDocument  AttachmentFileType = "document"
	AttachmentTypeAudio     AttachmentFileType = "audio"
	AttachmentTypeVoice     AttachmentFileType = "voice"
	AttachmentTypeAnimation AttachmentFileType = "animation"
)

// Attachment is a single media file -- or its thumbnail reference -- tied
// to one ChatHistory message. Only platform-native file identifiers are
// stored here (Telegram/Bale file_id strings); there is no byte storage,
// S3/MinIO key, or local path anywhere on this entity, by design.
type Attachment struct {
	ID                      uint               `gorm:"primaryKey" json:"id"`
	ChatHistoryID           uint               `gorm:"column:chat_history_id;not null;index:idx_attachments_chat_history_id" json:"chat_history_id"`
	PlatformFileID          string             `gorm:"column:platform_file_id;size:255;not null;index:idx_attachments_platform_file_id" json:"platform_file_id"`
	FileType                AttachmentFileType `gorm:"column:file_type;size:32;not null;index:idx_attachments_file_type" json:"file_type"`
	FileName                string             `gorm:"column:file_name;size:255" json:"file_name,omitempty"`
	MimeType                string             `gorm:"column:mime_type;size:128" json:"mime_type,omitempty"`
	FileSize                int64              `gorm:"column:file_size" json:"file_size,omitempty"`
	ThumbnailPlatformFileID string             `gorm:"column:thumbnail_platform_file_id;size:255;index:idx_attachments_thumb_file_id" json:"thumbnail_platform_file_id,omitempty"`
	Width                   int                `gorm:"column:width" json:"width,omitempty"`
	Height                  int                `gorm:"column:height" json:"height,omitempty"`
	Duration                int                `gorm:"column:duration" json:"duration,omitempty"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
	DeletedAt               gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (Attachment) TableName() string {
	return "attachments"
}
