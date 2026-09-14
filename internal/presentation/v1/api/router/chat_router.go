package apirouter

import (
	apimiddleware "messenger-backend/internal/presentation/middleware/api"

	"github.com/gin-gonic/gin"
)

// RegisterChatRoutes wires /api/v1/chats* including the nested
// /:id/history sub-resource. Every route requires a valid token, passes
// through Casbin, and is scoped to the caller's company.
func RegisterChatRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	chats := v1.Group("/chats")
	chats.Use(apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	chats.Use(apimiddleware.RBAC(deps.RBACRepository, deps.Logger))

	chats.GET("", deps.ChatHandler.List)

	chats.GET("/:id", deps.ChatHandler.GetByID)
	chats.PUT("/:id", deps.ChatHandler.Update)
	chats.DELETE("/:id", deps.ChatHandler.Delete)

	// chats.POST("", deps.ChatHandler.Fetch) // chats must already exist(with just adding bot to target chat) but without company
	chats.POST("/otp/send", deps.ChatHandler.SendOTP)

	history := chats.Group("/:id/history")
	history.GET("", deps.ChatHistoryHandler.List)
	history.GET("/:message_id", deps.ChatHistoryHandler.GetByID)
	history.DELETE("/:message_id", deps.ChatHistoryHandler.Delete)
}
