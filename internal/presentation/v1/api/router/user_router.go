package apirouter

import (
	"github.com/gin-gonic/gin"

	apimiddleware "messenger-backend/internal/presentation/middleware/api"
)

// RegisterUserRoutes  /api/v1/users*
func RegisterUserRoutes(v1 *gin.RouterGroup, deps Dependencies, cfg *Config) {
	users := v1.Group("/users")
	users.Use(apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	users.Use(apimiddleware.RBAC(deps.RBACRepository, deps.Logger))

	users.GET("/me", deps.UserHandler.Me)
	users.GET("", deps.UserHandler.List)
	users.GET("/:id", deps.UserHandler.GetByID)
	// users.POST("", deps.UserHandler.Create) ?!??!
	users.PUT("/:id", deps.UserHandler.Update)
	users.PUT("/:id/roles", deps.UserHandler.UpdateRoles)
	users.PUT("/:id/email", deps.UserHandler.UpdateEmail)
	users.PUT("/:id/phone", deps.UserHandler.UpdatePhone)
	users.PUT("/:id/username", deps.UserHandler.UpdateUsername)
	// Registered as /api/v1/users/:id (not the spec text's singular
	// "/user/:id") for consistency with the rest of this route group --
	// see the doc comment on UserHandler.Delete.
	users.DELETE("/:id", deps.UserHandler.Delete)
}
