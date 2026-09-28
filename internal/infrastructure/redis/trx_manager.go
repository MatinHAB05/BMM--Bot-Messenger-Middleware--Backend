package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTrxConflict = errors.New("redis transaction: watched key changed, retries exhausted")

type TrxManager interface {
	// WithTransaction runs fn inside MULTI/EXEC (no WATCH). Every command
	// issued through trxCtx is only QUEUED; results are not available inside fn.
	WithTransaction(ctx context.Context, fn func(trxCtx context.Context) error) error

	// WithWatch is optimistic locking (WATCH/MULTI/EXEC).
	//   read : runs on the watched connection; commands execute immediately,
	//          so you can read values and make decisions.
	//   write: commands are queued and applied atomically via EXEC. If any
	//          watched key changed since WATCH, nothing is applied and the
	//          whole thing (read + write) is retried.
	// Both funcs may run several times, so they must be idempotent / side-effect free
	// apart from Redis commands.
	WithWatch(ctx context.Context, keys []string, read, write func(trxCtx context.Context) error) error
}

type redisTrxManager struct {
	cache      Cache
	maxRetries int
	backoff    time.Duration
}

func NewTrxManager(cache Cache, maxRetries int, backoff time.Duration) TrxManager {
	return &redisTrxManager{cache: cache, maxRetries: maxRetries, backoff: backoff}
}

func (m *redisTrxManager) WithTransaction(ctx context.Context, fn func(trxCtx context.Context) error) error {
	_, err := m.cache.GetClient().TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		return fn(InjectTrx(ctx, m.cache.WithCmdable(pipe)))
	})
	return err
}

func (m *redisTrxManager) WithWatch(ctx context.Context, keys []string, read, write func(trxCtx context.Context) error) error {
	client := m.cache.GetClient()

	for attempt := 0; attempt < m.maxRetries; attempt++ {
		err := client.Watch(ctx, func(tx *redis.Tx) error {
			if read != nil {
				if err := read(InjectTrx(ctx, m.cache.WithCmdable(tx))); err != nil {
					return err
				}
			}
			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				return write(InjectTrx(ctx, m.cache.WithCmdable(pipe)))
			})
			return err
		}, keys...)

		if !errors.Is(err, redis.TxFailedErr) {
			return err // nil or a real error
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.backoff * time.Duration(attempt+1)):
		}
	}
	return fmt.Errorf("%w (keys=%v)", ErrTrxConflict, keys)
}
