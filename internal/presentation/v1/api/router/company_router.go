package apirouter

import (
	apimiddleware "messenger-backend/internal/presentation/middleware/api"

	"github.com/gin-gonic/gin"
)

// RegisterCompanyRoutes  /api/v1/companies/*
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
