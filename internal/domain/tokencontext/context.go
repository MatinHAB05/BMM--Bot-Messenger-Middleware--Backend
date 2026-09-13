// Package tokencontext provides small, typed helpers for stashing the
// verified PASETO payload on a *gin.Context so downstream middleware and
// handlers never touch Gin's untyped key/value store directly.
package tokencontext

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"messenger-backend/internal/domain/paseto"
)

const payloadKey = "auth_payload"

// SetPayload stores the verified token payload on the Gin context.
// Called by AuthMiddleware once a token has passed signature and
// blacklist checks.
func SetPayload(c *gin.Context, payload *paseto.Payload) {
	c.Set(payloadKey, payload)
}

// GetPayload retrieves the verified token payload from the Gin context.
// The second return value is false if AuthMiddleware has not run (or
// failed) for this request.
func GetPayload(c *gin.Context) (*paseto.Payload, bool) {
	value, exists := c.Get(payloadKey)
	if !exists {
		return nil, false
	}
	payload, ok := value.(*paseto.Payload)
	return payload, ok
}

// GetCompanyID is the single place that turns the authenticated request's
// tenant (payload.CompanyID, a string inside the token) into the uint
// every repository method expects. Handlers call this instead of parsing
// payload.CompanyID themselves, so company-scoping is applied the same
// way everywhere -- this is the "multitenancy context" extraction point:
// every company-scoped query in the system traces back to this call.
func GetCompanyID(c *gin.Context) (uint, bool) {
	payload, ok := GetPayload(c)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(payload.CompanyID, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}
