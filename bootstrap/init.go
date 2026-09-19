package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/panjf2000/ants/v2"
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
	"messenger-backend/internal/infrastructure/storage"
	apimiddleware "messenger-backend/internal/presentation/middleware/api"
	balemiddleware "messenger-backend/internal/presentation/middleware/bale"
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

type App struct {
	Const  *Constants
	Env    *Environment
	Logger logger.Logger
	DB     *gorm.DB
	Redis  *goredis.Client
	Router *gin.Engine

	telegramAdapter *telegram.Adapter
	baleAdapter     *telegram.Adapter //***

	emailPool     *ants.Pool
	broadcastPool *ants.Pool
}

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

	telLogger, _ := tellogger.NewLogger(false, "telegram")
	baleLogger, _ := tellogger.NewLogger(false, "bale")

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

	// Pools
	emailPool, err := ants.NewPool(5)
	if err != nil {
		return nil, fmt.Errorf("init email ants pool: %w", err)
	}

	broadcastPool, err := ants.NewPool(20)
	if err != nil {
		return nil, fmt.Errorf("init broadcast ants pool: %w", err)
	}

	// Repositories
	s3Repo, err := storage.NewMinIORepository(storage.Config{ ///////////////////////////////////////
		Endpoint:        "",
		AccessKeyID:     "",
		SecretAccessKey: "",
		UseSSL:          false,
		Region:          "",
		Bucket:          "",
		MaxConcurrency:  5,
	})
	if err != nil {
		return nil, fmt.Errorf("init s3: %w", err)
	}
	attachRepo := infrarepo.NewAttachmentRepository(db)
	tokenRepo := infrarepo.NewRedisTokenRepository(redisClient)
	channelPendingRepo := infrarepo.NewChannelPendingRepository(redisClient)
	chatHisRepo := infrarepo.NewChatHistoryRepository(db)
	chatRepo := infrarepo.NewChatRepository(db)
	companyRepo := infrarepo.NewCompanyRepository(db)
	otpRepo := infrarepo.NewOTPRepository(redisClient)
	rateLimitRepo := infrarepo.NewRedisRateLimiterRepository(redisClient)
	rbacRepo := infrarepo.NewRBACRepository(enforcer)
	sentbalemsgRepo := infrarepo.NewSentBaleMsgRepository(redisClient)
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
		SentBaleMsgRepository:    sentbalemsgRepo,
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
	}), env.Email.LogInFile, env.Email.RealSend, statics, log, emailPool)

	gen := otp.NewDefaultCodeGenerator()
	otpService, _ := appservice.NewOTPService(otpRepo, emailDelivery, env.App.AppEnv, log,
		otp.NewPhoneStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewEmailStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewLinkGroupChatCompanyStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewLinkChannelChatCompanyStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewRegisterUserStrategy(env.OTP.OTPTokenTTL, 5, gen),
	)
	attachService := appservice.NewAttachmentService(attachRepo, chatHisRepo, s3Repo, trxManger, log, "!!!!!!!!!!", time.Hour) ///////////////////////////////////////////////////////////
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
	sentbalemsgService := appservice.NewSentBaleMsgService(sentbalemsgRepo, log, appservice.SentBaleMsgServiceConfig{TTL: time.Millisecond * 5000})
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
		SentBaleMsgService:    sentbalemsgService,
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
	// Telegram:
	basicTelegramHandler := telegramhandlers.NewBasicHandler(chatService, chatHisService, attachService, s3Repo, nil, "!!!!", log, errlog) ////////////////////////////////////////////////////////
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
	basicBaleHandler := balehandlers.NewBasicHandler(chatService, chatHisService, sentbalemsgService, s3Repo, nil, "!!!!", log, errlog)
	directFeatChatCommandBaleHandler := balehandlers.NewDirectFeatChatCommandHandler(chatLinkService, errlog)
	featChannelBaleHandler := balehandlers.NewFeatChannelHandler(chatLinkService, env.Bot.BaleBotUsername, errlog)

	baleHandlers := balehandlers.BaleHandlers{
		BasicHandler:                 basicBaleHandler,
		DirectFeatChatCommandHandler: directFeatChatCommandBaleHandler,
		FeatChannelHandler:           featChannelBaleHandler,
	}

	baleDeps := balerouter.Dependencies{
		Repositories: Repos,
		Services:     Services,
		BaleHandlers: baleHandlers,
		Logger:       log,
		BaleLogger:   baleLogger,
	}
	baleCfg := balerouter.Config{
		Token:       env.Bot.BaleBotToken,
		BotUsername: env.Bot.BaleBotUsername,
	}

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
	broadcastService := appservice.NewBroadcastService(
		clients,
		chatRepo,
		chatHisRepo,
		sentbalemsgService,
		log,
		appservice.BroadcastConfig{JobTimeout: 0},
		broadcastPool,
	)

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
		baleAdapter:     baleAdapter,

		emailPool:     emailPool,
		broadcastPool: broadcastPool,
	}, nil
}

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

func (a *App) Shutdown(ctx context.Context) {
	if a.emailPool != nil {
		a.emailPool.Release()
	}
	if a.broadcastPool != nil {
		a.broadcastPool.Release()
	}

	if sqlDB, err := a.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			a.Logger.Warn("error closing database connection", logger.Err(err))
		}
	}
	if err := a.Redis.Close(); err != nil {
		a.Logger.Warn("error closing redis connection", logger.Err(err))
	}
}
