// Package router registers every v1 route and attaches the correct
// middleware chain to each. It is the single place that decides which
// endpoints are public, which require authentication, which require RBAC,
// and which carry an endpoint-specific rate limit -- keeping that policy
// out of the handlers themselves.
package apirouter

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/paseto"
	repository_contract "messenger-backend/internal/domain/repository"
	apihandler "messenger-backend/internal/presentation/v1/api/handler"
	"messenger-backend/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Config carries the per-scope sliding-window rate limits (requests/minute),
// sourced from the environment by bootstrap so they're tunable without a
// code change.
type Config struct {
	GlobalPerMinute    int
	AuthPerMinute      int
	BroadcastPerMinute int
}

// Dependencies bundles everything the router needs. Built once in
// bootstrap/init.go and passed in here.
type Dependencies struct {
	repository_contract.Repositories
	service_contract.Services
	apihandler.APIHandlers

	TokenMaker paseto.Maker
	Logger     logger.Logger
}

// New builds the fully wired Gin engine: global middleware first, then the
// versioned route tree under /api/v1.
func New(deps Dependencies, cfg *Config) *gin.Engine {
	engine := gin.New()

	// Recovery must be outermost so it can catch a panic anywhere
	// downstream, including in the rate limiter itself.

	//debug
	engine.Use(gin.Logger(), gin.Recovery())
	// engine.Use(middleware.Recovery(deps.Logger))

	// TODO : for now beacause we use polling method for get history of chat history we need to unlimited or very low limiter for endpoints SO when polling is replaced with ws rate limiter must used again
	// engine.Use(apimiddleware.RateLimit(deps.RateLimiterRepository, "global", cfg.GlobalPerMinute, time.Minute, deps.Logger))

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/api/v1")

	RegisterAuthRoutes(v1, deps, cfg)
	RegisterCompanyRoutes(v1, deps)
	RegisterUserRoutes(v1, deps, cfg)
	RegisterChatRoutes(v1, deps)
	RegisterBroadcastRoutes(v1, deps, cfg)
	RegisterAttachmentRoutes(v1, deps)

	return engine
}
