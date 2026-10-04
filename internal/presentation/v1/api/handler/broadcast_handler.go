package apihandler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/tokencontext"
)

type BroadcastHandlerConfig struct {
	MaxBroadcastAttachmentFileSize int
	MaxBroadcastAttachmentFiles    int
}

type BroadcastHandler struct {
	broadcastService service_contract.BroadcastService

	cfg *BroadcastHandlerConfig
}

func NewBroadcastHandler(
	broadcastService service_contract.BroadcastService,
	cfg *BroadcastHandlerConfig,
) *BroadcastHandler {
	return &BroadcastHandler{broadcastService: broadcastService, cfg: cfg}
}

// Send handles POST /api/v1/broadcast. The service fans the message out
// to every requested platform concurrently and only returns once all
// deliveries have finished, so the HTTP response itself is the execution
// status -- there is no separate polling endpoint.
//
// Two request shapes are accepted:
//   - application/json: {"message": "...", "platforms": [...]} -- plain
//     text broadcast, unchanged from before.
//   - multipart/form-data, when attachment(s) are included:
//     message          (optional; used as the caption on the first file)
//     platforms        (repeat the field once per platform:
//     platforms=telegram&platforms=bale)
//     attachment        the file(s) -- repeat this field for more than
//     one (e.g. 5 photos); all files in one request
//     must be the same attachment_type
//     attachment_type  one of photo|video|voice|document|animation,
//     applies to every "attachment" file in the request
func (h *BroadcastHandler) Send(c *gin.Context) {
	var req service_contract.BroadcastRequest

	if isMultipart(c) {
		parsed, err := h.parseBroadcastMultipart(c)
		if err != nil {
			fail(c, exception.Wrap(exception.ErrBadRequest, err))
			return
		}
		req = parsed
	} else if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	if req.Message == "" && req.Attachment == nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, fmt.Errorf("message or attachment is required")))
		return
	}
	if len(req.Platforms) == 0 {
		fail(c, exception.Wrap(exception.ErrBadRequest, fmt.Errorf("platforms is required")))
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

func isMultipart(c *gin.Context) bool {
	return strings.HasPrefix(c.ContentType(), "multipart/form-data")
}

// parseBroadcastMultipart builds a BroadcastRequest from a multipart
// form. "attachment" is optional and repeatable -- its absence yields a
// text-only request, same as the JSON path; every repetition is treated
// as one more file of the single attachment_type given.
func (h *BroadcastHandler) parseBroadcastMultipart(c *gin.Context) (service_contract.BroadcastRequest, error) {
	req := service_contract.BroadcastRequest{
		Message:   c.PostForm("message"),
		Platforms: c.PostFormArray("platforms"),
	}

	form, err := c.MultipartForm()
	if err != nil {
		return req, fmt.Errorf("parse multipart form: %w", err)
	}

	fileHeaders := form.File["attachment"]
	if len(fileHeaders) == 0 {
		return req, nil
	}
	// ? -1 == no limitation
	if h.cfg.MaxBroadcastAttachmentFiles != -1 && h.cfg.MaxBroadcastAttachmentFileSize != -1 && len(fileHeaders) > h.cfg.MaxBroadcastAttachmentFileSize {
		return req, fmt.Errorf("at most %d attachment files are allowed per broadcast, got %d", h.cfg.MaxBroadcastAttachmentFileSize, len(fileHeaders))
	}

	attachmentType := service_contract.BroadcastAttachmentType(c.PostForm("attachment_type"))
	switch attachmentType {
	case service_contract.BroadcastAttachmentPhoto,
		service_contract.BroadcastAttachmentVideo,
		service_contract.BroadcastAttachmentVoice,
		service_contract.BroadcastAttachmentAudio,
		service_contract.BroadcastAttachmentDocument,
		service_contract.BroadcastAttachmentAnimation:
	default:
		return req, fmt.Errorf("attachment_type must be one of photo|video|voice|document|animation, got %q", attachmentType)
	}

	files := make([]service_contract.BroadcastAttachmentFile, 0, len(fileHeaders))
	for _, fh := range fileHeaders {
		if fh.Size > int64(h.cfg.MaxBroadcastAttachmentFileSize) {
			return req, fmt.Errorf("attachment %q exceeds %d bytes", fh.Filename, int64(h.cfg.MaxBroadcastAttachmentFileSize))
		}

		f, err := fh.Open()
		if err != nil {
			return req, fmt.Errorf("open attachment %q: %w", fh.Filename, err)
		}
		data, readErr := io.ReadAll(f)
		f.Close()
		if readErr != nil {
			return req, fmt.Errorf("read attachment %q: %w", fh.Filename, readErr)
		}

		files = append(files, service_contract.BroadcastAttachmentFile{
			FileName:    fh.Filename,
			ContentType: fh.Header.Get("Content-Type"),
			Data:        data,
		})
	}

	req.Attachment = &service_contract.BroadcastAttachment{
		Type:  attachmentType,
		Files: files,
	}

	return req, nil
}

// Delete handles DELETE /api/v1/broadcast/:id
func (h *BroadcastHandler) Delete(c *gin.Context) {
	var req service_contract.DeleteBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	broadcasgMsgUUID, err := parseUUIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.broadcastService.DeleteBroadcast(c.Request.Context(), companyID, broadcasgMsgUUID, req); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "chat delete broadcast message deleted"})
}

// Delete handles DELETE /api/v1/broadcast/batch
func (h *BroadcastHandler) DeleteBatch(c *gin.Context) {
	var req service_contract.DeleteBroadcastBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	if err := h.broadcastService.DeleteBroadcastBatch(c.Request.Context(), companyID, req); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "chat delete broadcast message deleted"})
}

// Update /api/v1/broadcast/:id
func (h *BroadcastHandler) Update(c *gin.Context) {
	var req service_contract.UpdateBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, exception.Wrap(exception.ErrBadRequest, err))
		return
	}

	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	broadcasgMsgUUID, err := parseUUIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	if err := h.broadcastService.UpdateBroadcast(c.Request.Context(), companyID, broadcasgMsgUUID, req); err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"message": "chat updae broadcast message updated"})
}

// GetPlatforms handles GET /api/v1/broadcast/:id/platforms.
// It returns the distinct platforms the broadcast was actually sent to,
// so the frontend knows exactly which platforms to target on edit/delete.
func (h *BroadcastHandler) GetPlatforms(c *gin.Context) {
	companyID, ok := tokencontext.GetCompanyID(c)
	if !ok {
		fail(c, exception.ErrMissingToken)
		return
	}

	broadcastMsgUUID, err := parseUUIDParam(c, "id")
	if err != nil {
		fail(c, err)
		return
	}

	platforms, err := h.broadcastService.GetBroadcastPlatforms(c.Request.Context(), companyID, broadcastMsgUUID)
	if err != nil {
		fail(c, err)
		return
	}

	success(c, http.StatusOK, gin.H{"platforms": platforms})
}

