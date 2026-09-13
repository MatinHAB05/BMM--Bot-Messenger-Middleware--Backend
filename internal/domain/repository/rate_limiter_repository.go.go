package repository_contract

import (
	"context"
	"time"
)

type RateLimiterRepository interface {
	RecordAndCount(ctx context.Context, key string, member string, windowStart int64, now time.Time, window time.Duration) (int64, error)
}
