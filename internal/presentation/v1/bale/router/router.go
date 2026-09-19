package balerouter

// import (
// 	service_contract "messenger-backend/internal/application/contract"
// 	repository_contract "messenger-backend/internal/domain/repository"
// 	balemiddleware "messenger-backend/internal/presentation/middleware/bale"
// 	balehandlers "messenger-backend/internal/presentation/v1/bale/handler"
// 	"messenger-backend/pkg/logger"
// 	"messenger-backend/pkg/messenger/telegram"
// 	"messenger-backend/pkg/tellogger"

// 	"github.com/go-telegram/bot"
// )

// type Config struct {
// 	Token       string
// 	BotUsername string
// }

// // Dependencies bundles everything the router needs. Built once in
// // bootstrap/init.go and passed in here.
// type Dependencies struct {
// 	repository_contract.Repositories
// 	service_contract.Services
// 	balehandlers.BaleHandlers

// 	Logger     logger.Logger
// 	BaleLogger tellogger.Logger
// }

// func New(deps Dependencies, cfg *Config) (*telegram.Adapter, error) {
// 	baleOpts := []bot.Option{
// 		bot.WithServerURL("https://tapi.bale.ai"),
// 		bot.WithDebugHandler(bot.DebugHandler(deps.BaleLogger)),
// 		bot.WithDebug(),
// 		bot.WithWorkers(1), // pool worker for handle updates
// 	}

// 	baleAdapter, err := telegram.NewAdapter(cfg.Token, deps.BasicHandler.OnUpdate, deps.BasicHandler.OnMessage, baleOpts, setupRouting(deps, cfg))

// 	if err != nil {
// 		return nil, err
// 	}
// 	baleAdapter.SetPlatform("bale")

// 	return baleAdapter, nil
// }
// func setupRouting(deps Dependencies, cfg *Config) func(b *bot.Bot) {
// 	return func(b *bot.Bot) {
// 		// 1
// 		parseLinkOTPMiddleware := balemiddleware.ParseLinkOTPMiddleware(&balemiddleware.LinkCommandParser{})
// 		featChatsLinkCommandRouter := FeatChatsLinkCommandRouter
// 		b.RegisterHandlerMatchFunc(featChatsLinkCommandRouter, deps.DirectFeatChatCommandHandler.FeatChatsLinkCommand, parseLinkOTPMiddleware)

// 		// 2
// 		parseDeepStartGroupOTPMiddleware := balemiddleware.ParseDeepStartGroupOTPMiddleware(&balemiddleware.DeppStartGroupChatCompanyCommandParser{
// 			BaleUsername: cfg.BotUsername,
// 		})
// 		featDeepStartGroupCommandRouter := FeatDeepStartGroupCommandRouter(cfg.BotUsername)
// 		b.RegisterHandlerMatchFunc(featDeepStartGroupCommandRouter, deps.FeatDeepStartGroupCommand, parseDeepStartGroupOTPMiddleware)

// 		// 3
// 		parseDeepStartChannelOTPMiddleware := balemiddleware.ParseDeepStartChannelOTPMiddleware(&balemiddleware.DeppStartChannelChatCompanyCommandParser{})
// 		setChannelPendingDeepStartChannelCommandRouter := SetChannelPendingDeepStartChannelRouter
// 		b.RegisterHandlerMatchFunc(setChannelPendingDeepStartChannelCommandRouter, deps.SetChannelPendingDeepStartChannelCommand, parseDeepStartChannelOTPMiddleware)

// 		// 4
// 		botJoinedChannelMiddleware := balemiddleware.BotJoinedChannelMiddleware()
// 		registerChanneltAcceptnessRouter := RegisterChanneltAcceptnessRouter
// 		b.RegisterHandlerMatchFunc(registerChanneltAcceptnessRouter, deps.RegisterChannelAcceptance, botJoinedChannelMiddleware)
// 	}
// }
