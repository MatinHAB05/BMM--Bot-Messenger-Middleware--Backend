package router

import (
	"github.com/gin-gonic/gin"

	"messenger-backend/internal/presentation/middleware"
)

// RegisterUserRoutes wires /api/v1/users*. Every route requires a valid
// access token and passes through Casbin; the seeded policy set (see
// internal/infrastructure/seed) further restricts list/create/update/
// update-roles/delete to the "admin" role while "me" is open to any
// authenticated role. Every handler scopes to the caller's own company.
func RegisterUserRoutes(v1 *gin.RouterGroup, deps Dependencies, cfg *Config) {
	users := v1.Group("/users")
	users.Use(middleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	users.Use(middleware.RBAC(deps.RBACRepository, deps.Logger))

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
