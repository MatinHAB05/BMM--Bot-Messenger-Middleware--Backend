package apihandler

import (
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/tokencontext"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ChatHistoryHandler struct {
	chatHistoryService service_contract.ChatHistoryService
}

func NewChatHistoryHandler(chatHistoryService service_contract.ChatHistoryService) *ChatHistoryHandler {
	return &ChatHistoryHandler{chatHistoryService: chatHistoryService}
}

// List handles GET /api/v1/chats/:id/history -- paginated message
// history, filterable by search (text query against Content),
// media_type, from_date, and to_date.
func (h *ChatHistoryHandler) List(c *gin.Context) {
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

	query := service_contract.ChatHistoryListQuery{
		Search:    c.Query("search"),
		MediaType: c.Query("media_type"),
		FromDate:  parseDateQuery(c, "from_date"),
		ToDate:    parseDateQuery(c, "to_date"),
		Page:      atoiOrDefault(c.DefaultQuery("page", "1"), 1),
		PageSize:  atoiOrDefault(c.DefaultQuery("limit", "20"), 20),
	}

	messages, err := h.chatHistoryService.List(c.Request.Context(), companyID, chatID, query)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, messages)
}

// GetByID handles GET /api/v1/chats/:id/history/:message_id -- includes
// the message's RawPayload, unlike the list endpoint above.
func (h *ChatHistoryHandler) GetByID(c *gin.Context) {
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
	messageID, err := parseIDParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	message, err := h.chatHistoryService.GetByIDInCompany(c.Request.Context(), companyID, chatID, messageID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, message)
}

// Delete handles DELETE /api/v1/chats/:id/history/:message_id --
// soft-deletes a single message record.
func (h *ChatHistoryHandler) Delete(c *gin.Context) {
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
	messageID, err := parseIDParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.chatHistoryService.Delete(c.Request.Context(), companyID, chatID, messageID); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "chat history message deleted"})
}

// parseDateQuery accepts either a full RFC3339 timestamp or a bare
// YYYY-MM-DD date for from_date/to_date, returning nil (no filter,
// silently ignored) rather than failing the request on a malformed value.
func parseDateQuery(c *gin.Context, name string) *time.Time {
	raw := c.Query(name)
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return &t
	}
	return nil
}
