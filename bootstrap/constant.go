package bootstrap

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// TODO : replace some of them to .env
// Structural constants grouped into dedicated categories.
type ServerConstants struct {
	APIVersion       string        `json:"api_version"`
	HTTPReadTimeout  time.Duration `json:"http_read_timeout"`
	HTTPWriteTimeout time.Duration `json:"http_write_timeout"`
	ShutdownTimeout  time.Duration `json:"shutdown_timeout"`
}

type AuthDefaults struct {
	AccessTokenTTL  time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl"`
}

type PathConstants struct {
}

type BotConstants struct {
	ListenerRestartDelay time.Duration `json:"listener_restart_delay"`
}

type RedisConstants struct {
	MaxRetries int
	Backoff    time.Duration
}

// AppConstants holds all categorized static values
type Constants struct {
	Server ServerConstants `json:"server"`
	Auth   AuthDefaults    `json:"auth"`
	Paths  PathConstants   `json:"paths"`
	Redis  RedisConstants  `json:"redis"`
	Bot    BotConstants    `json:"bot"`
}

func LoadNewConstants() *Constants {
	cons := Constants{
		Server: ServerConstants{
			APIVersion:       "v1",
			HTTPReadTimeout:  15 * time.Second,
			HTTPWriteTimeout: 15 * time.Second,
			ShutdownTimeout:  10 * time.Second,
		},
		Auth: AuthDefaults{
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
		Paths: PathConstants{},
		Bot: BotConstants{
			ListenerRestartDelay: 5 * time.Second,
		},
		Redis: RedisConstants{
			MaxRetries: 5,
			Backoff:    5 * time.Millisecond,
		},
	}
	PrintConstants(&cons)
	return &cons
}

func PrintConstants(con *Constants) {
	prettyJSON, err := json.MarshalIndent(con, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal constants: %v", err)
		return
	}
	fmt.Println("=== Loaded Structural Constants ===")
	fmt.Println(string(prettyJSON))
	fmt.Println("===================================")
}
