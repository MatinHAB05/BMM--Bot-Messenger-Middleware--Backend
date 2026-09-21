package apirouter

import (
	apimiddleware "messenger-backend/internal/presentation/middleware/api"

	"github.com/gin-gonic/gin"
)

// RegisterAttachmentRoutes wires /api/v1/attachments* plus the nested
// .../chats/:id/history/:message_id/attachments* sub-resource used for
// creating/listing/deleting a single message's attachments. Every route
// requires a valid token and passes through RBAC, matching
// RegisterChatRoutes' pattern exactly.
//
// deps.AttachmentHandler is a new field this adds to the project's
// Dependencies struct (see INTEGRATION_NOTES.md) -- everything else here
// only uses fields RegisterChatRoutes already relies on.
func RegisterAttachmentRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	attachments := v1.Group("/attachments")
	attachments.Use(apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	attachments.Use(apimiddleware.RBAC(deps.RBACRepository, deps.Logger))

	attachments.GET("", deps.AttachmentHandler.ListAttachments)
	attachments.POST("/by-messages", deps.AttachmentHandler.GetAttachmentsByChatHistoryIDsBatch)
	attachments.GET("/by-platform-file/:platform_file_id", deps.AttachmentHandler.GetAttachmentByPlatformFileID)
	attachments.DELETE("/batch", deps.AttachmentHandler.DeleteAttachmentsByIDsBatch)

	attachments.GET("/:id", deps.AttachmentHandler.GetAttachmentByID)
	attachments.PUT("/:id", deps.AttachmentHandler.UpdateAttachment)
	attachments.PATCH("/:id/thumbnail", deps.AttachmentHandler.UpdateThumbnailID)
	attachments.DELETE("/:id", deps.AttachmentHandler.DeleteAttachmentByID)
	attachments.POST("/:id/restore", deps.AttachmentHandler.RestoreAttachmentByID)

	attachments.POST("/:id/links/download", deps.BatchGetDownloadLinksAttachmentsByAttachmentID)
	attachments.POST("/:id/links/download/batch", deps.BatchGetDownloadLinksAttachmentsByAttachmentID)

	// Nested under a single message (chat history row): creating and
	// bulk-reading/deleting attachments in the context of "this message".
	messageAttachments := v1.Group("/chats/:id/history/:message_id/attachments")
	messageAttachments.Use(apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	messageAttachments.Use(apimiddleware.RBAC(deps.RBACRepository, deps.Logger))

	messageAttachments.POST("", deps.AttachmentHandler.CreateAttachment)

	messageAttachments.POST("/links/download", deps.AttachmentHandler.GetDownloadLinksAttachmentsByMessageID)
	messageAttachments.POST("links/download/batch", deps.AttachmentHandler.BatchGetDownloadLinksAttachmentsByMessageID)

	messageAttachments.POST("/batch", deps.AttachmentHandler.CreateAttachmentsBatch)
	messageAttachments.GET("", deps.AttachmentHandler.GetAttachmentsByMessageID)
	messageAttachments.DELETE("", deps.AttachmentHandler.DeleteAttachmentsByMessageID)
}
