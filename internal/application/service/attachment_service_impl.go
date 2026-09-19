package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
	"messenger-backend/pkg/logger"
)

// attachmentService implements service_contract.AttachmentService.
// Cross-entity atomicity (IngestMessageWithAttachments,
// ReplaceMessageAttachments) is orchestrated via
// database.TrxManager.WithTransaction; object storage is reached only
// through repository_contract.StorageRepository, never minio-go directly
// (see that interface's doc comment).
type attachmentService struct {
	attachmentRepo  repository_contract.AttachmentRepository
	chatHistoryRepo repository_contract.ChatHistoryRepository
	storageRepo     repository_contract.StorageRepository
	trxManager      database.TrxManager
	log             logger.Logger

	// bucketName is the single bucket this service reads/writes --
	// StorageRepository itself supports multiple buckets, but nothing
	// so far in this codebase needs more than one.
	bucketName string
	// presignExpiry bounds how long a GetAttachmentDownloadURL /
	// GetAttachmentsDownloadURLsBatch link stays valid.
	presignExpiry time.Duration
}

func NewAttachmentService(
	attachmentRepo repository_contract.AttachmentRepository,
	chatHistoryRepo repository_contract.ChatHistoryRepository,
	storageRepo repository_contract.StorageRepository,
	trxManager database.TrxManager,
	log logger.Logger,
	bucketName string,
	presignExpiry time.Duration,
) service_contract.AttachmentService {
	if presignExpiry <= 0 {
		presignExpiry = 15 * time.Minute
	}
	return &attachmentService{
		attachmentRepo:  attachmentRepo,
		chatHistoryRepo: chatHistoryRepo,
		storageRepo:     storageRepo,
		trxManager:      trxManager,
		log:             log.With(logger.String("component", "attachment_service")),
		bucketName:      bucketName,
		presignExpiry:   presignExpiry,
	}
}

// --- Create ---

func (s *attachmentService) CreateAttachment(ctx context.Context, chatHistoryID uint, req service_contract.CreateAttachmentRequest) (*service_contract.AttachmentResponse, error) {
	att := service_contract.ToAttachmentEntity(chatHistoryID, req)

	if err := s.attachmentRepo.Create(ctx, &att); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment created", logger.Uint("attachment_id", att.ID), logger.Uint("chat_history_id", chatHistoryID))

	resp := service_contract.ToAttachmentResponse(att)
	return &resp, nil
}

func (s *attachmentService) CreateAttachmentsBatch(ctx context.Context, chatHistoryID uint, reqs []service_contract.CreateAttachmentRequest) ([]service_contract.AttachmentResponse, error) {
	if len(reqs) == 0 {
		return []service_contract.AttachmentResponse{}, nil
	}

	entities := service_contract.ToAttachmentEntities(chatHistoryID, reqs)
	if err := s.attachmentRepo.CreateBatch(ctx, entities); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachments batch created", logger.Uint("chat_history_id", chatHistoryID), logger.Int("count", len(entities)))

	return service_contract.ToAttachmentResponses(entities), nil
}

// UploadDirectAttachment streams fileHeader straight into object storage,
// then creates the DB record with the resulting StoragePath. If the DB
// write fails, the just-uploaded object is removed (best-effort) so a
// failed request doesn't leave an orphaned file behind.
func (s *attachmentService) UploadDirectAttachment(ctx context.Context, chatHistoryID uint, fileHeader *multipart.FileHeader) (*service_contract.AttachmentResponse, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, exception.Wrap(exception.ErrBadRequest, err)
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	objectKey := buildDirectUploadObjectKey(chatHistoryID, fileHeader.Filename)

	storagePath, err := s.storageRepo.UploadFile(ctx, s.bucketName, objectKey, file, fileHeader.Size, contentType)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	att := entity.Attachment{
		ChatHistoryID: chatHistoryID,
		// No platform (Telegram/Bale) file id for a direct client
		// upload -- the storage object key is this attachment's only
		// stable identifier, so it doubles as PlatformFileID too.
		PlatformFileID: objectKey,
		FileType:       inferFileType(contentType),
		FileName:       fileHeader.Filename,
		MimeType:       contentType,
		FileSize:       fileHeader.Size,
		StoragePath:    storagePath,
	}

	if err := s.attachmentRepo.Create(ctx, &att); err != nil {
		if delErr := s.storageRepo.DeleteFile(ctx, s.bucketName, storagePath); delErr != nil {
			s.log.Warn("failed to clean up orphaned upload after DB error", logger.Err(delErr), logger.String("storage_path", storagePath))
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment uploaded directly", logger.Uint("attachment_id", att.ID), logger.Uint("chat_history_id", chatHistoryID), logger.String("storage_path", storagePath))

	resp := service_contract.ToAttachmentResponse(att)
	return &resp, nil
}

// --- Read & query ---

func (s *attachmentService) GetAttachmentByID(ctx context.Context, id uint) (*service_contract.AttachmentResponse, error) {
	att, err := s.attachmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, exception.ErrAttachmentNotFound) {
			return nil, exception.ErrAttachmentNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToAttachmentResponse(*att)
	return &resp, nil
}

func (s *attachmentService) GetAttachmentsByMessageID(ctx context.Context, chatHistoryID uint) ([]service_contract.AttachmentResponse, error) {
	attachments, err := s.attachmentRepo.GetByChatHistoryID(ctx, chatHistoryID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	return service_contract.ToAttachmentResponses(attachments), nil
}

func (s *attachmentService) GetAttachmentsByChatHistoryIDsBatch(ctx context.Context, chatHistoryIDs []uint) (map[uint][]service_contract.AttachmentResponse, error) {
	byChatHistoryID, err := s.attachmentRepo.GetByChatHistoryIDs(ctx, chatHistoryIDs)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	return service_contract.ToAttachmentResponseMap(byChatHistoryID), nil
}

func (s *attachmentService) GetAttachmentByPlatformFileID(ctx context.Context, platformFileID string) (*service_contract.AttachmentResponse, error) {
	att, err := s.attachmentRepo.GetByPlatformFileID(ctx, platformFileID)
	if err != nil {
		if errors.Is(err, exception.ErrAttachmentNotFound) {
			return nil, exception.ErrAttachmentNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToAttachmentResponse(*att)
	return &resp, nil
}

func (s *attachmentService) ListAttachments(ctx context.Context, query service_contract.AttachmentListQuery) (*service_contract.AttachmentListResponse, error) {
	limit := query.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	attachments, total, err := s.attachmentRepo.List(ctx, limit, offset, query.Sort, query.FileType)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	return &service_contract.AttachmentListResponse{
		Attachments: service_contract.ToAttachmentResponses(attachments),
		Total:       total,
		Limit:       limit,
		Offset:      offset,
	}, nil
}

// GetAttachmentDownloadURL fetches attachment metadata and returns a
// presigned MinIO URL for its stored object.
func (s *attachmentService) GetAttachmentDownloadURL(ctx context.Context, attachmentID uint) (string, error) {
	att, err := s.attachmentRepo.GetByID(ctx, attachmentID)
	if err != nil {
		if errors.Is(err, exception.ErrAttachmentNotFound) {
			return "", exception.ErrAttachmentNotFound
		}
		return "", exception.Wrap(exception.ErrInternal, err)
	}
	if att.StoragePath == "" {
		return "", exception.Wrap(exception.ErrInternal, fmt.Errorf("attachment %d has no stored object yet", attachmentID))
	}

	presignedURL, err := s.storageRepo.GetPresignedURL(ctx, s.bucketName, att.StoragePath, s.presignExpiry)
	if err != nil {
		return "", exception.Wrap(exception.ErrInternal, err)
	}
	return presignedURL, nil
}

// GetAttachmentsDownloadURLsBatch fetches multiple attachments and
// generates presigned URLs in one batch call. Attachments with no
// StoragePath (upload not completed) are silently skipped rather than
// failing the whole batch.
func (s *attachmentService) GetAttachmentsDownloadURLsBatch(ctx context.Context, attachmentIDs []uint) (map[uint]string, error) {
	if len(attachmentIDs) == 0 {
		return map[uint]string{}, nil
	}

	attachments, err := s.attachmentRepo.GetByIDs(ctx, attachmentIDs)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	keyToID := make(map[string]uint, len(attachments))
	keys := make([]string, 0, len(attachments))
	for _, att := range attachments {
		if att.StoragePath == "" {
			continue
		}
		keyToID[att.StoragePath] = att.ID
		keys = append(keys, att.StoragePath)
	}
	if len(keys) == 0 {
		return map[uint]string{}, nil
	}

	urlsByKey, err := s.storageRepo.GetPresignedURLsBatch(ctx, s.bucketName, keys, s.presignExpiry)
	if err != nil && len(urlsByKey) == 0 {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	result := make(map[uint]string, len(urlsByKey))
	for key, presignedURL := range urlsByKey {
		result[keyToID[key]] = presignedURL
	}
	return result, nil
}

// --- Update ---

func (s *attachmentService) UpdateAttachment(ctx context.Context, req service_contract.UpdateAttachmentRequest) (*service_contract.AttachmentResponse, error) {
	fields := service_contract.ToUpdateFields(req)
	if len(fields) > 0 {
		if err := s.attachmentRepo.UpdateFields(ctx, req.ID, fields); err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
	}

	att, err := s.attachmentRepo.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, exception.ErrAttachmentNotFound) {
			return nil, exception.ErrAttachmentNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment updated", logger.Uint("attachment_id", req.ID))

	resp := service_contract.ToAttachmentResponse(*att)
	return &resp, nil
}

func (s *attachmentService) UpdateThumbnailID(ctx context.Context, attachmentID uint, thumbnailPlatformFileID string) error {
	err := s.attachmentRepo.UpdateFields(ctx, attachmentID, map[string]interface{}{
		"thumbnail_platform_file_id": thumbnailPlatformFileID,
	})
	if err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment thumbnail updated", logger.Uint("attachment_id", attachmentID))
	return nil
}

// --- Delete & restore ---

func (s *attachmentService) DeleteAttachment(ctx context.Context, id uint) error {
	if err := s.attachmentRepo.DeleteByID(ctx, id); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment deleted", logger.Uint("attachment_id", id))
	return nil
}

func (s *attachmentService) DeleteAttachmentsByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.attachmentRepo.DeleteByIDs(ctx, ids); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachments batch deleted", logger.Int("count", len(ids)))
	return nil
}

func (s *attachmentService) DeleteAttachmentsByMessageID(ctx context.Context, chatHistoryID uint) error {
	if err := s.attachmentRepo.DeleteByChatHistoryID(ctx, chatHistoryID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachments deleted for message", logger.Uint("chat_history_id", chatHistoryID))
	return nil
}

// DeleteAttachmentWithStorage soft-deletes the DB record and removes its
// object(s) from storage. Storage cleanup is best-effort: a failure there
// is logged, not returned, since the DB row -- the source of truth for
// "does this attachment exist" -- is already gone by that point, and an
// orphaned object is a cheaper problem than a delete a caller can never
// get to succeed.
func (s *attachmentService) DeleteAttachmentWithStorage(ctx context.Context, attachmentID uint) error {
	att, err := s.attachmentRepo.GetByID(ctx, attachmentID)
	if err != nil {
		if errors.Is(err, exception.ErrAttachmentNotFound) {
			return exception.ErrAttachmentNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.attachmentRepo.DeleteByID(ctx, attachmentID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	if att.StoragePath != "" {
		if err := s.storageRepo.DeleteFile(ctx, s.bucketName, att.StoragePath); err != nil {
			s.log.Warn("failed to delete attachment object from storage", logger.Err(err), logger.Uint("attachment_id", attachmentID), logger.String("storage_path", att.StoragePath))
		}
	}
	if att.ThumbnailStoragePath != "" {
		if err := s.storageRepo.DeleteFile(ctx, s.bucketName, att.ThumbnailStoragePath); err != nil {
			s.log.Warn("failed to delete attachment thumbnail from storage", logger.Err(err), logger.Uint("attachment_id", attachmentID), logger.String("storage_path", att.ThumbnailStoragePath))
		}
	}

	s.log.Info("attachment deleted with storage cleanup", logger.Uint("attachment_id", attachmentID))
	return nil
}

// DeleteAttachmentsWithStorageBatch is the batch form of
// DeleteAttachmentWithStorage, using StorageRepository.DeleteFilesBatch
// (MinIO's native bulk-delete) for the object removal.
func (s *attachmentService) DeleteAttachmentsWithStorageBatch(ctx context.Context, attachmentIDs []uint) error {
	if len(attachmentIDs) == 0 {
		return nil
	}

	attachments, err := s.attachmentRepo.GetByIDs(ctx, attachmentIDs)
	if err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.attachmentRepo.DeleteByIDs(ctx, attachmentIDs); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	keys := make([]string, 0, len(attachments)*2)
	for _, att := range attachments {
		if att.StoragePath != "" {
			keys = append(keys, att.StoragePath)
		}
		if att.ThumbnailStoragePath != "" {
			keys = append(keys, att.ThumbnailStoragePath)
		}
	}

	if len(keys) > 0 {
		if err := s.storageRepo.DeleteFilesBatch(ctx, s.bucketName, keys); err != nil {
			s.log.Warn("failed to batch-delete attachment objects from storage", logger.Err(err), logger.Int("key_count", len(keys)))
		}
	}

	s.log.Info("attachments batch-deleted with storage cleanup", logger.Int("count", len(attachmentIDs)))
	return nil
}

func (s *attachmentService) RestoreAttachment(ctx context.Context, id uint) (*service_contract.AttachmentResponse, error) {
	if err := s.attachmentRepo.RestoreByID(ctx, id); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	att, err := s.attachmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, exception.ErrAttachmentNotFound) {
			return nil, exception.ErrAttachmentNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment restored", logger.Uint("attachment_id", id))

	resp := service_contract.ToAttachmentResponse(*att)
	return &resp, nil
}

// ReplaceMessageAttachments soft-deletes chatHistoryID's existing
// attachments and inserts reqs in their place inside a single
// transaction. Old rows are tombstoned (deleted_at set) rather than
// hard-deleted, so a message's attachment history stays auditable. Note
// this does NOT clean up the old rows' storage objects (unlike
// DeleteAttachmentWithStorage) -- use that explicitly first if the old
// files should be removed from MinIO too, since a tombstoned-but-still-
// downloadable attachment is sometimes exactly what's wanted (e.g. "edit
// history").
func (s *attachmentService) ReplaceMessageAttachments(ctx context.Context, chatHistoryID uint, reqs []service_contract.CreateAttachmentRequest) ([]service_contract.AttachmentResponse, error) {
	entities := service_contract.ToAttachmentEntities(chatHistoryID, reqs)

	err := s.trxManager.WithTransaction(ctx, func(trxCtx context.Context) error {
		if err := s.attachmentRepo.DeleteByChatHistoryID(trxCtx, chatHistoryID); err != nil {
			return err
		}
		if len(entities) == 0 {
			return nil
		}
		return s.attachmentRepo.CreateBatch(trxCtx, entities)
	})
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("message attachments replaced", logger.Uint("chat_history_id", chatHistoryID), logger.Int("new_count", len(entities)))

	return service_contract.ToAttachmentResponses(entities), nil
}

// IngestMessageWithAttachments persists one ChatHistory row and its
// attachments as a single transaction: chatHistoryRepo.Create and
// attachmentRepo.CreateBatch are both called with trxCtx (the context
// TrxManager injected its transaction into), so both repositories --
// via their own database.ExtractTrxOrDB(ctx, r.db) -- resolve to the SAME
// underlying transaction automatically. Either both rows land, or
// neither does. Any StoragePath/ThumbnailStoragePath already set on
// req.Attachments (by the caller, having already uploaded to MinIO) is
// persisted as-is -- this method does not itself touch object storage.
func (s *attachmentService) IngestMessageWithAttachments(ctx context.Context, req service_contract.IngestMessageWithAttachmentsRequest) error {
	message := &entity.ChatHistory{
		ChatID:            req.ChatID,
		PlatformMessageID: req.PlatformMessageID,
		SenderID:          req.SenderID,
		SenderName:        req.SenderName,
		Content:           req.Content,
		MediaType:         req.MediaType,
		RawPayload:        datatypes.JSON(req.RawPayload),
		MessageTimestamp:  req.MessageTimestamp,
	}

	err := s.trxManager.WithTransaction(ctx, func(trxCtx context.Context) error {
		if err := s.chatHistoryRepo.Create(trxCtx, message); err != nil {
			return err
		}

		if len(req.Attachments) == 0 {
			return nil
		}

		attachments := service_contract.ToAttachmentEntities(message.ID, req.Attachments)
		return s.attachmentRepo.CreateBatch(trxCtx, attachments)
	})
	if err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("message ingested with attachments",
		logger.Uint("chat_id", req.ChatID),
		logger.Uint("chat_history_id", message.ID),
		logger.Int("attachment_count", len(req.Attachments)),
	)
	return nil
}

// --- helpers ---

// inferFileType maps a direct upload's Content-Type to our FileType
// vocabulary. Falls back to "document" for anything not obviously a
// photo/video/audio -- see entity.AttachmentFileType's doc comment for
// the full list.
func inferFileType(contentType string) entity.AttachmentFileType {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return entity.AttachmentTypePhoto
	case strings.HasPrefix(contentType, "video/"):
		return entity.AttachmentTypeVideo
	case strings.HasPrefix(contentType, "audio/"):
		return entity.AttachmentTypeAudio
	default:
		return entity.AttachmentTypeDocument
	}
}

// buildDirectUploadObjectKey mirrors the structured path convention used
// for bot-ingested media (see telegramhandlers.buildObjectKey) but under
// a "direct/" prefix and keyed by chatHistoryID rather than company/chat,
// since a direct API upload is already scoped to one specific message.
func buildDirectUploadObjectKey(chatHistoryID uint, fileName string) string {
	now := time.Now().UTC()
	safeName := fileName
	if safeName == "" {
		safeName = uuid.NewString()
	}
	return fmt.Sprintf("direct/chat_histories/%d/%04d/%02d/%s", chatHistoryID, now.Year(), int(now.Month()), safeName)
}
