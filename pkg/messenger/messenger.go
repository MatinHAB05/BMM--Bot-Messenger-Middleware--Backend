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
	"time"
)

// MessengerClient is implemented independently by each messenger engine.
type MessengerClient interface {
	// SendMessage delivers content to targetID (a chat/channel/group id,
	// or an @username where the platform supports it).
	SendMessage(ctx context.Context, targetID string, content string) (*MessageUpdate, error)
	// Platform returns the stable lowercase identifier used throughout the
	// API (request payloads, log fields, DB rows) to refer to this engine,
	// e.g. "telegram" or "bale".
	Platform() string
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
}
