// Package telegram is a self-contained adapter around github.com/go-telegram/bot.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"messenger-backend/pkg/messenger"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Update and Message re-export the go-telegram/bot SDK's raw update and
// message types under this package. Nothing outside package telegram
// should need to import github.com/go-telegram/bot/models directly --
// callers (package telegramhandlers, etc.) use telegram.Update /
// telegram.Message instead, and all Telegram-SDK-shaped logic
// (extraction, field mapping) stays behind this adapter.
type (
	Update  = models.Update
	Message = models.Message
)

type ChatUpdate struct {
	TargetID    string
	Title       string
	ChatType    string
	Username    string          // @username, empty for private chats without one
	FirstName   string          // private chats only
	LastName    string          // private chats only
	IsForum     bool            // true if the supergroup has topics enabled
	Bio         string          // private chats only, if shared with the bot
	Description string          // groups/channels/supergroups
	InviteLink  string          // primary invite link, if known from this update
	MessageID   *int64          // Telegram's message_id, if this update carried a message; nil for MyChatMember/ChatMember/ChatJoinRequest
	RawPayload  json.RawMessage // full update JSON, for forensics/replay
}

type MessageUpdate struct {
	TargetID          string          // chat/channel id, same shape as ChatUpdate.TargetID
	ChatType          string          // "private", "group", "supergroup", "channel"
	ChatTitle         string          // empty for private chats
	PlatformMessageID int64           // Telegram's message_id, unique within the chat
	SenderID          string          // From.ID, or SenderChat.ID for channel posts/anonymous admins
	SenderName        string          // display name resolved from From or SenderChat
	Content           string          // Text, or Caption for captioned media
	MediaType         string          // "text", "photo", "video", "voice", "document", ... (see MessageMediaType)
	IsEdited          bool            // true for edited_message / edited_channel_post
	ReplyToMessageID  *int64          // set when this message is a reply
	Timestamp         time.Time       // message send time
	RawPayload        json.RawMessage // full update JSON, for forensics/replay
}

type UpdateHandler func(ctx context.Context, update ChatUpdate)
type MessageHandler func(ctx context.Context, msg *Update)
type TelegramHandler func(ctx context.Context, bot *tgbot.Bot, update *models.Update)

// RouterFunc allows external packages to attach custom handlers, middlewares, or routes to the bot.
type RouterFunc func(b *tgbot.Bot)

type Adapter struct {
	bot      *tgbot.Bot
	platform string
}

// Bot exposes the underlying tgbot.Bot instance for external usage.
func (a *Adapter) Bot() *tgbot.Bot {
	return a.bot
}

// NewAdapter constructs a Telegram adapter and allows registering custom routes via setupRouter callback.
func NewAdapter(token string, onUpdate UpdateHandler, onMessage MessageHandler, setRouterConfigs []tgbot.Option, setupRouter RouterFunc) (*Adapter, error) {
	a := &Adapter{}

	opts := []tgbot.Option{
		tgbot.WithMiddlewares(a.handler(onUpdate, onMessage)),
	}
	opts = append(opts, setRouterConfigs...)

	b, err := tgbot.New(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("telegram: failed to initialize bot client: %w", err)
	}

	// Invoke external router configuration if provided
	if setupRouter != nil {
		setupRouter(b)
	}

	a.bot = b

	return a, nil
}

func (a *Adapter) SetPlatform(platform string) string {
	a.platform = platform
	return a.platform
}

func (a *Adapter) Platform() string {
	return a.platform
}

// SendMessage sends a text message to targetID and returns the resulting
// MessageUpdate as reported by Telegram.
func (a *Adapter) SendMessage(ctx context.Context, targetID string, content string) (*messenger.MessageUpdate, error) {
	mes, err := a.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: ChatID(targetID),
		Text:   content,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: send message to %q: %w", targetID, err)
	}

	update, err := toMessageUpdate(mes)
	if err != nil {
		return nil, fmt.Errorf("telegram: convert sent message %d: %w", mes.ID, err)
	}
	return update, nil
}

func (a *Adapter) EditMessageText(ctx context.Context, targetID string, msgID int, content string) (*messenger.MessageUpdate, error) {
	mes, err := a.bot.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:    targetID,
		MessageID: msgID,
		Text:      content,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: update message from chat %q - msgid %d: %w", targetID, msgID, err)
	}

	update, err := toMessageUpdate(mes)
	if err != nil {
		return nil, fmt.Errorf("telegram: convert sent message %d: %w", mes.ID, err)
	}
	return update, nil
}

// DeleteMessage deletes msgID from the chat identified by targetID.
func (a *Adapter) DeleteMessage(ctx context.Context, targetID string, msgID int) error {
	ok, err := a.bot.DeleteMessage(ctx, &tgbot.DeleteMessageParams{
		ChatID:    targetID,
		MessageID: msgID,
	})
	if err != nil {
		return fmt.Errorf("telegram: delete message %d in %q: %w", msgID, targetID, err)
	}
	if !ok {
		return fmt.Errorf("telegram: delete message %d in %q: not deleted", msgID, targetID)
	}
	return nil
}

// toMessageUpdate maps a Telegram message into the adapter-agnostic
// messenger.MessageUpdate representation.
func toMessageUpdate(mes *models.Message) (*messenger.MessageUpdate, error) {
	senderID, senderName := MessageSender(mes)

	raw, err := json.Marshal(mes)
	if err != nil {
		return nil, fmt.Errorf("marshal raw payload: %w", err)
	}

	return &messenger.MessageUpdate{
		TargetID:          strconv.FormatInt(mes.Chat.ID, 10),
		ChatType:          string(mes.Chat.Type),
		ChatTitle:         ChatTitle(mes.Chat),
		PlatformMessageID: int64(mes.ID),
		SenderID:          senderID,
		SenderName:        senderName,
		Content:           MessageContent(mes),
		MediaType:         MessageMediaType(mes),
		IsEdited:          false,
		ReplyToMessageID:  replyToMessageID(mes),
		Timestamp:         time.Unix(int64(mes.Date), 0).UTC(),
		RawPayload:        raw,
	}, nil
}

// replyToMessageID extracts the ID of the message being replied to, if any.
func replyToMessageID(mes *models.Message) *int64 {
	if mes.ReplyToMessage == nil {
		return nil
	}
	id := int64(mes.ReplyToMessage.ID)
	return &id
}

func (a *Adapter) Listen(ctx context.Context) {
	a.bot.Start(ctx)
}

func (a *Adapter) handler(onUpdate UpdateHandler, onMessage MessageHandler) func(next tgbot.HandlerFunc) tgbot.HandlerFunc {
	return func(next tgbot.HandlerFunc) tgbot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if onUpdate != nil {
				log.Println("im a onUpdate")
				if chat, ok := ExtractChat(update); ok {
					raw, err := json.Marshal(update)
					if err != nil {
						raw = nil
					}

					onUpdate(ctx, ChatUpdate{
						TargetID:    strconv.FormatInt(chat.ID, 10),
						Title:       ChatTitle(chat),
						ChatType:    string(chat.Type),
						Username:    chat.Username,
						FirstName:   chat.FirstName,
						LastName:    chat.LastName,
						IsForum:     chat.IsForum,
						Bio:         "",
						Description: "",
						InviteLink:  "",
						MessageID:   ChatUpdateMessageID(update),
						RawPayload:  raw,
					})
				}
			}

			if onMessage != nil {
				log.Println("im a onMessage")
				onMessage(ctx, update)
			}
			next(ctx, b, update)
		}
	}
}

// ExtractChat pulls chat/channel identity out of whichever update variant carries it.
func ExtractChat(update *models.Update) (models.Chat, bool) {
	switch {
	case update.Message != nil:
		return update.Message.Chat, true
	case update.EditedMessage != nil:
		return update.EditedMessage.Chat, true
	case update.ChannelPost != nil:
		return update.ChannelPost.Chat, true
	case update.EditedChannelPost != nil:
		return update.EditedChannelPost.Chat, true
	case update.MyChatMember != nil:
		return update.MyChatMember.Chat, true
	case update.ChatMember != nil:
		return update.ChatMember.Chat, true
	case update.ChatJoinRequest != nil:
		return update.ChatJoinRequest.Chat, true
	default:
		return models.Chat{}, false
	}
}

// ChatUpdateMessageID returns the message_id of whichever message variant the update carries.
func ChatUpdateMessageID(update *models.Update) *int64 {
	var msg *models.Message
	switch {
	case update.Message != nil:
		msg = update.Message
	case update.EditedMessage != nil:
		msg = update.EditedMessage
	case update.ChannelPost != nil:
		msg = update.ChannelPost
	case update.EditedChannelPost != nil:
		msg = update.EditedChannelPost
	default:
		return nil
	}
	id := int64(msg.ID)
	return &id
}

// ExtractMessage pulls a MessageUpdate out of whichever update variant carries message content.
func ExtractMessage(update *models.Update) (MessageUpdate, bool) {
	var (
		msg      *models.Message
		isEdited bool
	)

	isEdited = false

	switch {
	case update.Message != nil:
		msg = update.Message
	case update.EditedMessage != nil:
		msg = update.EditedMessage
		isEdited = true
	case update.ChannelPost != nil:
		msg = update.ChannelPost
	case update.EditedChannelPost != nil:
		msg = update.EditedChannelPost
		isEdited = true
	default:
		return MessageUpdate{}, false
	}

	senderID, senderName := MessageSender(msg)

	var replyToID *int64
	if msg.ReplyToMessage != nil {
		id := int64(msg.ReplyToMessage.ID)
		replyToID = &id
	}

	raw, err := json.Marshal(update)
	if err != nil {
		raw = nil
	}

	return MessageUpdate{
		TargetID:          strconv.FormatInt(msg.Chat.ID, 10),
		ChatType:          string(msg.Chat.Type),
		ChatTitle:         ChatTitle(msg.Chat),
		PlatformMessageID: int64(msg.ID),
		SenderID:          senderID,
		SenderName:        senderName,
		Content:           MessageContent(msg),
		MediaType:         MessageMediaType(msg),
		IsEdited:          isEdited,
		ReplyToMessageID:  replyToID,
		Timestamp:         time.Unix(int64(msg.Date), 0).UTC(),
		RawPayload:        raw,
	}, true
}

// ExtractRawMessage returns the message-bearing field off whichever
// update type arrived (a normal message, an edit, or a channel post all
// carry the same shape) -- the raw SDK message itself, rather than the
// adapter-agnostic MessageUpdate that ExtractMessage returns. Callers
// that need fields MessageUpdate doesn't carry (e.g. attachment
// extraction, which needs Photo/Video/Document/Audio/Voice/Animation)
// use this instead.
//
// Moved here from package telegramhandlers, where it lived as an
// unexported extractTelegramMessage -- unchanged logic, just relocated
// next to the rest of the update-extraction helpers it belongs with.
// Note it does not check EditedChannelPost, unlike ExtractChat/
// ExtractMessage above -- that's how it was written in
// telegramhandlers too, preserved as-is rather than "fixed" as part of
// this move.
func ExtractRawMessage(update *Update) *Message {
	switch {
	case update.Message != nil:
		return update.Message
	case update.EditedMessage != nil:
		return update.EditedMessage
	case update.ChannelPost != nil:
		return update.ChannelPost
	default:
		return nil
	}
}

// MessageSender resolves a sender identity from a message.
func MessageSender(msg *models.Message) (id string, name string) {
	switch {
	case msg.From != nil && msg.From.ID != 0:
		return strconv.FormatInt(msg.From.ID, 10), UserDisplayName(*msg.From)
	case msg.SenderChat != nil:
		return "", "?"
		// return strconv.FormatInt(msg.SenderChat.ID, 10), chatTitle(*msg.SenderChat)
	default:
		return "", ""
	}
}

// UserDisplayName mirrors ChatTitle's fallback chain for the User type.
func UserDisplayName(u models.User) string {
	name := u.FirstName
	if u.LastName != "" {
		name = name + " " + u.LastName
	}
	if name == "" {
		name = u.Username
	}
	return name
}

// SenderID returns from's Telegram user ID as a string, or "" if from is
// nil.
//
// Moved here from package telegramhandlers (was an unexported
// senderID(from *models.User)) -- same behavior, just relocated so
// telegramhandlers doesn't need to touch models.User directly.
func SenderID(from *models.User) string {
	if from == nil {
		return ""
	}
	return strconv.FormatInt(from.ID, 10)
}

// SenderDisplayName returns from's display name, or "" if from is nil.
//
// Moved here from package telegramhandlers (was an unexported
// senderName(from *models.User)), whose body was already byte-for-byte
// identical to UserDisplayName's fallback chain (FirstName + LastName,
// falling back to Username) modulo the nil check -- so rather than
// duplicate that logic in its new home too, this just adds the nil
// check and delegates to UserDisplayName. Output is unchanged for every
// input.
func SenderDisplayName(from *models.User) string {
	if from == nil {
		return ""
	}
	return UserDisplayName(*from)
}

// MessageContent picks Text or Caption from a message.
func MessageContent(msg *models.Message) string {
	if msg.Text != "" {
		return msg.Text
	}
	return msg.Caption
}

// MessageMediaType classifies the message payload.
func MessageMediaType(msg *models.Message) string {
	switch {
	case len(msg.Photo) > 0:
		return "photo"
	case msg.Video != nil:
		return "video"
	case msg.Voice != nil:
		return "voice"
	case msg.VideoNote != nil:
		return "video_note"
	case msg.Audio != nil:
		return "audio"
	case msg.Document != nil:
		return "document"
	case msg.Sticker != nil:
		return "sticker"
	case msg.Animation != nil:
		return "animation"
	case msg.Contact != nil:
		return "contact"
	case msg.Location != nil:
		return "location"
	case msg.Venue != nil:
		return "venue"
	case msg.Poll != nil:
		return "poll"
	case msg.Dice != nil:
		return "dice"
	default:
		return "text"
	}
}

// ChatTitle extracts display title from Chat.
func ChatTitle(chat models.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	name := chat.FirstName
	if chat.LastName != "" {
		name = name + " " + chat.LastName
	}
	if name == "" {
		name = chat.Username
	}
	return name
}

// ChatID converts string target ID into int64 or string suitable for tgbot.SendMessageParams.
func ChatID(targetID string) any {
	if id, err := strconv.ParseInt(targetID, 10, 64); err == nil {
		return id
	}
	return targetID
}

func AddSignToUsername(username string) string {
	if strings.HasPrefix(username, "@") {
		return username
	}

	return "@" + username
}

func ExtractBotJoinedChannelInfo(update *models.Update) (chatID int64, inviterID int64, ok bool) {
	if update.MyChatMember == nil {
		return 0, 0, false
	}

	mcm := update.MyChatMember

	if mcm.Chat.Type != models.ChatTypeChannel {
		return 0, 0, false
	}

	oldStatus := mcm.OldChatMember.Type
	newStatus := mcm.NewChatMember.Type

	wasOutside := oldStatus == models.ChatMemberTypeLeft || oldStatus == models.ChatMemberTypeBanned
	isNowInside := newStatus == models.ChatMemberTypeMember || newStatus == models.ChatMemberTypeAdministrator

	if wasOutside && isNowInside {
		return mcm.Chat.ID, mcm.From.ID, true
	}

	return 0, 0, false
}
