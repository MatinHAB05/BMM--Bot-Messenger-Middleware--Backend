package telegramrouter

import (
	service_contract "messenger-backend/internal/application/contract"
	repository_contract "messenger-backend/internal/domain/repository"
	telegrammiddleware "messenger-backend/internal/presentation/middleware/telegram"
	telegramhandlers "messenger-backend/internal/presentation/v1/telegram/handler"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger/telegram"
	"messenger-backend/pkg/tellogger"

	"github.com/go-telegram/bot"
)

type Config struct {
	Token       string
	BotUsername string
}

// Dependencies bundles everything the router needs. Built once in
// bootstrap/init.go and passed in here.
type Dependencies struct {
	repository_contract.Repositories
	service_contract.Services
	telegramhandlers.TelegramHandlers

	Logger    logger.Logger
	TelLogger tellogger.Logger
}

func New(deps Dependencies, cfg *Config) (*telegram.Adapter, error) {
	telegramOpts := []bot.Option{
		bot.WithDebugHandler(bot.DebugHandler(deps.TelLogger)),
		bot.WithDebug(),
	}

	telegramAdapter, err := telegram.NewAdapter(cfg.Token, deps.BasicHandler.OnUpdate, deps.BasicHandler.OnMessage, telegramOpts, setupRouting(deps, cfg))
	if err != nil {
		return nil, err
	}

	return telegramAdapter, nil
}
func setupRouting(deps Dependencies, cfg *Config) func(b *bot.Bot) {
	return func(b *bot.Bot) {
		// 1
		parseLinkOTPMiddleware := telegrammiddleware.ParseLinkOTPMiddleware(&telegrammiddleware.LinkCommandParser{})
		featChatsLinkCommandRouter := FeatChatsLinkCommandRouter
		b.RegisterHandlerMatchFunc(featChatsLinkCommandRouter, deps.DirectFeatChatCommandHandler.FeatChatsLinkCommand, parseLinkOTPMiddleware)

		// 2
		parseDeepStartGroupOTPMiddleware := telegrammiddleware.ParseDeepStartGroupOTPMiddleware(&telegrammiddleware.DeppStartGroupChatCompanyCommandParser{
			TelegramUsername: cfg.BotUsername,
		})
		featDeepStartGroupCommandRouter := FeatDeepStartGroupCommandRouter(cfg.BotUsername)
		b.RegisterHandlerMatchFunc(featDeepStartGroupCommandRouter, deps.FeatDeepStartGroupCommand, parseDeepStartGroupOTPMiddleware)

		// 3
		parseDeepStartChannelOTPMiddleware := telegrammiddleware.ParseDeepStartChannelOTPMiddleware(&telegrammiddleware.DeppStartChannelChatCompanyCommandParser{})
		setChannelPendingDeepStartChannelCommandRouter := SetChannelPendingDeepStartChannelRouter
		b.RegisterHandlerMatchFunc(setChannelPendingDeepStartChannelCommandRouter, deps.SetChannelPendingDeepStartChannelCommand, parseDeepStartChannelOTPMiddleware)

		// 4
		botJoinedChannelMiddleware := telegrammiddleware.BotJoinedChannelMiddleware()
		registerChanneltAcceptnessRouter := RegisterChanneltAcceptnessRouter
		b.RegisterHandlerMatchFunc(registerChanneltAcceptnessRouter, deps.RegisterChannelAcceptance, botJoinedChannelMiddleware)
	}
}
