package apihandler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// maxBroadcastAttachmentSize bounds a single broadcast attachment upload.
// 50MB matches Telegram's own general ceiling for bot-uploaded files;
// adjust if your Bale integration (or Telegram's per-type limits: e.g.
// 1MB for a valid .ogg voice note) needs something tighter.
const maxBroadcastAttachmentSize = 50 << 20

// Send handles POST /api/v1/broadcast. The service fans the message out
// to every requested platform concurrently and only returns once all
// deliveries have finished, so the HTTP response itself is the execution
// status -- there is no separate polling endpoint.
//
// Two request shapes are accepted:
//   - application/json: {"message": "...", "platforms": [...]} -- plain
//     text broadcast, unchanged from before.
//   - multipart/form-data, when an attachment is included:
//     message          (optional; used as the caption)
//     platforms        (repeat the field once per platform:
//     platforms=telegram&platforms=bale)
//     attachment       the file itself
//     attachment_type  one of photo|video|voice|document|animation
//     attachment_name  optional; defaults to the uploaded file's name
func (h *BroadcastHandler) Send(c *gin.Context) {
	var req service_contract.BroadcastRequest

	if isMultipart(c) {
		parsed, err := parseBroadcastMultipart(c)
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
// form. The "attachment" field is optional -- its absence just yields a
// text-only request, same as the JSON path.
func parseBroadcastMultipart(c *gin.Context) (service_contract.BroadcastRequest, error) {
	req := service_contract.BroadcastRequest{
		Message:   c.PostForm("message"),
		Platforms: c.PostFormArray("platforms"),
	}

	fileHeader, err := c.FormFile("attachment")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return req, nil
		}
		return req, fmt.Errorf("read attachment: %w", err)
	}

	if fileHeader.Size > maxBroadcastAttachmentSize {
		return req, fmt.Errorf("attachment exceeds %d bytes", maxBroadcastAttachmentSize)
	}

	attachmentType := service_contract.BroadcastAttachmentType(c.PostForm("attachment_type"))
	switch attachmentType {
	case service_contract.BroadcastAttachmentPhoto,
		service_contract.BroadcastAttachmentVideo,
		service_contract.BroadcastAttachmentVoice,
		service_contract.BroadcastAttachmentDocument,
		service_contract.BroadcastAttachmentAnimation:
	default:
		return req, fmt.Errorf("attachment_type must be one of photo|video|voice|document|animation, got %q", attachmentType)
	}

	f, err := fileHeader.Open()
	if err != nil {
		return req, fmt.Errorf("open attachment: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return req, fmt.Errorf("read attachment bytes: %w", err)
	}

	fileName := c.PostForm("attachment_name")
	if fileName == "" {
		fileName = fileHeader.Filename
	}

	req.Attachment = &service_contract.BroadcastAttachment{
		Type:     attachmentType,
		FileName: fileName,
		Data:     data,
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
