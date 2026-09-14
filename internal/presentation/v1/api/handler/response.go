package apihandler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"messenger-backend/internal/domain/exception"
	"messenger-backend/pkg/logger"
)

var errLogger logger.Logger

func InitErrLogger(l logger.Logger) {
	errLogger = l
}

// success writes the uniform success envelope used by every handler.
func success(c *gin.Context, status int, data interface{}) {
	errLogger.Info("success", logger.Any("data", data))
	c.JSON(status, gin.H{
		"success": true,
		"data":    data,
	})
}

// fail writes the uniform error envelope derived from an
// *exception.AppError. Any other error type is treated as an unexpected
// internal failure so its details are never leaked to the client.
func fail(c *gin.Context, err error) {
	appErr, ok := err.(*exception.AppError)
	if !ok {
		appErr = exception.ErrInternal

	}
	errLogger.Error(appErr.Err, appErr.Message, logger.String("code", appErr.Code), logger.Bool("is_app_err", ok))

	c.JSON(appErr.HTTPStatus, gin.H{
		"success":  false,
		"code":     appErr.Code,
		"error":    appErr.Message,
		"ok_debug": ok,
	})
}

// parseIDParam parses a uint route parameter (e.g. :id, :message_id),
// wrapping a bad value as ErrBadRequest rather than letting a raw
// strconv error leak to the client.
func parseIDParam(c *gin.Context, name string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, exception.Wrap(exception.ErrBadRequest, err)
	}
	return uint(id), nil
}

// atoiOrDefault parses a query parameter as a positive int, falling back
// to fallback on any parse error or non-positive value. Used for page/
// page_size query params, which should never hard-fail a request.
func atoiOrDefault(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func parseUint(s string) (uint, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
