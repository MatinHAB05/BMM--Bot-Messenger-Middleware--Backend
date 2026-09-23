package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChatHistory is a single ingested message belonging to a Chat. RawPayload
// preserves the full, platform-native update JSON for forensics/replay;
// Content/SenderName/MediaType are extracted onto plain columns for quick
// display and filtering (search, media_type) without touching the JSONB
// blob on every query.
//
// MediaType is a plain string (not a Go-level enum type) since Telegram/
// Bale keep adding message kinds, but its documented vocabulary is now:
// "text", "photo", "video", "document", "audio", "voice", "animation", or
// "mixed" (a message carrying more than one Attachment of different
// FileTypes).
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

	IsBroadcast    bool
	HasAttachments bool
	BroadcastUUID  *uuid.UUID `gorm:"column:broadcast_uuid"`

	// Attachments holds every media file (and thumbnail reference) tied to
	// this message. ON DELETE CASCADE at the DB level (see the attachments
	// migration) means deleting a ChatHistory row also removes its
	// attachments there; that's independent of the soft-delete cascade
	// AttachmentService.DeleteAttachmentsByMessageID/
	// ReplaceMessageAttachments perform at the application level, since a
	// gorm.DeletedAt soft delete never reaches the DB-level FK trigger.
	Attachments []Attachment `gorm:"foreignKey:ChatHistoryID;constraint:OnDelete:CASCADE" json:"attachments,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ChatHistory) TableName() string {
	return "chat_histories"
}
