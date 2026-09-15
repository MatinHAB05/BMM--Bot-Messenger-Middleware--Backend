// Package bootstrap is the application's composition root. Init wires
// every dependency -- database, Redis, PASETO, Casbin, repositories,
// services, handlers, the router, the seeder, and the Telegram/Bale
// messenger engines -- into a single *App. StartListeners then launches
// the background bot update listener goroutines described in spec
// section D, entirely from this package.

// todo : close conncetions
package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	service_contract "messenger-backend/internal/application/contract"
	appservice "messenger-backend/internal/application/service"
	"messenger-backend/internal/domain/otp"
	"messenger-backend/internal/domain/paseto"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
	"messenger-backend/internal/infrastructure/rbac"
	"messenger-backend/internal/infrastructure/redis"
	redisinfra "messenger-backend/internal/infrastructure/redis"
	infrarepo "messenger-backend/internal/infrastructure/repository"
	"messenger-backend/internal/infrastructure/seed"
	apimiddleware "messenger-backend/internal/presentation/middleware/api"
	balemiddleware "messenger-backend/internal/presentation/middleware/bale" // ✅ اضافه شد -- اگر مسیر واقعی پکیج فرق دارد، این را اصلاح کن
	telegrammiddleware "messenger-backend/internal/presentation/middleware/telegram"
	apihandler "messenger-backend/internal/presentation/v1/api/handler"
	apirouter "messenger-backend/internal/presentation/v1/api/router"
	balehandlers "messenger-backend/internal/presentation/v1/bale/handler"
	balerouter "messenger-backend/internal/presentation/v1/bale/router"
	telegramhandlers "messenger-backend/internal/presentation/v1/telegram/handler"
	telegramrouter "messenger-backend/internal/presentation/v1/telegram/router"

	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger"
	"messenger-backend/pkg/messenger/telegram"
	"messenger-backend/pkg/tellogger"
	mail "messenger-backend/pkg/wneessen-go-mail"
	"messenger-backend/static"
)

// App holds every long-lived dependency the process needs. Assembled once
// by Init; used by cmd/app/main.go to serve HTTP, start listeners, and
// shut everything down cleanly.
type App struct {
	Const  *Constants
	Env    *Environment
	Logger logger.Logger
	DB     *gorm.DB
	Redis  *goredis.Client
	Router *gin.Engine

	telegramAdapter *telegram.Adapter
	baleAdapter     *telegram.Adapter //***
}

// Init loads configuration and constructs the full dependency graph. It
// does not start the HTTP server or the background listeners -- see
// cmd/app/main.go and StartListeners respectively -- so that callers (and
// tests) can inspect/modify App before anything starts accepting traffic.
func Init(ctx context.Context) (*App, error) {
	env := LoadEnvironment()
	cons := LoadNewConstants()

	log, err := logger.New(env.Logger.Path, env.Logger.CleanPath, env.App.AppEnv != "production")
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	errlog, err := logger.New(env.Logger.ErrPath, env.Logger.ErrCleanPath, env.App.AppEnv != "production")
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	telLogger, _ := tellogger.NewLogger(false)

	if env.Auth.PasetoSymmetricKey == "" {
		return nil, fmt.Errorf("PASETO_SYMMETRIC_KEY is not set (generate one with `openssl rand -hex 32`)")
	}

	db, err := database.NewPostgresDatabase(ctx, &database.Config{
		Host:     env.Database.Host,
		Port:     env.Database.Port,
		User:     env.Database.User,
		Password: env.Database.Password,
		DBName:   env.Database.Name,
		SSLMode:  env.Database.SSLMode,
	})
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	if err := runMigrations(db.GetGormDB(), env.Migration.MigrationsDirPath, log); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	redisClient, err := redisinfra.NewRedisDatabase(ctx, &redis.Config{
		Host:      env.Redis.Host,
		Port:      env.Redis.Port,
		Password:  env.Redis.Password,
		RDBNumber: env.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	tokenMaker, err := paseto.NewPasetoMaker(env.Auth.PasetoSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("init paseto maker: %w", err)
	}

	enforcer, err := rbac.NewEnforcer(db, &rbac.Config{
		ModelConfigFilePath: env.Casbin.RBACModelFilePath,
	})
	if err != nil {
		return nil, fmt.Errorf("init casbin enforcer: %w", err)
	}

	//Repo
	tokenRepo := infrarepo.NewRedisTokenRepository(redisClient)
	channelPendingRepo := infrarepo.NewChannelPendingRepository(redisClient)
	chatHisRepo := infrarepo.NewChatHistoryRepository(db)
	chatRepo := infrarepo.NewChatRepository(db)
	companyRepo := infrarepo.NewCompanyRepository(db)
	otpRepo := infrarepo.NewOTPRepository(redisClient)
	rateLimitRepo := infrarepo.NewRedisRateLimiterRepository(redisClient)
	rbacRepo := infrarepo.NewRBACRepository(enforcer)
	userRepo := infrarepo.NewUserRepository(db)
	Repos := repository_contract.Repositories{
		AuthnTokenRepository:     tokenRepo,
		ChannelPendingRepository: channelPendingRepo,
		ChatHistoryRepository:    chatHisRepo,
		ChatRepository:           chatRepo,
		CompanyRepository:        companyRepo,
		OTPRepository:            otpRepo,
		RateLimiterRepository:    rateLimitRepo,
		RBACRepository:           rbacRepo,
		UserRepository:           userRepo,
	}

	// statics
	statics, err := static.InitStaticFiles()
	if err != nil {
		return nil, fmt.Errorf("init statics : %w", err)
	}

	//Service(It is not all of them)
	trxManger := database.NewTrxManager(db)
	emailDelivery := appservice.NewEmailService(mail.NewMailer(mail.EmailConfig{
		From:     env.Email.BMMEmail,
		Password: env.Email.BMMEmailAppPassword,
		SMTPHost: env.Email.SMTPHost,
		SMTPPort: env.Email.SMTPPort,
	}), env.Email.LogInFile, env.Email.RealSend, statics, log)
	gen := otp.NewDefaultCodeGenerator()
	otpService, _ := appservice.NewOTPService(otpRepo, emailDelivery, env.App.AppEnv, log,
		otp.NewPhoneStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewEmailStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewLinkGroupChatCompanyStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewLinkChannelChatCompanyStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewRegisterUserStrategy(env.OTP.OTPTokenTTL, 5, gen),
	)
	authService := appservice.NewAuthService(userRepo, companyRepo, rbacRepo, tokenRepo, otpService, trxManger, tokenMaker, log, &appservice.AuthServiceConfig{
		AccessTokenTTL:  env.Auth.AccessTokenTTL,
		RefreshTokenTTL: env.Auth.RefreshTokenTTL,
		AppEnv:          env.App.AppEnv,
	})

	channelPendingService := appservice.NewChannelPendingService(channelPendingRepo, log, appservice.ChannelPendingServiceConfig{
		TTL: env.OTP.OTPTokenTTL,
	})
	chatHisService := appservice.NewChatHistoryService(chatRepo, chatHisRepo, log)
	chatService := appservice.NewChatService(chatRepo, chatHisRepo, companyRepo, otpService, log)
	chatLinkService := appservice.NewChatLinkService(chatService, otpService, channelPendingService, log, errlog)
	companyServie := appservice.NewCompanyService(companyRepo, otpService, log)
	rbacService := appservice.NewRBACService(rbacRepo, log)
	userService := appservice.NewUserService(userRepo, otpService, rbacRepo, log)
	Services := service_contract.Services{
		AuthService:           authService,
		BroadcastService:      nil, // fill it after create mess adapters
		ChannelPendingService: channelPendingService,
		ChatHistoryService:    chatHisService,
		ChatLinkService:       chatLinkService,
		ChatService:           chatService,
		CompanyService:        companyServie,
		OTPService:            otpService,
		RBACService:           rbacService,
		UserService:           userService,
	}

	// seed
	if _, err := seed.Run(ctx, companyRepo, userRepo, rbacRepo, chatRepo, seed.Config{
		AdminUsername:        env.Admin.Username,
		AdminPassword:        env.Admin.Password,
		AdminEmail:           env.Admin.Email,
		AdminPhone:           env.Admin.Phone,
		CompanyName:          env.Admin.CompanyName,
		CompanyCode:          env.Admin.CompanyCode,
		AdminChatsRawPayload: env.Admin.Chats,
	}, log); err != nil {
		return nil, fmt.Errorf("seed database: %w", err)
	}
	////////////////////////////////////////////////////////////////////////////////////////////

	// platforms handlers
	//Telegram :
	basicTelegramHandler := telegramhandlers.NewBasicHandler(chatService, chatHisService, log, errlog)
	directFeatChatCommandTelegramHandler := telegramhandlers.NewDirectFeatChatCommandHandler(chatLinkService, errlog)
	featChannelTelegramHandler := telegramhandlers.NewFeatChannelHandler(chatLinkService, env.Bot.TelegramBotUsername, errlog)

	telegramHandlers := telegramhandlers.TelegramHandlers{
		BasicHandler:                 basicTelegramHandler,
		DirectFeatChatCommandHandler: directFeatChatCommandTelegramHandler,
		FeatChannelHandler:           featChannelTelegramHandler,
	}

	telegramDeps := telegramrouter.Dependencies{
		Repositories:     Repos,
		Services:         Services,
		TelegramHandlers: telegramHandlers,
		Logger:           log,
		TelLogger:        telLogger,
	}
	telegramCfg := telegramrouter.Config{
		Token:       env.Bot.TelegramBotToken,
		BotUsername: env.Bot.TelegramBotUsername,
	}

	// Bale
	basicBaleHandler := balehandlers.NewBasicHandler(chatService, chatHisService, log, errlog)
	directFeatChatCommandBaleHandler := balehandlers.NewDirectFeatChatCommandHandler(chatLinkService, errlog)
	featChannelBaleHandler := balehandlers.NewFeatChannelHandler(chatLinkService, env.Bot.BaleBotUsername, errlog) // ✅ فیکس شد: BaleBotUsername

	baleHandlers := balehandlers.BaleHandlers{ // ✅ فیکس شد: دیگه اسم پکیج رو شادو نمی‌کنه
		BasicHandler:                 basicBaleHandler,
		DirectFeatChatCommandHandler: directFeatChatCommandBaleHandler,
		FeatChannelHandler:           featChannelBaleHandler,
	}

	baleDeps := balerouter.Dependencies{
		Repositories: Repos,
		Services:     Services,
		BaleHandlers: baleHandlers, // ✅
		Logger:       log,
		TelLogger:    telLogger,
	}
	baleCfg := balerouter.Config{
		Token:       env.Bot.BaleBotToken,
		BotUsername: env.Bot.BaleBotUsername,
	}

	// Each messenger engine is wired independently and only enabled when
	// its token is configured, so the backend still boots cleanly with
	// just one platform (or neither, for local development without bots).
	var clients []messenger.MessengerClient

	var telegramAdapter *telegram.Adapter

	if env.Bot.TelegramBotToken != "" {
		telegramAdapter, err = telegramrouter.New(telegramDeps, &telegramCfg)

		if err != nil {
			return nil, fmt.Errorf("init telegram adapter: %w", err)
		}
		clients = append(clients, telegramAdapter)
	} else {
		log.Warn("TELEGRAM_BOT_TOKEN not set -- Telegram engine disabled")
	}

	var baleAdapter *telegram.Adapter

	if env.Bot.BaleBotToken != "" {
		baleAdapter, err = balerouter.New(baleDeps, &baleCfg)

		if err != nil {
			return nil, fmt.Errorf("init bale adapter: %w", err)
		}
		clients = append(clients, baleAdapter)
	} else {
		log.Warn("BALE_BOT_TOKEN not set -- Bale engine disabled")
	}

	// ... Services
	broadcastService := appservice.NewBroadcastService(clients, chatRepo, chatHisRepo, log, appservice.BroadcastConfig{
		WorkerCount: 10,
		QueueSize:   10,
	})
	Services.BroadcastService = broadcastService

	//Handlers
	apihandler.InitErrLogger(errlog)
	apimiddleware.InitErrLogger(errlog)
	telegrammiddleware.InitErrLogger(errlog)
	balemiddleware.InitErrLogger(errlog)

	authHandler := apihandler.NewAuthHandler(authService)
	broadcastHandler := apihandler.NewBroadcastHandler(broadcastService)
	chatHandler := apihandler.NewChatHandler(chatService, companyServie)
	chatHisHandler := apihandler.NewChatHistoryHandler(chatHisService)
	companyHandler := apihandler.NewCompanyHandler(companyServie)
	userHandler := apihandler.NewUserHandler(userService)
	APIHandlers := apihandler.APIHandlers{
		AuthHandler:        authHandler,
		BroadcastHandler:   broadcastHandler,
		ChatHandler:        chatHandler,
		ChatHistoryHandler: chatHisHandler,
		CompanyHandler:     companyHandler,
		UserHandler:        userHandler,
	}

	deps := apirouter.Dependencies{
		Repositories: Repos,
		Services:     Services,
		APIHandlers:  APIHandlers,
		TokenMaker:   tokenMaker,
		Logger:       log,
	}

	if env.App.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	return &App{
		Env:    env,
		Const:  cons,
		Logger: log,

		DB:    db.GetGormDB(),
		Redis: redisClient.GetRDB(),

		Router: apirouter.New(deps, &apirouter.Config{
			GlobalPerMinute:    env.RateLimit.GlobalPerMinute,
			AuthPerMinute:      env.RateLimit.AuthPerMinute,
			BroadcastPerMinute: env.RateLimit.BroadcastPerMinute,
		}),

		telegramAdapter: telegramAdapter,
		baleAdapter:     baleAdapter, // ✅ فیکس شد: دیگه nil نیست و کامنت مرده پاک شد
	}, nil
}

// runMigrations applies every pending Goose migration under MigrationsDir
// against the given GORM connection's underlying *sql.DB.
func runMigrations(db *gorm.DB, migrationDirPath string, log logger.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("acquire sql.DB for migrations: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(sqlDB, migrationDirPath); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	log.Info("database migrations applied", logger.String("dir", migrationDirPath))
	return nil
}

// StartListeners launches the Telegram and Bale background bot update
// listeners (spec section D), one goroutine each, decoupled from one
// another and from the HTTP server. Each listener is purely an I/O
// ingress adapter -- the onUpdate closures built in Init do nothing but
// call ChannelService.RecordActivity, so no business logic lives here or
// in pkg/messenger. Listeners run until ctx is cancelled; an unexpected
// exit (or panic) is logged and the listener is restarted after a short
// backoff rather than taking the rest of the process down with it.
func (a *App) StartListeners(ctx context.Context) {
	if a.telegramAdapter != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					a.Logger.Error(nil, "bot listener panicked",
						logger.String("listener", "telegram"), logger.Any("panic", r))
				}
			}()

			a.runListenerWithRestart(ctx, "telegram", func(ctx context.Context) {
				a.telegramAdapter.Listen(ctx)
			})

		}()
	}
	if a.baleAdapter != nil {

		go func() {
			defer func() {
				if r := recover(); r != nil {
					a.Logger.Error(nil, "bot listener panicked",
						logger.String("listener", "bale"), logger.Any("panic", r))
				}
			}()
			a.runListenerWithRestart(ctx, "bale", func(ctx context.Context) {
				a.baleAdapter.Listen(ctx)
			})
		}()

	}
}

func (a *App) runListenerWithRestart(ctx context.Context, name string, run func(ctx context.Context)) {
	for {
		if ctx.Err() != nil {
			return
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					a.Logger.Error(nil, "bot listener panicked",
						logger.String("listener", name), logger.Any("panic", r))
				}
			}()
			a.Logger.Info("starting bot update listener", logger.String("listener", name))
			run(ctx)
		}()

		if ctx.Err() != nil {
			return
		}

		a.Logger.Warn("bot update listener stopped unexpectedly, restarting",
			logger.String("listener", name), logger.Duration("backoff", a.Const.Bot.ListenerRestartDelay))

		select {
		case <-ctx.Done():
			return
		case <-time.After(a.Const.Bot.ListenerRestartDelay):
		}
	}
}

// Shutdown closes external connections (database, Redis). The HTTP
// server itself is shut down separately by main.go via http.Server.Shutdown,
// and listener goroutines stop on their own once ctx (passed to
// StartListeners) is cancelled.
func (a *App) Shutdown(ctx context.Context) {
	if sqlDB, err := a.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			a.Logger.Warn("error closing database connection", logger.Err(err))
		}
	}
	if err := a.Redis.Close(); err != nil {
		a.Logger.Warn("error closing redis connection", logger.Err(err))
	}

}
