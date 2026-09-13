package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/tokencontext"
)

type BroadcastHandler struct {
	broadcastService service_contract.BroadcastService
}

func NewBroadcastHandler(broadcastService service_contract.BroadcastService) *BroadcastHandler {
	return &BroadcastHandler{broadcastService: broadcastService}
}

// Broadcast handles POST /api/v1/broadcast. The service fans the message
// out to every requested platform concurrently and only returns once all
// deliveries have finished, so the HTTP response itself is the execution
// status -- there is no separate polling endpoint.
func (h *BroadcastHandler) Broadcast(c *gin.Context) {
	var req service_contract.BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	payload, ok := tokencontext.GetPayload(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	company_id, err := parseUint(payload.CompanyID)
	if err != nil {
		fail(c, err)
		return
	}
	result, err := h.broadcastService.Broadcast(c.Request.Context(), company_id, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, result)
}
