package apirouter

import (
	apimiddleware "messenger-backend/internal/presentation/middleware/api"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes wires /api/v1/auth/*. login, refresh, and the two
// OTP endpoints are intentionally public (that's the point of OTP --
// authenticating without an existing token); logout requires a token so
// it knows which one to revoke. All five carry a tighter, endpoint-
// specific rate limit as a high-friction surface.
func RegisterAuthRoutes(v1 *gin.RouterGroup, deps Dependencies, cfg *Config) {
	auth := v1.Group("/auth")
	auth.Use(apimiddleware.RateLimit(deps.RateLimiterRepository, "auth", cfg.AuthPerMinute, time.Minute, deps.Logger))

	auth.POST("/login", deps.AuthHandler.Login)
	auth.POST("/register-with-company", deps.AuthHandler.RegisterWithCompany)
	auth.POST("/refresh", deps.AuthHandler.Refresh)
	auth.POST("/logout", apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger), deps.AuthHandler.Logout)

	auth.POST("/verify-register-otp", deps.AuthHandler.VerifyRegistionrWithOTP)

	auth.POST("/otp/send", deps.AuthHandler.SendOTP)
	auth.POST("/otp/verify", deps.AuthHandler.VerifyOTP)
}
