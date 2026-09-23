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
	baleAdapter     *telegram.Adapter // Bale speaks a Telegram-compatible bot API, so it reuses the same adapter type.

	emailPool     *ants.Pool
	broadcastPool *ants.Pool
}

// Init wires up the whole application: config, infrastructure, repositories,
// domain services, the Telegram/Bale bot platforms and the HTTP API, in that
// order. Each stage is a small helper below so this function stays a plain,
// top-to-bottom readable list of "what happens first".
func Init(ctx context.Context) (*App, error) {
	// ----------------------------------------------------------------
	// 1. Configuration
	// ----------------------------------------------------------------
	env := LoadEnvironment()
	cons := LoadNewConstants()

	// ----------------------------------------------------------------
	// 2. Logging
	// ----------------------------------------------------------------
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

	if err := validatePasetoKey(env); err != nil {
		return nil, err
	}

	// ----------------------------------------------------------------
	// 3. Database & migrations
	// ----------------------------------------------------------------
	db, err := database.NewPostgresDatabase(ctx, newDatabaseConfig(env))
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	if err := runMigrations(db.GetGormDB(), env.Migration.MigrationsDirPath, log); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	// ----------------------------------------------------------------
	// 4. Redis
	// ----------------------------------------------------------------
	redisClient, err := redisinfra.NewRedisDatabase(ctx, newRedisConfig(env))
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	// ----------------------------------------------------------------
	// 5. Auth primitives (token maker + RBAC enforcer)
	// ----------------------------------------------------------------
	tokenMaker, err := paseto.NewPasetoMaker(env.Auth.PasetoSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("init paseto maker: %w", err)
	}

	enforcer, err := rbac.NewEnforcer(db, newRBACConfig(env))
	if err != nil {
		return nil, fmt.Errorf("init casbin enforcer: %w", err)
	}

	// ----------------------------------------------------------------
	// 6. Worker pools
	// ----------------------------------------------------------------
	emailPool, err := newAntsPool(5, "email")
	if err != nil {
		return nil, err
	}

	broadcastPool, err := newAntsPool(20, "broadcast")
	if err != nil {
		return nil, err
	}

	
	batchPool, err := newAntsPool(20, "batch")
	if err != nil {
		return nil, err
	}

	// ----------------------------------------------------------------
	// 7. Repositories
	// ----------------------------------------------------------------
	s3Repo, err := storage.NewMinIORepository(newS3StorageConfig(env))
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
	mediaGroupRepo := infrarepo.NewMediaGroupRepository(redisClient)
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
		AttachmentRepository:     attachRepo,
		RateLimiterRepository:    rateLimitRepo,
		RBACRepository:           rbacRepo,
		SentBaleMsgRepository:    sentbalemsgRepo,
		UserRepository:           userRepo,
	}

	// ----------------------------------------------------------------
	// 8. Static assets
	// ----------------------------------------------------------------
	statics, err := static.InitStaticFiles()
	if err != nil {
		return nil, fmt.Errorf("init statics : %w", err)
	}

	// ----------------------------------------------------------------
	// 9. Domain services (not an exhaustive list)
	// ----------------------------------------------------------------
	trxManger := database.NewTrxManager(db)
	emailDelivery := appservice.NewEmailService(mail.NewMailer(newMailerConfig(env)), env.Email.LogInFile, env.Email.RealSend, statics, log, emailPool)
	mediaGroupService := appservice.NewMediaGroupService(mediaGroupRepo, log, newMediaGroupServiceConfig())

	gen := otp.NewDefaultCodeGenerator()
	otpService, _ := appservice.NewOTPService(otpRepo, emailDelivery, env.App.AppEnv, log,
		otp.NewPhoneStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewEmailStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewLinkGroupChatCompanyStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewLinkChannelChatCompanyStrategy(env.OTP.OTPTokenTTL, 5, gen),
		otp.NewRegisterUserStrategy(env.OTP.OTPTokenTTL, 5, gen),
	)
	attachService := appservice.NewAttachmentService(mediaGroupService, attachRepo, chatHisRepo, s3Repo, trxManger, log, env.S3.Bucket, time.Hour)
	authService := appservice.NewAuthService(userRepo, companyRepo, rbacRepo, tokenRepo, otpService, trxManger, tokenMaker, log, newAuthServiceConfig(env))

	channelPendingService := appservice.NewChannelPendingService(channelPendingRepo, log, newChannelPendingServiceConfig(env))
	chatHisService := appservice.NewChatHistoryService(chatRepo, chatHisRepo, log)
	chatService := appservice.NewChatService(chatRepo, chatHisRepo, companyRepo, otpService, log)
	chatLinkService := appservice.NewChatLinkService(chatService, otpService, channelPendingService, log, errlog)
	companyServie := appservice.NewCompanyService(companyRepo, otpService, log)
	rbacService := appservice.NewRBACService(rbacRepo, log)
	sentbalemsgService := appservice.NewSentBaleMsgService(sentbalemsgRepo, log, newSentBaleMsgServiceConfig())
	userService := appservice.NewUserService(userRepo, otpService, rbacRepo, log)
	Services := service_contract.Services{
		AuthService:           authService,
		BroadcastService:      nil, // filled in once the Telegram/Bale adapters exist, see step 12/13 below.
		ChannelPendingService: channelPendingService,
		ChatHistoryService:    chatHisService,
		ChatLinkService:       chatLinkService,
		ChatService:           chatService,
		CompanyService:        companyServie,
		OTPService:            otpService,
		AttachmentService:     attachService,
		RBACService:           rbacService,
		SentBaleMsgService:    sentbalemsgService,
		UserService:           userService,
	}

	// ----------------------------------------------------------------
	// 10. Seed database
	// ----------------------------------------------------------------
	if _, err := seed.Run(ctx, companyRepo, userRepo, rbacRepo, chatRepo, newSeedConfig(env), log); err != nil {
		return nil, fmt.Errorf("seed database: %w", err)
	}

	// ==================================================================
	// 11. Messaging platforms
	//
	// Two bot platforms are wired up the same way: build the handlers,
	// bundle them into that platform's Dependencies/Config, then try to
	// start the adapter. If the platform's bot token isn't configured we
	// just log a warning and skip it instead of failing startup — a
	// deployment might only want one of the two.
	// ==================================================================

	// --- Telegram -----------------------------------------------------

	basicTelegramHandler := telegramhandlers.NewBasicHandler(chatService, chatHisService, attachService, s3Repo, env.S3.Bucket, log, errlog)
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

	// --- Bale -----------------------------------------------------------
	// Bale (بله) is an Iranian messenger with a Telegram-compatible bot API,
	// so its handlers/router mirror the Telegram ones above almost exactly —
	// same command handlers, same dependency shape, just its own bot token
	// and its own logger/adapter instance.
	//TODO
	basicBaleHandler := balehandlers.NewBasicHandler(chatService, chatHisService, attachService, sentbalemsgService, s3Repo, env.S3.Bucket, log, errlog)
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

	// --- Start whichever adapters have a bot token configured ----------
	var clients []messenger.MessengerClient

	telegramAdapter, err := newTelegramAdapter(env.Bot.TelegramBotToken, telegramDeps, telegramCfg, log)
	if err != nil {
		return nil, err
	}
	if telegramAdapter != nil {
		clients = append(clients, telegramAdapter)
	}
	//TODO

	baleAdapter, err := newBaleAdapter(env.Bot.BaleBotToken, baleDeps, baleCfg, log)
	if err != nil {
		return nil, err
	}
	if baleAdapter != nil {
		clients = append(clients, baleAdapter)
	}

	// ----------------------------------------------------------------
	// 12. Broadcast service — depends on the messaging clients above.
	// ----------------------------------------------------------------
	broadcastService := appservice.NewBroadcastService(
		clients,
		chatRepo,
		chatHisRepo,
		baleDeps,
		s3Repo,
		log,
		newBroadcastServiceConfig(env),
		broadcastPool,
		batchPool,
	)

	Services.BroadcastService = broadcastService

	// ----------------------------------------------------------------
	// 13. HTTP API handlers & router
	// ----------------------------------------------------------------
	apihandler.InitErrLogger(errlog)
	apimiddleware.InitErrLogger(errlog)
	telegrammiddleware.InitErrLogger(errlog)
	balemiddleware.InitErrLogger(errlog)

	attachmenHandler := apihandler.NewAttachmentHandler(attachService)
	authHandler := apihandler.NewAuthHandler(authService)
	broadcastHandler := apihandler.NewBroadcastHandler(broadcastService)
	chatHandler := apihandler.NewChatHandler(chatService, companyServie)
	chatHisHandler := apihandler.NewChatHistoryHandler(chatHisService)
	companyHandler := apihandler.NewCompanyHandler(companyServie)
	userHandler := apihandler.NewUserHandler(userService)
	APIHandlers := apihandler.APIHandlers{
		AttachmentHandler:  attachmenHandler,
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

	// ----------------------------------------------------------------
	// 14. Assemble the App
	// ----------------------------------------------------------------
	return &App{
		Env:    env,
		Const:  cons,
		Logger: log,

		DB:    db.GetGormDB(),
		Redis: redisClient.GetRDB(),

		Router: apirouter.New(deps, newAPIRouterConfig(env)),

		telegramAdapter: telegramAdapter,

		baleAdapter: baleAdapter, // nil, // baleAdapter,

		emailPool:     emailPool,
		broadcastPool: broadcastPool,
	}, nil
}

// ======================================================================
// Small config builders
//
// These just turn *Environment fields (or, in a couple of cases, fixed
// placeholder values) into the Config structs each package expects. Pulling
// them out of Init keeps the constructor calls above readable — the inputs
// themselves are untouched, only where they're built changed.
// ======================================================================

func validatePasetoKey(env *Environment) error {
	if env.Auth.PasetoSymmetricKey == "" {
		return fmt.Errorf("PASETO_SYMMETRIC_KEY is not set (generate one with `openssl rand -hex 32`)")
	}
	return nil
}

func newDatabaseConfig(env *Environment) *database.Config {
	return &database.Config{
		Host:     env.Database.Host,
		Port:     env.Database.Port,
		User:     env.Database.User,
		Password: env.Database.Password,
		DBName:   env.Database.Name,
		SSLMode:  env.Database.SSLMode,
	}
}

func newRedisConfig(env *Environment) *redis.Config {
	return &redis.Config{
		Host:      env.Redis.Host,
		Port:      env.Redis.Port,
		Password:  env.Redis.Password,
		RDBNumber: env.Redis.DB,
	}
}

func newRBACConfig(env *Environment) *rbac.Config {
	return &rbac.Config{
		ModelConfigFilePath: env.Casbin.RBACModelFilePath,
	}
}

// newAntsPool creates a worker pool of the given size and wraps any error
// with which pool failed, matching the "init <label> ants pool" messages
// used before this was pulled out into a helper.
func newAntsPool(size int, label string) (*ants.Pool, error) {
	pool, err := ants.NewPool(size)
	if err != nil {
		return nil, fmt.Errorf("init %s ants pool: %w", label, err)
	}
	return pool, nil
}

// newS3StorageConfig is left with the same empty/placeholder values as
// before — nothing here has been filled in yet.
func newS3StorageConfig(env *Environment) storage.Config {
	return storage.Config{
		Endpoint:        env.S3.Endpoint,
		AccessKeyID:     env.S3.AccessKeyID,
		SecretAccessKey: env.S3.SecretAccessKey,
		UseSSL:          env.S3.UseSSL,
		Region:          env.S3.Region,
		Bucket:          env.S3.Bucket,
		MaxConcurrency:  env.S3.MaxConcurrency,
	}
}
func newMailerConfig(env *Environment) mail.EmailConfig {
	return mail.EmailConfig{
		From:     env.Email.BMMEmail,
		Password: env.Email.BMMEmailAppPassword,
		SMTPHost: env.Email.SMTPHost,
		SMTPPort: env.Email.SMTPPort,
	}
}

func newAuthServiceConfig(env *Environment) *appservice.AuthServiceConfig {
	return &appservice.AuthServiceConfig{
		AccessTokenTTL:  env.Auth.AccessTokenTTL,
		RefreshTokenTTL: env.Auth.RefreshTokenTTL,
		AppEnv:          env.App.AppEnv,
	}
}

func newChannelPendingServiceConfig(env *Environment) appservice.ChannelPendingServiceConfig {
	return appservice.ChannelPendingServiceConfig{
		TTL: env.OTP.OTPTokenTTL,
	}
}

func newSentBaleMsgServiceConfig() appservice.SentBaleMsgServiceConfig {
	return appservice.SentBaleMsgServiceConfig{
		TTL: time.Second * 5000,
	}
}

func newMediaGroupServiceConfig() appservice.MediaGroupServiceConfig {
	return appservice.MediaGroupServiceConfig{
		TTL: time.Minute * 5,
	}
}

func newBroadcastServiceConfig(env *Environment) appservice.BroadcastConfig {
	return appservice.BroadcastConfig{
		JobTimeout:       0,
		AttachmentBucket: env.S3.Bucket,
	}
}

func newSeedConfig(env *Environment) seed.Config {
	return seed.Config{
		SuperAdminUsername:        env.SuperAdmin.Username,
		SuperAdminPassword:        env.SuperAdmin.Password,
		SuperAdminEmail:           env.SuperAdmin.Email,
		SuperAdminPhone:           env.SuperAdmin.Phone,
		CompanyName:               env.SuperAdmin.CompanyName,
		CompanyCode:               env.SuperAdmin.CompanyCode,
		SuperAdminChatsRawPayload: env.SuperAdmin.Chats,
	}
}

func newAPIRouterConfig(env *Environment) *apirouter.Config {
	return &apirouter.Config{
		GlobalPerMinute:    env.RateLimit.GlobalPerMinute,
		AuthPerMinute:      env.RateLimit.AuthPerMinute,
		BroadcastPerMinute: env.RateLimit.BroadcastPerMinute,
	}
}

// ======================================================================
// Messaging platform adapters
// ======================================================================

// newTelegramAdapter starts the Telegram adapter, or returns (nil, nil) and
// logs a warning if no bot token is configured — startup is not supposed to
// fail just because Telegram is disabled for this deployment.
func newTelegramAdapter(token string, deps telegramrouter.Dependencies, cfg telegramrouter.Config, log logger.Logger) (*telegram.Adapter, error) {
	if token == "" {
		log.Warn("TELEGRAM_BOT_TOKEN not set -- Telegram engine disabled")
		return nil, nil
	}

	adapter, err := telegramrouter.New(deps, &cfg)
	if err != nil {
		return nil, fmt.Errorf("init telegram adapter: %w", err)
	}
	return adapter, nil
}

// newBaleAdapter is the Bale (بله) equivalent of newTelegramAdapter above:
// same "skip and warn if no token" behavior, just pointed at the Bale
// router/config instead of Telegram's.
func newBaleAdapter(token string, deps balerouter.Dependencies, cfg balerouter.Config, log logger.Logger) (*telegram.Adapter, error) {
	if token == "" {
		log.Warn("BALE_BOT_TOKEN not set -- Bale engine disabled")
		return nil, nil
	}

	adapter, err := balerouter.New(deps, &cfg)
	if err != nil {
		return nil, fmt.Errorf("init bale adapter: %w", err)
	}
	return adapter, nil
}

// ======================================================================
// Migrations
// ======================================================================

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

// ======================================================================
// App lifecycle: starting/restarting the bot listeners, and shutdown
// ======================================================================

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
