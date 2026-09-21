package service_contract

import (
	"context"
	"mime/multipart"
	"time"

	"messenger-backend/internal/domain/entity"
)

// ==========================================
// 1. DTOs
// ==========================================

// CreateAttachmentRequest is the input for creating a single Attachment.
// The owning ChatHistory is always supplied separately (a chatHistoryID
// parameter, or implicitly via IngestMessageWithAttachmentsRequest) rather
// than on this DTO, since the caller always already knows it.
//
// StoragePath/ThumbnailStoragePath are optional: set them when the
// caller has ALREADY uploaded the bytes to object storage before calling
// (the Telegram ingestion flow does this -- fetch from Telegram, upload
// to MinIO, then build this request with the resulting object key).
// Leave both blank for a request that only records metadata with no
// backing object yet.
type CreateAttachmentRequest struct {
	PlatformFileID          string `json:"platform_file_id"`
	FileType                string `json:"file_type"`
	FileName                string `json:"file_name,omitempty"`
	MimeType                string `json:"mime_type,omitempty"`
	FileSize                int64  `json:"file_size,omitempty"`
	ThumbnailPlatformFileID string `json:"thumbnail_platform_file_id,omitempty"`
	StoragePath             string `json:"storage_path,omitempty"`
	ThumbnailStoragePath    string `json:"thumbnail_storage_path,omitempty"`
	Width                   int    `json:"width,omitempty"`
	Height                  int    `json:"height,omitempty"`
	Duration                int    `json:"duration,omitempty"`
}

// UpdateAttachmentRequest is the input for a partial attachment update.
// Only non-zero-valued fields are applied -- see ToUpdateFields -- so
// omitting a field in the request leaves that column untouched rather
// than clobbering it with a zero value.
type UpdateAttachmentRequest struct {
	ID                      uint   `json:"id"`
	FileName                string `json:"file_name,omitempty"`
	MimeType                string `json:"mime_type,omitempty"`
	ThumbnailPlatformFileID string `json:"thumbnail_platform_file_id,omitempty"`
	Width                   int    `json:"width,omitempty"`
	Height                  int    `json:"height,omitempty"`
	Duration                int    `json:"duration,omitempty"`
}

// AttachmentResponse is the ONLY shape AttachmentService ever returns for
// an attachment -- no entity.Attachment crosses the service boundary in
// either direction. StoragePath/ThumbnailStoragePath are the raw MinIO
// object keys (NOT downloadable URLs -- use GetAttachmentDownloadURL /
// GetAttachmentsDownloadURLsBatch for those); they're included here
// mainly so a caller can tell whether the upload has completed yet
// (empty means "not uploaded" -- see entity.Attachment's doc comment).
type AttachmentResponse struct {
	ID                      uint   `json:"id"`
	ChatHistoryID           uint   `json:"chat_history_id"`
	PlatformFileID          string `json:"platform_file_id"`
	FileType                string `json:"file_type"`
	FileName                string `json:"file_name,omitempty"`
	MimeType                string `json:"mime_type,omitempty"`
	FileSize                int64  `json:"file_size,omitempty"`
	ThumbnailPlatformFileID string `json:"thumbnail_platform_file_id,omitempty"`
	StoragePath             string `json:"storage_path,omitempty"`
	ThumbnailStoragePath    string `json:"thumbnail_storage_path,omitempty"`
	Width                   int    `json:"width,omitempty"`
	Height                  int    `json:"height,omitempty"`
	Duration                int    `json:"duration,omitempty"`
	CreatedAt               string `json:"created_at"`
}

// AttachmentListQuery carries GET /attachments' pagination and filter.
type AttachmentListQuery struct {
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Sort     string `json:"sort"`
	FileType string `json:"file_type"`
}

// AttachmentListResponse is returned by ListAttachments.
type AttachmentListResponse struct {
	Attachments []AttachmentResponse `json:"attachments"`
	Total       int64                `json:"total"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}

// IngestMessageWithAttachmentsRequest is the input for
// AttachmentService.IngestMessageWithAttachments: everything needed to
// create one ChatHistory row plus its attachments atomically. RawPayload
// is a plain []byte (not gorm.io/datatypes.JSON) precisely because this
// is a DTO -- the service maps it to the entity's JSONB column type
// internally; nothing GORM-specific belongs on this struct.
//
// MediaGroupID is the platform's (Telegram/Bale) media_group_id. An album
// is delivered as SEPARATE updates -- one per file -- that share this ID.
// When it is set, the service folds every update of the same
// (ChatID, MediaGroupID) into ONE ChatHistory row (created by the first
// update to arrive) with one Attachment per file. Leave it empty for
// ordinary single-message ingestion.
type IngestMessageWithAttachmentsRequest struct {
	ChatID            uint                      `json:"chat_id"`
	MediaGroupID      string                    `json:"media_group_id,omitempty"` //todo : WARNING!
	PlatformMessageID int64                     `json:"platform_message_id"`
	SenderID          string                    `json:"sender_id,omitempty"`
	SenderName        string                    `json:"sender_name,omitempty"`
	Content           string                    `json:"content,omitempty"`
	MediaType         string                    `json:"media_type"`
	RawPayload        []byte                    `json:"raw_payload,omitempty"`
	MessageTimestamp  time.Time                 `json:"message_timestamp"`
	Attachments       []CreateAttachmentRequest `json:"attachments,omitempty"`
}

// ==========================================
// 2. Interface
// ==========================================

// AttachmentService is the sole entry point for Attachment business
// logic. No entity.Attachment -- or any other GORM entity -- crosses this
// boundary in either direction: every input is a primitive,
// context.Context, or a Request DTO; every output is a primitive, error,
// or Response DTO. Entity<->DTO mapping happens exclusively via the
// To*/mapper functions in attachment_mapper.go, inside the implementation
// (internal/application/service), never at the call site -- including in
// HTTP handlers, which must never import internal/domain/entity for
// Attachment purposes.
//
// Object-storage-backed methods (GetAttachmentDownloadURL and below) are
// implemented against repository_contract.StorageRepository, never
// against minio-go directly -- see that interface's doc comment for the
// "zero SDK leakage" rule this enforces.
type AttachmentService interface {
	CreateAttachment(ctx context.Context, chatHistoryID uint, req CreateAttachmentRequest) (*AttachmentResponse, error)
	CreateAttachmentsBatch(ctx context.Context, chatHistoryID uint, reqs []CreateAttachmentRequest) ([]AttachmentResponse, error)
	// UploadDirectAttachment handles a direct client file upload (e.g. a
	// multipart/form-data POST): it streams fileHeader's content straight
	// into object storage, then creates the DB record with the resulting
	// StoragePath already populated.
	UploadDirectAttachment(ctx context.Context, chatHistoryID uint, fileHeader *multipart.FileHeader) (*AttachmentResponse, error)

	GetAttachmentByID(ctx context.Context, id uint) (*AttachmentResponse, error)
	GetAttachmentsByMessageID(ctx context.Context, chatHistoryID uint) ([]AttachmentResponse, error)

	GetAttachmentsByChatHistoryIDsBatch(ctx context.Context, chatHistoryIDs []uint) (map[uint][]AttachmentResponse, error)
	GetAttachmentByPlatformFileID(ctx context.Context, platformFileID string) (*AttachmentResponse, error)
	ListAttachments(ctx context.Context, query AttachmentListQuery) (*AttachmentListResponse, error)
	// GetAttachmentDownloadURL returns a time-limited presigned MinIO URL
	// for the attachment's stored object. Errors if the attachment has no
	// StoragePath yet (upload not completed/attempted).
	GetAttachmentDownloadURL(ctx context.Context, attachmentID uint) (string, error)
	// GetAttachmentsDownloadURLsBatch is the batch form of the above;
	// attachments with no StoragePath are silently omitted from the
	// result map rather than failing the whole batch.
	GetAttachmentsDownloadURLsBatch(ctx context.Context, attachmentIDs []uint) (map[uint]string, error)

	UpdateAttachment(ctx context.Context, req UpdateAttachmentRequest) (*AttachmentResponse, error)
	UpdateThumbnailID(ctx context.Context, attachmentID uint, thumbnailPlatformFileID string) error

	DeleteAttachment(ctx context.Context, id uint) error
	DeleteAttachmentsByIDs(ctx context.Context, ids []uint) error
	DeleteAttachmentsByMessageID(ctx context.Context, chatHistoryID uint) error
	// DeleteAttachmentWithStorage soft-deletes the DB record AND removes
	// its object(s) from storage (best-effort: a storage-delete failure
	// is logged, not returned, since the DB row -- the source of truth
	// for "does this exist" -- is already gone by that point).
	DeleteAttachmentWithStorage(ctx context.Context, attachmentID uint) error
	// DeleteAttachmentsWithStorageBatch is the batch form of the above,
	// using StorageRepository.DeleteFilesBatch for the object removal.
	DeleteAttachmentsWithStorageBatch(ctx context.Context, attachmentIDs []uint) error
	RestoreAttachment(ctx context.Context, id uint) (*AttachmentResponse, error)

	// ReplaceMessageAttachments atomically soft-deletes chatHistoryID's
	// existing attachments and inserts reqs in their place (old rows are
	// tombstoned, not erased -- see the implementation's doc comment).
	ReplaceMessageAttachments(ctx context.Context, chatHistoryID uint, reqs []CreateAttachmentRequest) ([]AttachmentResponse, error)
	// IngestMessageWithAttachments persists one ChatHistory row and its
	// attachments as a single transaction via database.TrxManager --
	// either both land, or neither does. Callers that already uploaded
	// media to storage (e.g. the Telegram ingestion flow) set
	// StoragePath/ThumbnailStoragePath on each CreateAttachmentRequest
	// before calling this.
	//
	// If req.MediaGroupID is set, updates of the same album are merged:
	// the first one creates the ChatHistory row and records
	// (MediaGroupID, ChatID) -> ChatHistoryID in MediaGroupService; every
	// later one only adds its attachment(s) to that cached ChatHistory.
	IngestMessageWithAttachments(ctx context.Context, req IngestMessageWithAttachmentsRequest) error
}

// ==========================================
// 3. Mappers
// ==========================================

// The mapper functions below are plain, stateless functions with no
// shared or mutable package-level state -- every input is read-only and
// every output is freshly allocated -- so they are thread-safe by
// construction: concurrent goroutines calling them with different
// arguments never interfere with one another.

// ToAttachmentEntity maps a CreateAttachmentRequest (already known to
// belong to chatHistoryID) into a new, unsaved entity.Attachment.
func ToAttachmentEntity(chatHistoryID uint, req CreateAttachmentRequest) entity.Attachment {
	return entity.Attachment{
		ChatHistoryID:           chatHistoryID,
		PlatformFileID:          req.PlatformFileID,
		FileType:                entity.AttachmentFileType(req.FileType),
		FileName:                req.FileName,
		MimeType:                req.MimeType,
		FileSize:                req.FileSize,
		ThumbnailPlatformFileID: req.ThumbnailPlatformFileID,
		StoragePath:             req.StoragePath,
		ThumbnailStoragePath:    req.ThumbnailStoragePath,
		Width:                   req.Width,
		Height:                  req.Height,
		Duration:                req.Duration,
	}
}

// ToAttachmentEntities maps a slice of requests in one pass.
func ToAttachmentEntities(chatHistoryID uint, reqs []CreateAttachmentRequest) []entity.Attachment {
	entities := make([]entity.Attachment, 0, len(reqs))
	for _, req := range reqs {
		entities = append(entities, ToAttachmentEntity(chatHistoryID, req))
	}
	return entities
}

// ToAttachmentResponse maps a persisted Attachment into its
// service-facing representation.
func ToAttachmentResponse(attachment entity.Attachment) AttachmentResponse {
	return AttachmentResponse{
		ID:                      attachment.ID,
		ChatHistoryID:           attachment.ChatHistoryID,
		PlatformFileID:          attachment.PlatformFileID,
		FileType:                string(attachment.FileType),
		FileName:                attachment.FileName,
		MimeType:                attachment.MimeType,
		FileSize:                attachment.FileSize,
		ThumbnailPlatformFileID: attachment.ThumbnailPlatformFileID,
		StoragePath:             attachment.StoragePath,
		ThumbnailStoragePath:    attachment.ThumbnailStoragePath,
		Width:                   attachment.Width,
		Height:                  attachment.Height,
		Duration:                attachment.Duration,
		CreatedAt:               attachment.CreatedAt.Format(time.RFC3339),
	}
}

// ToAttachmentResponses maps a slice of entities in one pass.
func ToAttachmentResponses(attachments []entity.Attachment) []AttachmentResponse {
	responses := make([]AttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		responses = append(responses, ToAttachmentResponse(a))
	}
	return responses
}

// ToAttachmentResponseMap maps a chatHistoryID->[]entity.Attachment map
// (as returned by AttachmentRepository.GetByChatHistoryIDs) into its
// response-DTO equivalent, for GetAttachmentsByChatHistoryIDsBatch.
func ToAttachmentResponseMap(byChatHistoryID map[uint][]entity.Attachment) map[uint][]AttachmentResponse {
	result := make(map[uint][]AttachmentResponse, len(byChatHistoryID))
	for chatHistoryID, attachments := range byChatHistoryID {
		result[chatHistoryID] = ToAttachmentResponses(attachments)
	}
	return result
}

// ToUpdateFields turns a partial update request into the {column: value}
// map UpdateFields/UpdateFieldsBatch expect, including only the fields
// the caller actually set: FileName/MimeType/ThumbnailPlatformFileID
// non-empty, Width/Height/Duration non-zero. This is what makes
// UpdateAttachmentRequest a genuine partial update -- a field the caller
// left at its zero value is never written, so it can't clobber existing
// data.
func ToUpdateFields(req UpdateAttachmentRequest) map[string]interface{} {
	fields := make(map[string]interface{})
	if req.FileName != "" {
		fields["file_name"] = req.FileName
	}
	if req.MimeType != "" {
		fields["mime_type"] = req.MimeType
	}
	if req.ThumbnailPlatformFileID != "" {
		fields["thumbnail_platform_file_id"] = req.ThumbnailPlatformFileID
	}
	if req.Width != 0 {
		fields["width"] = req.Width
	}
	if req.Height != 0 {
		fields["height"] = req.Height
	}
	if req.Duration != 0 {
		fields["duration"] = req.Duration
	}
	return fields
}
