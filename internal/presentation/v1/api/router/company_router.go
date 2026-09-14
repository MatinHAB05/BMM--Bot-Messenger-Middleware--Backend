package apirouter

import (
	apimiddleware "messenger-backend/internal/presentation/middleware/api"

	"github.com/gin-gonic/gin"
)

// RegisterCompanyRoutes wires /api/v1/companies/*. All three require a
// valid token and pass through Casbin; the seeded policy set restricts
// update/delete to the "admin" role, and the service layer additionally
// rejects any :id that isn't the caller's own company (see
// CompanyService.Update/Delete) -- there is no cross-company access.
func RegisterCompanyRoutes(v1 *gin.RouterGroup, deps Dependencies) {
	v1.POST("/companies", deps.CompanyHandler.Create)

	companies := v1.Group("/companies")

	companies.Use(apimiddleware.Auth(deps.TokenMaker, deps.AuthnTokenRepository, deps.Logger))
	companies.Use(apimiddleware.RBAC(deps.RBACRepository, deps.Logger))

	companies.GET("/me", deps.CompanyHandler.Me)
	companies.PUT("/:id", deps.CompanyHandler.Update)
	companies.DELETE("/:id", deps.CompanyHandler.Delete)

	companies.POST("/users/send-register-otp", deps.CompanyHandler.SendRegistionrWithOTP)

}
