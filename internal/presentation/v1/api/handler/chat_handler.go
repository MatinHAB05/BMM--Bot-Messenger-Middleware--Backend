package apihandler

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/tokencontext"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService    service_contract.ChatService
	companyService service_contract.CompanyService
}

func NewChatHandler(
	chatService service_contract.ChatService,
	companyService service_contract.CompanyService,
) *ChatHandler {
	return &ChatHandler{
		chatService:    chatService,
		companyService: companyService,
	}
}

// List handles GET /api/v1/chats -- paginated, filterable by platform,
// chat_type, and is_active, scoped to the caller's company.
func (h *ChatHandler) List(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	query := service_contract.ChatListQuery{
		Platform: c.Query("platform"),
		ChatType: c.Query("chat_type"),
		Page:     atoiOrDefault(c.DefaultQuery("page", "1"), 1),
		PageSize: atoiOrDefault(c.DefaultQuery("page_size", "20"), 20),
	}
	if v := c.Query("is_active"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			query.IsActive = &b
		}
	}

	chats, err := h.chatService.List(c.Request.Context(), companyID, query)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, chats)
}

// GetByID handles GET /api/v1/chats/:id.
func (h *ChatHandler) GetByID(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}
	chatID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	chat, err := h.chatService.GetByIDInCompany(c.Request.Context(), companyID, chatID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, chat)
}

// // Create handles POST /api/v1/chats -- manually registering/linking an
// // existing platform chat to the caller's company.
// func (h *ChatHandler) Fetch(c *gin.Context) {
// 	companyID, ok := tokencontext.GetCompanyID(c)
// 	if !ok {
// 		fail(c, exception.ErrMissingToken)
// 		return
// 	}

// 	var req service_contract.CreateChatRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		fail(c, exception.Wrap(exception.ErrBadRequest, err))
// 		return
// 	}

// 	chat, err := h.chatService.Create(c.Request.Context(), companyID, req)
// 	if err != nil {
// 		fail(c, err)
// 		return
// 	}

// 	success(c, http.StatusCreated, chat)
// }

// Update handles PUT /api/v1/chats/:id.
func (h *ChatHandler) Update(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}
	chatID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	var req service_contract.UpdateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	chat, err := h.chatService.Update(c.Request.Context(), companyID, chatID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, chat)
}

// Delete handles DELETE /api/v1/chats/:id. Soft-deletes/deactivates the
// chat.
func (h *ChatHandler) Delete(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}
	chatID, err := parseIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.chatService.Delete(c.Request.Context(), companyID, chatID); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "chat deactivated"})
}

// POST /api/v1/chats/otp/send.
func (h *ChatHandler) SendOTP(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	res, err := h.chatService.SendOTP(c.Request.Context(), companyID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, res)
}
