package balemiddleware

import (
	"context"
	"messenger-backend/pkg/messenger/telegram"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Define a custom context key type to avoid collisions
type channelInfoContextKey string

const (
	channelChatIDContextKey channelInfoContextKey = "bot_joined_channel_id"
	inviterIDContextKey     channelInfoContextKey = "bot_joined_inviter_id"
)

// BotJoinedChannelMiddleware checks if the update is a channel join event and sets channel info in ctx.
func BotJoinedChannelMiddleware() bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			chatID, inviterID, ok := telegram.ExtractBotJoinedChannelInfo(update)
			if ok {
				ctx = SetBotJoinedChannelInfo(ctx, chatID, inviterID)
			}

			next(ctx, b, update)
		}
	}
}

// SetBotJoinedChannelInfo stores channelID and inviterID into the context.
func SetBotJoinedChannelInfo(ctx context.Context, chatID int64, inviterID int64) context.Context {
	ctx = context.WithValue(ctx, channelChatIDContextKey, chatID)
	ctx = context.WithValue(ctx, inviterIDContextKey, inviterID)
	return ctx
}

// GetBotJoinedChannelInfo retrieves channelID, inviterID, and ok flag from the context.
func GetBotJoinedChannelInfo(ctx context.Context) (chatID int64, inviterID int64, ok bool) {
	chatID, ok1 := ctx.Value(channelChatIDContextKey).(int64)
	inviterID, ok2 := ctx.Value(inviterIDContextKey).(int64)

	if !ok1 || !ok2 {
		return 0, 0, false
	}

	return chatID, inviterID, true
}
