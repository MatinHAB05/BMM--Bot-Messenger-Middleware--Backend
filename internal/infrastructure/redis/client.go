// Package redis owns the Redis connection used throughout the app for
// PASETO refresh sessions (refresh_token:<user_id>:<token_id>), the access
// token revocation blacklist (blacklist:access_token:<jti>), and the
// sliding-window rate limiter's sorted sets.
package redis

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host      string
	Port      string
	Password  string
	RDBNumber int
}

type Cache interface {
	GetRDB() *redis.Client
	GetRedisConfig() Config
}

type RedisDatabase struct {
	rdb *redis.Client
	*Config
}

var (
	rdbOnce     sync.Once
	rdbInstance *RedisDatabase
	rdbErr      error
)

func NewRedisDatabase(ctx context.Context, cfg *Config) (Cache, error) {
	rdbOnce.Do(func() {
		address := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
		rdb := redis.NewClient(&redis.Options{
			Addr:     address,
			Password: cfg.Password,
			DB:       cfg.RDBNumber,
		})
		_, err := rdb.Ping(ctx).Result()
		if err != nil {
			rdbErr = fmt.Errorf("failed to connect to Redis: %w", err)
			return
		}
		rdbInstance = &RedisDatabase{rdb: rdb, Config: cfg}
	})

	if rdbErr != nil {
		return nil, rdbErr
	}

	return rdbInstance, nil
}

func (rdb *RedisDatabase) GetRDB() *redis.Client {
	return rdbInstance.rdb
}

func (rdb *RedisDatabase) GetRedisConfig() Config {
	return *rdb.Config
}
