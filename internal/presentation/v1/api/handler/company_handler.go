package apihandler

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/tokencontext"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CompanyHandler struct {
	companyService service_contract.CompanyService
}

func NewCompanyHandler(
	companyService service_contract.CompanyService,
) *CompanyHandler {
	return &CompanyHandler{
		companyService: companyService,
	}
}

// POST /api/v1/companies/
func (h *CompanyHandler) Create(c *gin.Context) {
	var req service_contract.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	company, err := h.companyService.Create(c.Request.Context(), req)
	if err != nil {
		fail(c, err)
		return
	}
	success(c, http.StatusOK, company)
}

// Me handles GET /api/v1/companies/me.
func (h *CompanyHandler) Me(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	company, err := h.companyService.Me(c.Request.Context(), companyID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, company)
}

// Update handles PUT /api/v1/companies/:id (admin of that company only --
// the service rejects any :id that isn't the caller's own company).
func (h *CompanyHandler) Update(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	var req service_contract.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	company, err := h.companyService.Update(c.Request.Context(), companyID, targetID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, company)
}

// Delete handles DELETE /api/v1/companies/:id (admin of that company
// only). Soft-deletes/deactivates the company.
func (h *CompanyHandler) Delete(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}
	targetID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.companyService.Delete(c.Request.Context(), companyID, targetID); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "company deactivated"})
}

func (h *CompanyHandler) SendRegistionrWithOTP(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	var req service_contract.SendRegistionrWithOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.companyService.SendRegistionrWithOTP(c.Request.Context(), companyID,req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}
