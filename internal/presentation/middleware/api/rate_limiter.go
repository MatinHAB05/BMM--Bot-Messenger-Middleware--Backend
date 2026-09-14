// Package middleware's rate_limiter.go implements a Redis-backed sliding
// window rate limiter (per section A.2 of the spec): a sorted set per
// client+scope holds one member per request, scored by its timestamp;
// on each request we trim anything older than the window, add the new
// request, and count what remains.
package apimiddleware

import (
	"fmt"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/domain/tokencontext"
	"messenger-backend/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiterMiddleware gin.HandlerFunc

// RateLimit admits at most `limit` requests within any trailing `window`
// for a given identifier: the authenticated user's id when Auth has
// already run and attached a payload, otherwise the client IP. `scope`
// namespaces the Redis key so a global limiter and a per-endpoint limiter
// (e.g. "auth", "broadcast") don't share state.
func RateLimit(rateLimiterRepo repository_contract.RateLimiterRepository, scope string, limit int, window time.Duration, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := c.ClientIP()
		if payload, ok := tokencontext.GetPayload(c); ok {
			identifier = "user:" + payload.UserID
		}

		key := fmt.Sprintf("rate_limit:%s:%s", scope, identifier)
		ctx := c.Request.Context()
		now := time.Now()
		windowStart := now.Add(-window).UnixNano()
		member := fmt.Sprintf("%d-%s", now.UnixNano(), c.Request.Header.Get("X-Request-Id"))

		count, err := rateLimiterRepo.RecordAndCount(ctx, key, member, windowStart, now, window)
		if err != nil {
			log.Error(err, "rate limiter execution failed", logger.String("key", key))
			// Fail open: a transient Redis issue should degrade rate
			// limiting, not take the whole API down with it.
			c.Next()
			return
		}

		if count > int64(limit) {
			respondError(c, exception.ErrRateLimited)
			return
		}

		c.Next()
	}
}
