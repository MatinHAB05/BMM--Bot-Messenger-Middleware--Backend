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
	TelegramBotToken    string `mapstructure:"TELEGRAM_BOT_TOKEN" json:"telegram_bot_token"`
	BaleBotToken        string `mapstructure:"BALE_BOT_TOKEN" json:"bale_bot_token"`
	TelegramBotUsername string `mapstructure:"TELEGRAM_BOT_USERNAME" json:"telegram_bot_username"`
	BaleBotUsername     string `mapstructure:"BALE_BOT_USERNAME" json:"bale_bot_username"`
}

type SuperAdminConfig struct {
	Username    string `mapstructure:"SUPERADMIN_USERNAME" json:"super_admin_username"`
	Password    string `mapstructure:"SUPERADMIN_PASSWORD" json:"super_admin_password"`
	Email       string `mapstructure:"SUPERADMIN_EMAIL" json:"super_admin_email"`
	Phone       string `mapstructure:"SUPERADMIN_PHONE" json:"super_admin_phone"`
	CompanyName string `mapstructure:"SUPERADMIN_COMPANY_NAME" json:"super_admin_company_name"`
	CompanyCode string `mapstructure:"SUPERADMIN_COMPANY_CODE" json:"super_admin_company_code"`

	Chats string ` json:"chats"`
}

type RateLimitConfig struct {
	GlobalPerMinute    int `mapstructure:"RATE_LIMIT_GLOBAL_PER_MINUTE" json:"rate_limit_global_per_minute"`
	AuthPerMinute      int `mapstructure:"RATE_LIMIT_AUTH_PER_MINUTE" json:"rate_limit_auth_per_minute"`
	BroadcastPerMinute int `mapstructure:"RATE_LIMIT_BROADCAST_PER_MINUTE" json:"rate_limit_broadcast_per_minute"`
}

type LoggerConfig struct {
	CleanPath string `mapstructure:"LOG_DIR_FILE_PATH_CLEAN" json:"log_dir_file_path_clean"`
	Path      string `mapstructure:"LOG_DIR_FILE_PATH" json:"log_dir_file_path"`

	ErrCleanPath string `mapstructure:"LOG_ERR_DIR_FILE_PATH_CLEAN" json:"log_err_dir_file_path_clean"`
	ErrPath      string `mapstructure:"LOG_ERR_DIR_FILE_PATH" json:"log_err_dir_file_path"`
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
	BMMEmail            string `mapstructure:"BMM_EMAIL" json:"bmm_email"`
	BMMEmailAppPassword string `mapstructure:"BMM_EMAIL_APP_PASSWORD" json:"bmm_email_app_password"`
	SMTPHost            string `mapstructure:"SMTP_HOST" json:"smtp_host"`
	SMTPPort            string `mapstructure:"SMTP_HOST_PORT" json:"smtp_host_port"`
	RealSend            bool   `mapstructure:"BMM_EMAIL_REAL_SEND" json:"bmm_email_real_send"`
	LogInFile           bool   `mapstructure:"BMM_EMAIL_LOG_IN_FILE" json:"bmm_email_log_in_file"`
}

type S3Config struct {
	Endpoint        string `json:"endpoint" mapstructure:"RUSTFS_ENDPOINT"`
	AccessKeyID     string `json:"access_key_id" mapstructure:"RUSTFS_ACCESS_KEY_ID"`
	SecretAccessKey string `json:"secret_access_key" mapstructure:"RUSTFS_SECRET_ACCESS_KEY"`
	UseSSL          bool   `json:"use_ssl" mapstructure:"RUSTFS_USE_SSL"`
	Region          string `json:"region" mapstructure:"RUSTFS_REGION"`
	Bucket          string `json:"bucket" mapstructure:"RUSTFS_BUCKET"`
	MaxConcurrency  int    `json:"max_currency" mapstructure:"RUSTFS_MAX_CURRENCY"`
}

type AttachmentConfig struct {
	MaxBroadcastAttachmentFileSize int `json:"max_broadcast_attachment_file_size" mapstructure:"ATTACHMENT_MAX_BROADCAST_ATTACHMENT_FILE_SIZE"` // ? : -1 === no limits
	MaxBroadcastAttachmentFiles    int `json:"max_broadcast_attachment_files" mapstructure:"ATTACHMENT_MAX_BROADCAST_ATTACHMENT_FILES"`         // ? : -1 === no limits
}

type Environment struct {
	App        AppConfig        `mapstructure:",squash" json:"app"`
	Database   DatabaseConfig   `mapstructure:",squash" json:"database"`
	Redis      RedisConfig      `mapstructure:",squash" json:"redis"`
	Auth       AuthConfig       `mapstructure:",squash" json:"auth"`
	OTP        OTPConfig        `mapstructure:",squash" json:"otp"`
	Bot        BotConfig        `mapstructure:",squash" json:"bot"`
	SuperAdmin SuperAdminConfig `mapstructure:",squash" json:"super_admin"`
	RateLimit  RateLimitConfig  `mapstructure:",squash" json:"rate_limit"`
	Logger     LoggerConfig     `mapstructure:",squash" json:"logger"`
	Casbin     CasbinConfig     `mapstructure:",squash" json:"casbin"`
	Migration  MigrationConfig  `mapstructure:",squash" json:"migration"`
	Email      EmailConfig      `mapstructure:",squash" json:"email"`
	S3         S3Config         `mapstructure:",squash" json:"s3"`
	Attachment AttachmentConfig `mapstructure:",squash" json:"attachment"`
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

	adminChatsFile := getEnv("ADMIN_CHATS_FILE_PATH", "./adminchats.env.json")
	fileInfo, statErr := os.Stat(adminChatsFile)
	if statErr != nil || fileInfo.IsDir() {
		if statErr != nil && os.IsNotExist(statErr) {
			log.Printf("Warning: admin chats file %q not found, defaulting to empty chats list []", adminChatsFile)
		} else if fileInfo != nil && fileInfo.IsDir() {
			log.Printf("Warning: admin chats file %q is a directory, defaulting to empty chats list []", adminChatsFile)
		} else if statErr != nil {
			log.Printf("Warning: could not stat admin chats file %q (%v), defaulting to empty chats list []", adminChatsFile, statErr)
		}
		env.SuperAdmin.Chats = "[]"
	} else {
		rawChats, err := os.ReadFile(adminChatsFile)
		if err != nil {
			log.Fatalf("Failed to read-admin-chats from %q: %v", adminChatsFile, err)
		}
		env.SuperAdmin.Chats = string(rawChats)
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
