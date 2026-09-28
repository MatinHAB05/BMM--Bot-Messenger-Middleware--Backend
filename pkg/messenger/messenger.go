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
	"errors"
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
	EditMessageText(ctx context.Context, targetID string, msgID int, content string, hasAttachments bool) (*MessageUpdate, error)

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

// Sentinel errors a MessengerClient implementation can wrap and return
// from any of its methods, so callers that depend only on this package
// -- never on a concrete engine's SDK (see the package doc comment) --
// can still classify a failure with errors.Is(err, messenger.ErrorXxx)
// instead of importing e.g. github.com/go-telegram/bot themselves to
// check its own error sentinels. Each engine adapter is responsible for
// mapping its SDK's errors onto these (see pkg/messenger/telegram's
// mapError).
var (
	// ErrorForbidden means the engine has no access to perform the
	// action, e.g. the user blocked the bot, or the bot was removed
	// from the chat/channel/group.
	ErrorForbidden = errors.New("messenger: forbidden")

	// ErrorBadRequest means the request itself was malformed, e.g. an
	// invalid target id or an unsupported parameter combination.
	ErrorBadRequest = errors.New("messenger: bad request")

	// ErrorUnauthorized means the engine rejected the bot's
	// credentials (token, ...).
	ErrorUnauthorized = errors.New("messenger: unauthorized")

	// ErrorTooManyRequests means the engine is rate-limiting this bot.
	// Callers should back off and retry rather than treat this as a
	// permanent failure.
	ErrorTooManyRequests = errors.New("messenger: too many requests")

	// ErrorNotFound means the target of the request (chat, message,
	// file, ...) no longer exists from the engine's point of view.
	ErrorNotFound = errors.New("messenger: not found")

	// ErrorConflict means the request conflicts with the engine's
	// current state, e.g. two long-polling update loops running at
	// once for the same bot token.
	ErrorConflict = errors.New("messenger: conflict")
)
