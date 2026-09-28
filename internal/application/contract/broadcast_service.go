package service_contract

import (
	"context"

	"github.com/google/uuid"
)

// BroadcastAttachmentType enumerates the media kinds a broadcast request
// can attach. Mirrors messenger.AttachmentType, kept as its own type here
// (rather than importing pkg/messenger) so the contract package stays
// free of infrastructure-level dependencies.
type BroadcastAttachmentType string

const (
	BroadcastAttachmentPhoto     BroadcastAttachmentType = "photo"
	BroadcastAttachmentVideo     BroadcastAttachmentType = "video"
	BroadcastAttachmentVoice     BroadcastAttachmentType = "voice"
	BroadcastAttachmentDocument  BroadcastAttachmentType = "document"
	BroadcastAttachmentAnimation BroadcastAttachmentType = "animation"
	BroadcastAttachmentAudio     BroadcastAttachmentType = "audio"
)

// BroadcastAttachmentFile is a single uploaded file within a broadcast's
// attachment batch.
type BroadcastAttachmentFile struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

// BroadcastAttachment is the attachment batch a caller adds to a
// Broadcast request: one Type shared by every file in Files (e.g. 5
// photos, or 3 videos -- never a mix of photo+video in one request).
// Each file's Data is fully buffered (not streamed): Broadcast fans the
// same files out to every target chat concurrently, and each target
// needs an independent read over the same bytes.
type BroadcastAttachment struct {
	Type  BroadcastAttachmentType   `json:"type"`
	Files []BroadcastAttachmentFile `json:"files"`
}

type BroadcastRequest struct {
	// Message is the text to send, or the caption (applied to the first
	// file only when Attachment has several) when Attachment is set. At
	// least one of Message/Attachment must be present -- enforced by the
	// handler/service, not by a binding tag, since Attachment never
	// arrives via JSON binding (see BroadcastHandler.Send).
	Message   string   `json:"message" form:"message"`
	Platforms []string `json:"platforms" form:"platforms" binding:"required"`

	// Attachment is populated by the handler from a multipart/form-data
	// request; it is never bound from JSON.
	Attachment *BroadcastAttachment `json:"-"`
}

type DeleteBroadcastRequest struct {
	Platforms []string `json:"platforms" binding:"required"`
}

// DeleteBroadcastBatchRequest deletes several broadcasts (each identified
// by its own UUID) in one call, all restricted to the same Platforms set.
type DeleteBroadcastBatchRequest struct {
	Platforms    []string    `json:"platforms" binding:"required"`
	BroadcastIDS []uuid.UUID `json:"brodcast_ids"  binding:"required"`
}

type UpdateBroadcastRequest struct {
	Platforms []string             `json:"platforms" binding:"required"`
	NewMessge UpdateChatHisRequest `json:"message" binding:"required"`
}

type BroadcastResult struct {
	Platform string   `json:"platform"`
	Success  bool     `json:"success"`
	Error    []string `json:"error,omitempty"`
}

type BroadcastTarget struct {
	Name      string `json:"name"`
	Platform  string `json:"platform"`
	IsPrivate bool   `json:"is_private"`
	Type      string `json:"type"`
}

type BroadcastResponse struct {
	Targets []BroadcastTarget `json:"targets"`
	Results []BroadcastResult `json:"results"`
}

// BroadcastService fans messages out to configured messenger clients concurrently.
type BroadcastService interface {
	Broadcast(ctx context.Context, companyID uint, req BroadcastRequest) (*BroadcastResponse, error)
	DeleteBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req DeleteBroadcastRequest) error
	DeleteBroadcastBatch(ctx context.Context, companyID uint, req DeleteBroadcastBatchRequest) error
	UpdateBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req UpdateBroadcastRequest) error
}
