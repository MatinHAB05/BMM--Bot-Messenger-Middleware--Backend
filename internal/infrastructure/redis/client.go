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

// Cache is the analogue of database.Database.
//
//   - GetRDB() returns a redis.Cmdable. Outside a transaction it is the real
//     *redis.Client; inside a transaction it is a *redis.Tx (watch phase) or a
//     redis.Pipeliner (MULTI/EXEC phase). Repositories must only use the
//     Cmdable API so they work in all three modes.
//   - GetClient() returns the raw *redis.Client (Watch/Close/Ping, etc.).
//   - WithCmdable() is the analogue of Database.WithTx().
type Cache interface {
	GetRDB() redis.Cmdable
	GetClient() *redis.Client
	WithCmdable(c redis.Cmdable) Cache
	GetRedisConfig() Config
}

type RedisDatabase struct {
	rdb *redis.Client
	cmd redis.Cmdable
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
		if _, err := rdb.Ping(ctx).Result(); err != nil {
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

func (r *RedisDatabase) GetRDB() redis.Cmdable {
	if r.cmd != nil {
		return r.cmd
	}
	return r.rdb
}

func (r *RedisDatabase) GetClient() *redis.Client { return r.rdb }

func (r *RedisDatabase) WithCmdable(c redis.Cmdable) Cache {
	return &RedisDatabase{rdb: r.rdb, cmd: c, Config: r.Config}
}

func (r *RedisDatabase) GetRedisConfig() Config { return *r.Config }
