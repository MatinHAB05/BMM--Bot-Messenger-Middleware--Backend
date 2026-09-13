package handler

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/tokencontext"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService service_contract.UserService
}

func NewUserHandler(userService service_contract.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Me handles GET /api/v1/users/me.
func (h *UserHandler) Me(c *gin.Context) {
	payload, ok := tokencontext.GetPayload(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	user, err := h.userService.Me(c.Request.Context(), companyID, payload.UserID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}

// List handles GET /api/v1/users (paginated via ?page & ?page_size,
// scoped to the caller's company).
func (h *UserHandler) List(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}

	users, err := h.userService.List(c.Request.Context(), companyID, page, pageSize)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, users)
}

// GetByID handles GET /api/v1/users/:id (scoped to the caller's company).
func (h *UserHandler) GetByID(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), companyID, c.Param("id"))
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}

/*Create ?!??!
// Create handles POST /api/v1/users (admin only, within the caller's
// company).
func (h *UserHandler) Create(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	user, err := h.userService.Create(c.Request.Context(), companyID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusCreated, user)
}
*/

// Update handles PUT /api/v1/users/:id -- profile/status changes within
// the caller's company. Role changes go through UpdateRoles instead.
func (h *UserHandler) Update(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	user, err := h.userService.Update(c.Request.Context(), companyID, c.Param("id"), req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}

// UpdateRoles handles PUT /api/v1/users/:id/roles (admin only) and
// synchronizes the change straight into the Casbin policy store, scoped
// to the caller's company domain.
func (h *UserHandler) UpdateRoles(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.UpdateUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	user, err := h.userService.UpdateRoles(c.Request.Context(), companyID, c.Param("id"), req.Roles)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}

// Delete handles DELETE /api/v1/users/:id (admin only, within the
// caller's company). Soft-deletes the user (see the User entity's
// gorm.DeletedAt field).
func (h *UserHandler) Delete(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	if err := h.userService.Delete(c.Request.Context(), companyID, c.Param("id")); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "user deactivated"})
}

// UpdateRoles handles PUT /api/v1/users/:id/roles (admin only) and
// synchronizes the change straight into the Casbin policy store, scoped
// to the caller's company domain.
func (h *UserHandler) UpdateEmail(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.UpdateUserEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	user, err := h.userService.UpdateEmail(c.Request.Context(), companyID, userID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}

// UpdateRoles handles PUT /api/v1/users/:id/roles (admin only) and
// synchronizes the change straight into the Casbin policy store, scoped
// to the caller's company domain.
func (h *UserHandler) UpdatePhone(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.UpdateUserPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	user, err := h.userService.UpdatePhone(c.Request.Context(), companyID, userID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}

func (h *UserHandler) UpdateUsername(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.UpdateUserUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	userID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	user, err := h.userService.UpdateUsername(c.Request.Context(), companyID, userID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, user)
}
