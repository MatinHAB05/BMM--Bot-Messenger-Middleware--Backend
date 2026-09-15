package balerouter

import (
	"messenger-backend/pkg/messenger/telegram"
	"strings"

	"github.com/go-telegram/bot/models"
)

func FeatChatsLinkCommandRouter(update *models.Update) bool {
	msg, _ := telegram.ExtractMessage(update)
	if msg.IsEdited {
		return false
	}
	data := msg.Content
	return strings.HasPrefix(data, "/link") // /link <otp-code>

}

func FeatDeepStartGroupCommandRouter(telegramUsername string) func(update *models.Update) bool {
	return func(update *models.Update) bool {
		msg, _ := telegram.ExtractMessage(update)
		if msg.IsEdited {
			return false
		}
		data := msg.Content
		return strings.HasPrefix(data, "/start"+telegram.AddSignToUsername(telegramUsername)+" ") // /start<bot-username> <otp-code>
	}
}

func SetChannelPendingDeepStartChannelRouter(update *models.Update) bool {
	msg, _ := telegram.ExtractMessage(update)
	if msg.IsEdited {
		return false
	}
	text := strings.TrimSpace(msg.Content)
	args := strings.Fields(text)

	// /start <otp-code>
	return len(args) == 2 && args[0] != "/start"
}

func RegisterChanneltAcceptnessRouter(update *models.Update) bool {
	_, _, ok := telegram.ExtractBotJoinedChannelInfo(update)
	return ok
}
