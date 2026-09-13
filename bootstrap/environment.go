package bootstrap

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type AppConfig struct {
	AppEnv string `mapstructure:"APP_ENV" json:"app_env"`
	Port   string `mapstructure:"PORT" json:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST" json:"db_host"`
	Port     string `mapstructure:"DB_PORT" json:"db_port"`
	User     string `mapstructure:"DB_USER" json:"db_user"`
	Password string `mapstructure:"DB_PASSWORD" json:"db_password"`
	Name     string `mapstructure:"DB_NAME" json:"db_name"`
	SSLMode  string `mapstructure:"DB_SSLMODE" json:"db_ssl_mode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST" json:"redis_host"`
	Port     string `mapstructure:"REDIS_PORT" json:"redis_port"`
	Password string `mapstructure:"REDIS_PASSWORD" json:"redis_password"`
	DB       int    `mapstructure:"REDIS_DB" json:"redis_db"`
}

type AuthConfig struct {
	PasetoSymmetricKey string        `mapstructure:"PASETO_SYMMETRIC_KEY" json:"paseto_symmetric_key"`
	AccessTokenTTL     time.Duration `mapstructure:"ACCESS_TOKEN_TTL" json:"access_token_ttl"`
	RefreshTokenTTL    time.Duration `mapstructure:"REFRESH_TOKEN_TTL" json:"refresh_token_ttl"`
}

type BotConfig struct {
	TelegramBotToken string `mapstructure:"TELEGRAM_BOT_TOKEN" json:"telegram_bot_token"`
	BaleBotToken     string `mapstructure:"BALE_BOT_TOKEN" json:"bale_bot_token"`
}

type AdminConfig struct {
	Username    string `mapstructure:"ADMIN_USERNAME" json:"admin_username"`
	Password    string `mapstructure:"ADMIN_PASSWORD" json:"admin_password"`
	Email       string `mapstructure:"ADMIN_EMAIL" json:"admin_email"`
	Phone       string `mapstructure:"ADMIN_PHONE" json:"admin_phone"`
	CompanyName string `mapstructure:"ADMIN_COMPANY_NAME" `
	CompanyCode string `mapstructure:"ADMIN_COMPANY_CODE" `

	Chats string `mapstructure:"ADMIN_CHATS" json:"-"`
}

type RateLimitConfig struct {
	GlobalPerMinute    int `mapstructure:"RATE_LIMIT_GLOBAL_PER_MINUTE" json:"rate_limit_global_per_minute"`
	AuthPerMinute      int `mapstructure:"RATE_LIMIT_AUTH_PER_MINUTE" json:"rate_limit_auth_per_minute"`
	BroadcastPerMinute int `mapstructure:"RATE_LIMIT_BROADCAST_PER_MINUTE" json:"rate_limit_broadcast_per_minute"`
}

type LoggerConfig struct {
	CleanPath string `mapstructure:"LOG_DIR_FILE_PATH_CLEAN" json:"log_dir_file_path_clean"`
	Path      string `mapstructure:"LOG_DIR_FILE_PATH" json:"log_dir_file_path"`

	ErrCleanPath string `mapstructure:"LOG_ERR_DIR_FILE_PATH_CLEAN"`
	ErrPath      string `mapstructure:"LOG_ERR_DIR_FILE_PATH"`
}

type CasbinConfig struct {
	RBACModelFilePath string `mapstructure:"CASBIN_CONF_FILE_PATH" json:"rbac_model_path"`
}

type MigrationConfig struct {
	MigrationsDirPath string `mapstructure:"DATABASE_MIGRATION_DIR_PATH" json:"migrations_dir_path"`
}

type OTPConfig struct {
	OTPTokenTTL time.Duration `mapstructure:"OTP_TOKEN_TTL" json:"otp_token_ttl"`
}

type EmailConfig struct {
	BMMEmail            string `mapstructure:"BMM_EMAIL"              json:"bmm_email"`
	BMMEmailAppPassword string `mapstructure:"BMM_EMAIL_APP_PASSWORD" json:"bmm_email_app_password"`
	SMTPHost            string `mapstructure:"SMTP_HOST"                        json:"smtp_host"`
	SMTPPort            string `mapstructure:"SMTP_HOST_PORT"                   json:"smtp_host_port"`
	RealSend            bool `mapstructure:"BMM_EMAIL_REAL_SEND"           `
	LogInFile           bool `mapstructure:"BMM_EMAIL_LOG_IN_FILE"       `
}

type Environment struct {
	App       AppConfig       `mapstructure:",squash" json:"app"`
	Database  DatabaseConfig  `mapstructure:",squash" json:"database"`
	Redis     RedisConfig     `mapstructure:",squash" json:"redis"`
	Auth      AuthConfig      `mapstructure:",squash" json:"auth"`
	OTP       OTPConfig       `mapstructure:",squash" json:"otp"`
	Bot       BotConfig       `mapstructure:",squash" json:"bot"`
	Admin     AdminConfig     `mapstructure:",squash" json:"admin"`
	RateLimit RateLimitConfig `mapstructure:",squash" json:"rate_limit"`
	Logger    LoggerConfig    `mapstructure:",squash" json:"logger"`
	Casbin    CasbinConfig    `mapstructure:",squash" json:"casbin"`
	Migration MigrationConfig `mapstructure:",squash" json:"migration"`
	Email     EmailConfig     `mapstructure:",squash" json:"email"`
}

func LoadEnvironment() *Environment {
	v := viper.New()

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Printf("No .env file found or error reading it, using environment variables and defaults: %v", err)
	}

	var env Environment
	if err := v.Unmarshal(&env); err != nil {
		log.Fatalf("Failed to unmarshal environment config: %v", err)
	}

	printConfig(&env)

	return &env
}

func printConfig(env *Environment) {
	prettyJSON, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		log.Printf("Failed to print loaded config: %v", err)
		return
	}
	fmt.Println("=== Loaded Configuration ===")
	fmt.Println(string(prettyJSON))
	fmt.Println("=============================")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return parsed
}
