package apimiddleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"messenger-backend/pkg/logger"
)

type RecoveryMiddleware gin.HandlerFunc

// Recovery recovers from any panic in a downstream handler, logs it via
// the structured Zerolog-backed logger (instead of a raw stack dump to
// stdout), and responds with a uniform 500 instead of crashing the
// process. Register this first in the middleware chain.
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error(nil, "recovered from panic",
					logger.Any("panic", r),
					logger.String("path", c.Request.URL.Path),
					logger.String("method", c.Request.Method),
					logger.String("client_ip", c.ClientIP()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"code":    "INTERNAL_ERROR",
					"error":   "internal server error",
				})
			}
		}()
		c.Next()
	}
}
