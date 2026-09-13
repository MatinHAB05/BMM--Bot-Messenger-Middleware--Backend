package handler

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service_contract.AuthService
}

func NewAuthHandler(authService service_contract.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req service_contract.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	tokens, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, tokens)
}

// Login handles POST /api/v1/auth/register-with-company
func (h *AuthHandler) RegisterWithCompany(c *gin.Context) {
	var req service_contract.RegisterWithCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	me, com, err := h.authService.RegisterMeWithCompany(c.Request.Context(), req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, map[string]any{
		"me": me, "company": com,
	})
}

func (h *AuthHandler) VerifyRegistionrWithOTP(c *gin.Context) {
	var req service_contract.VerifyRegisterWithOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	me, com, err := h.authService.RegisterWithOTP(c.Request.Context(), req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, map[string]any{
		"me": me, "company": com,
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req service_contract.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	tokens, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, tokens)
}

// Logout handles POST /api/v1/auth/logout. It sits behind AuthMiddleware,
// which has already verified the token by the time we get here -- but the
// service needs the raw token string (to recompute jti and remaining TTL
// for the blacklist entry), so we re-extract it from the header rather
// than threading it through tokencontext.
func (h *AuthHandler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	fields := strings.Fields(header)
	if len(fields) != 2 {
		fail(c, exception.ErrMissingToken)
		return
	}

	if err := h.authService.Logout(c.Request.Context(), fields[1]); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "logged out"})
}

// SendOTP handles POST /api/v1/auth/otp/send.
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req service_contract.SendAuthOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.authService.SendOTP(c.Request.Context(), req.Identifier, req.Type)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

// VerifyOTP handles POST /api/v1/auth/otp/verify.
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req service_contract.VerifyAuthOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	status, err := h.authService.VerifyOTP(c.Request.Context(), req.Identifier, req.Type, req.Code)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, map[string]any{
		"status": status,
	})
}
