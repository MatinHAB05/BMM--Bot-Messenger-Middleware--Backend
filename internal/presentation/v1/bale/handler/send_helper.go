// telegramhandlers/send_helper.go
package balehandlers

import (
	"context"

	"messenger-backend/internal/domain/exception"
	"messenger-backend/pkg/logger"

	"github.com/go-telegram/bot"
)

// sendText centralizes the "send a plain text reply, log on failure"
// pattern repeated across every command handler in this package.
func sendText(ctx context.Context, b *bot.Bot, errLog logger.Logger, chatID any, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		errLog.Error(err, exception.ErrSendMessagePlatform.Error(), logger.Any("chat_id", chatID))
	}
}
