package middleware

import (
	"github.com/gin-gonic/gin"

	"messenger-backend/internal/domain/exception"
	"messenger-backend/pkg/logger"
)

var errLogger logger.Logger

func InitErrLogger(l logger.Logger) {
	errLogger = l
}

// respondError writes the uniform error envelope and aborts the chain so
// no downstream middleware or handler runs. Shared by every middleware in
// this package to keep the error response shape consistent with the
// handler layer's own (separate, intentionally duplicated per the strict
// per-layer directory boundaries) helper in v1/handler/response.go.
func respondError(c *gin.Context, err *exception.AppError) {
	errLogger.Error(err.Err, err.Message, logger.String("code", err.Code), logger.Bool("is_app_err", true))
	c.AbortWithStatusJSON(err.HTTPStatus, gin.H{
		"success": false,
		"code":    err.Code,
		"error":   err.Message,
	})
}
