package apirouter

import (
	apimiddleware "messenger-backend/internal/presentation/middleware/api"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterBroadcastRoutes wires POST /api/v1/broadcast: authenticated,
// RBAC-checked, and rate-limited as a high-friction surface per spec A.2.
func RegisterBroadcastRoutes(v1 *gin.RouterGroup, deps Dependencies, cfg *Config) {
	broadcast := v1.Group("/broadcast")
	broadcast.Use(apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	broadcast.Use(apimiddleware.RBAC(deps.RBACRepository, deps.Logger))
	broadcast.Use(apimiddleware.RateLimit(deps.RateLimiterRepository, "broadcast", cfg.BroadcastPerMinute, time.Minute, deps.Logger))

	broadcast.POST("", deps.BroadcastHandler.Send)
	broadcast.GET("/:id/platforms", deps.BroadcastHandler.GetPlatforms)
	broadcast.DELETE("/:id", deps.BroadcastHandler.Delete)
	broadcast.DELETE("/batch", deps.BroadcastHandler.DeleteBatch)
	broadcast.PUT("/:id", deps.BroadcastHandler.Update)
}
