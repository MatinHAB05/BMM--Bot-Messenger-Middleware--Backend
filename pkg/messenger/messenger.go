// Package messenger defines the platform-agnostic contract that every
// concrete messenger engine (pkg/messenger/telegram, pkg/messenger/bale)
// implements. The broadcast service and the bootstrap wiring depend only
// on this interface -- never on a concrete SDK type -- which is what lets
// Telegram and Bale stay fully decoupled from one another while still
// being fanned out to uniformly.
package messenger

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// MessengerClient is implemented independently by each messenger engine.
type MessengerClient interface {
	// SendMessage delivers content to targetID (a chat/channel/group id,
	// or an @username where the platform supports it).
	SendMessage(ctx context.Context, targetID string, content string) (*MessageUpdate, error)

	// SendAttachment delivers a single media file (photo/video/voice/
	// document/animation) to targetID, with content used as the caption
	// (may be empty). Mirrors SendMessage's return shape/error handling.
	// Sending several files (e.g. 5 photos) means calling this once per
	// file -- that fan-out is the caller's job (see broadcastService),
	// not this method's.
	//
	// attachment.Data is read exactly once by the implementation -- a
	// caller that sends the same attachment to several targets (e.g. a
	// broadcast fan-out) must give each call its own reader over the same
	// underlying bytes (e.g. a fresh bytes.NewReader per call).
	SendAttachment(ctx context.Context, targetID string, content string, attachment Attachment) (*MessageUpdate, error)

	DeleteMessage(ctx context.Context, targetID string, msgID int) error
	EditMessageText(ctx context.Context, targetID string, msgID int, content string) (*MessageUpdate, error)

	// Platform returns the stable lowercase identifier used throughout the
	// API (request payloads, log fields, DB rows) to refer to this engine,
	// e.g. "telegram" or "bale".
	Platform() string
}

// AttachmentType enumerates the media kinds SendAttachment accepts. Kept
// deliberately narrow to what Telegram/Bale both expose distinct "send"
// methods for.
type AttachmentType string

const (
	AttachmentPhoto     AttachmentType = "photo"
	AttachmentVideo     AttachmentType = "video"
	AttachmentVoice     AttachmentType = "voice"
	AttachmentAudio     AttachmentType = "audio"
	AttachmentDocument  AttachmentType = "document"
	AttachmentAnimation AttachmentType = "animation"
)

// Attachment is a single media file to send alongside (or instead of) a
// text message.
type Attachment struct {
	Type     AttachmentType
	FileName string
	Data     io.Reader
}

// AttachmentMeta is the platform-native file metadata a messenger engine
// reports back after a SendAttachment call -- the pieces
// entity.Attachment needs (PlatformFileID above all) that a plain
// MessageUpdate has no other field for. It's nil on a MessageUpdate
// returned by SendMessage/EditMessageText, since those never carry
// media.
type AttachmentMeta struct {
	PlatformFileID          string
	ThumbnailPlatformFileID string
	FileName                string
	MimeType                string
	FileSize                int64
	Width                   int
	Height                  int
	Duration                int
}

type MessageUpdate struct {
	TargetID          string          // chat/channel id, same shape as ChatUpdate.TargetID
	ChatType          string          // "private", "group", "supergroup", "channel"
	ChatTitle         string          // empty for private chats
	PlatformMessageID int64           // Telegram's message_id, unique within the chat
	SenderID          string          // From.ID, or SenderChat.ID for channel posts/anonymous admins
	SenderName        string          // display name resolved from From or SenderChat
	Content           string          // Text, or Caption for captioned media
	MediaType         string          // "text", "photo", "video", "voice", "document", ... (see messageMediaType)
	IsEdited          bool            // true for edited_message / edited_channel_post
	ReplyToMessageID  *int64          // set when this message is a reply
	Timestamp         time.Time       // message send time (edit time is not separately exposed by Bot API here)
	RawPayload        json.RawMessage // full update JSON, for forensics/replay

	// Attachment carries platform file metadata (file_id, mime type,
	// dimensions, ...) when this message was sent via SendAttachment; nil
	// for a plain text SendMessage/EditMessageText result.
	Attachment *AttachmentMeta
}
