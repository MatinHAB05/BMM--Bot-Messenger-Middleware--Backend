package apimiddleware

import (
	"github.com/gin-gonic/gin"

	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/domain/tokencontext"
	"messenger-backend/pkg/logger"
)

// RBAC enforces Casbin policy for the current request. It must run after
// Auth (it reads the payload Auth attaches to the context) and uses:
//   - subject: the authenticated user's id (payload.UserID)
//   - domain:  the authenticated user's company id (payload.CompanyID) --
//     this is what makes every permission check tenant-scoped; a
//     policy or role granted in one company has no effect in another.
//   - object:  the route's registered pattern via c.FullPath(), e.g.
//     "/api/v1/users/:id/roles" -- matching literally against
//     policies rather than the concrete request path means a
//     policy for that route covers every :id value with no
//     wildcard/regex matching required in the Casbin model.
//   - action:  the HTTP method
func RBAC(enforcer repository_contract.RBACRepository, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, ok := tokencontext.GetPayload(c)
		if !ok {
			respondError(c, exception.ErrMissingToken)
			return
		}

		obj := c.FullPath()
		act := c.Request.Method

		allowed, err := enforcer.Enforce(payload.UserID, payload.CompanyID, obj, act)
		if err != nil {
			log.Error(err, "casbin enforcement error",
				logger.String("user_id", payload.UserID),
				logger.String("company_id", payload.CompanyID),
				logger.String("obj", obj),
				logger.String("act", act),
			)
			respondError(c, exception.ErrInternal)
			return
		}
		if !allowed {
			respondError(c, exception.ErrForbidden)
			return
		}

		c.Next()
	}
}
