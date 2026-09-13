package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"messenger-backend/internal/presentation/middleware"
)

// RegisterBroadcastRoutes wires POST /api/v1/broadcast: authenticated,
// RBAC-checked, and rate-limited as a high-friction surface per spec A.2.
func RegisterBroadcastRoutes(v1 *gin.RouterGroup, deps Dependencies, cfg *Config) {
	broadcast := v1.Group("/broadcast")
	broadcast.Use(middleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	broadcast.Use(middleware.RBAC(deps.RBACRepository, deps.Logger))
	broadcast.Use(middleware.RateLimit(deps.RateLimiterRepository, "broadcast", cfg.BroadcastPerMinute, time.Minute, deps.Logger))

	broadcast.POST("", deps.BroadcastHandler.Broadcast)
}
