package apihandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
)

// AttachmentHandler is a pure HTTP adapter: it binds/validates requests,
// calls AttachmentService, and shapes the response -- it never touches a
// repository or a domain entity directly (Attachment's own entity type
// never appears anywhere in this file).
type AttachmentHandler struct {
	attachmentService service_contract.AttachmentService
}

func NewAttachmentHandler(attachmentService service_contract.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{attachmentService: attachmentService}
}

// --- Creation ---

// CreateAttachment handles POST .../history/:message_id/attachments.
func (h *AttachmentHandler) CreateAttachment(c *gin.Context) {
	chatHistoryID, err := parseUintParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	var req service_contract.CreateAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.attachmentService.CreateAttachment(c.Request.Context(), chatHistoryID, req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusCreated, resp)
}

// createAttachmentsBatchRequest wraps the batch body -- a bare JSON array
// as the whole body is ambiguous with future extension (e.g. adding
// options alongside the list later), so this is a small object instead.
type createAttachmentsBatchRequest struct {
	Attachments []service_contract.CreateAttachmentRequest `json:"attachments" binding:"required,min=1"`
}

// CreateAttachmentsBatch handles POST .../history/:message_id/attachments/batch.
func (h *AttachmentHandler) CreateAttachmentsBatch(c *gin.Context) {
	chatHistoryID, err := parseUintParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	var req createAttachmentsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.attachmentService.CreateAttachmentsBatch(c.Request.Context(), chatHistoryID, req.Attachments)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusCreated, resp)
}

// --- Query & retrieval ---

// GetAttachmentByID handles GET /attachments/:id.
func (h *AttachmentHandler) GetAttachmentByID(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	resp, err := h.attachmentService.GetAttachmentByID(c.Request.Context(), id)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

// GetAttachmentsByMessageID handles GET .../history/:message_id/attachments.
func (h *AttachmentHandler) GetAttachmentsByMessageID(c *gin.Context) {
	chatHistoryID, err := parseUintParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	resp, err := h.attachmentService.GetAttachmentsByMessageID(c.Request.Context(), chatHistoryID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

func (h *AttachmentHandler) GetDownloadLinksAttachmentsByMessageID(c *gin.Context) {
	chatHistoryID, err := parseUintParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	resp, err := h.attachmentService.GetAttachmentDownloadURLByMessageID(c.Request.Context(), chatHistoryID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

func (h *AttachmentHandler) BatchGetDownloadLinksAttachmentsByMessageID(c *gin.Context) {
	var req getByChatHistoryIDsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.attachmentService.GetAttachmentsDownloadURLsBatchByMessageIDs(c.Request.Context(), req.ChatHistoryIDs)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)

}

func (h *AttachmentHandler) GetDownloadLinksAttachmentsByAttachmentID(c *gin.Context) {
	attachID, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	resp, err := h.attachmentService.GetAttachmentDownloadURL(c.Request.Context(), attachID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

func (h *AttachmentHandler) BatchGetDownloadLinksAttachmentsByAttachmentID(c *gin.Context) {
	var req getByAttachIDsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.attachmentService.GetAttachmentsDownloadURLsBatch(c.Request.Context(), req.AttachIDs)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)

}

// getByChatHistoryIDsBatchRequest is the body for the batch-by-message
// lookup -- a POST-with-body rather than repeated query params, since an
// arbitrarily long id list is unwieldy (and easy to get wrong) as a query
// string.
type getByChatHistoryIDsBatchRequest struct {
	ChatHistoryIDs []uint `json:"chat_history_ids" binding:"required,min=1"`
}

type getByAttachIDsBatchRequest struct {
	AttachIDs []uint `json:"attachment_ids" binding:"required,min=1"`
}

// GetAttachmentsByChatHistoryIDsBatch handles POST /attachments/by-messages.
func (h *AttachmentHandler) GetAttachmentsByChatHistoryIDsBatch(c *gin.Context) {
	var req getByChatHistoryIDsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	resp, err := h.attachmentService.GetAttachmentsByChatHistoryIDsBatch(c.Request.Context(), req.ChatHistoryIDs)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

// GetAttachmentByPlatformFileID handles GET /attachments/by-platform-file/:platform_file_id.
func (h *AttachmentHandler) GetAttachmentByPlatformFileID(c *gin.Context) {
	platformFileID := c.Param("platform_file_id")
	if platformFileID == "" {
		fail(c, exception.ErrBadRequest)
		return
	}

	resp, err := h.attachmentService.GetAttachmentByPlatformFileID(c.Request.Context(), platformFileID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

// ListAttachments handles GET /attachments?limit=&offset=&sort=&file_type=.
func (h *AttachmentHandler) ListAttachments(c *gin.Context) {
	query := service_contract.AttachmentListQuery{
		Limit:    atoiOrDefault(c.DefaultQuery("limit", "20"), 20),
		Offset:   atoiOrDefault(c.DefaultQuery("offset", "0"), 0),
		Sort:     strings.TrimSpace(c.Query("sort")),
		FileType: strings.TrimSpace(c.Query("file_type")),
	}

	resp, err := h.attachmentService.ListAttachments(c.Request.Context(), query)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

// --- Update ---

// UpdateAttachment handles PUT /attachments/:id.
func (h *AttachmentHandler) UpdateAttachment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	var req service_contract.UpdateAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}
	req.ID = id // the path id is authoritative, never trust a body-supplied one

	resp, err := h.attachmentService.UpdateAttachment(c.Request.Context(), req)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

type updateThumbnailRequest struct {
	ThumbnailPlatformFileID string `json:"thumbnail_platform_file_id" binding:"required"`
}

// UpdateThumbnailID handles PATCH /attachments/:id/thumbnail.
func (h *AttachmentHandler) UpdateThumbnailID(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	var req updateThumbnailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	if err := h.attachmentService.UpdateThumbnailID(c.Request.Context(), id, req.ThumbnailPlatformFileID); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "thumbnail updated"})
}

// --- Delete & restore ---

// DeleteAttachmentByID handles DELETE /attachments/:id.
func (h *AttachmentHandler) DeleteAttachmentByID(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.attachmentService.DeleteAttachment(c.Request.Context(), id); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "attachment deleted"})
}

type deleteByIDsBatchRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// DeleteAttachmentsByIDsBatch handles DELETE /attachments/batch.
func (h *AttachmentHandler) DeleteAttachmentsByIDsBatch(c *gin.Context) {
	var req deleteByIDsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	if err := h.attachmentService.DeleteAttachmentsByIDs(c.Request.Context(), req.IDs); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "attachments deleted", "count": len(req.IDs)})
}

// DeleteAttachmentsByMessageID handles DELETE .../history/:message_id/attachments.
func (h *AttachmentHandler) DeleteAttachmentsByMessageID(c *gin.Context) {
	chatHistoryID, err := parseUintParam(c, "message_id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.attachmentService.DeleteAttachmentsByMessageID(c.Request.Context(), chatHistoryID); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "attachments deleted for message"})
}

// RestoreAttachmentByID handles POST /attachments/:id/restore.
func (h *AttachmentHandler) RestoreAttachmentByID(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	resp, err := h.attachmentService.RestoreAttachment(c.Request.Context(), id)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, resp)
}

// --- shared helpers ---

func parseUintParam(c *gin.Context, name string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, exception.Wrap(exception.ErrBadRequest, err)
	}
	return uint(id), nil
}
