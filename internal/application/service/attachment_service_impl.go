package service

import (
	"context"
	"errors"
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
	"messenger-backend/pkg/logger"

	"gorm.io/datatypes"
)

// attachmentService implements service_contract.AttachmentService. Cross-entity
// atomicity (IngestMessageWithAttachments, ReplaceMessageAttachments) is
// orchestrated here via database.TrxManager.WithTransaction, never by a
// repository method accepting a transaction handle directly -- see
// AttachmentRepository's doc comment for the context-propagation pattern
// this relies on.
type attachmentService struct {
	attachmentRepo  repository_contract.AttachmentRepository
	chatHistoryRepo repository_contract.ChatHistoryRepository
	trxManager      database.TrxManager
	log             logger.Logger
}

func NewAttachmentService(
	attachmentRepo repository_contract.AttachmentRepository,
	chatHistoryRepo repository_contract.ChatHistoryRepository,
	trxManager database.TrxManager,
	log logger.Logger,
) service_contract.AttachmentService {
	return &attachmentService{
		attachmentRepo:  attachmentRepo,
		chatHistoryRepo: chatHistoryRepo,
		trxManager:      trxManager,
		log:             log.With(logger.String("component", "attachment_service")),
	}
}

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

func (s *attachmentService) DeleteAttachment(ctx context.Context, id uint) error {
	if err := s.attachmentRepo.DeleteByID(ctx, id); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachment deleted", logger.Uint("attachment_id", id))
	return nil
}

func (s *attachmentService) DeleteAttachmentsByMessageID(ctx context.Context, chatHistoryID uint) error {
	if err := s.attachmentRepo.DeleteByChatHistoryID(ctx, chatHistoryID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("attachments deleted for message", logger.Uint("chat_history_id", chatHistoryID))
	return nil
}

// ReplaceMessageAttachments soft-deletes chatHistoryID's existing
// attachments and inserts reqs in their place inside a single
// transaction. Old rows are tombstoned (deleted_at set) rather than
// hard-deleted, so a message's attachment history stays auditable.
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
// TrxManager injected its transaction into), so both repositories -- via
// their own database.ExtractTrxOrDB(ctx, r.db) -- resolve to the SAME
// underlying transaction automatically. Either both rows land, or
// neither does.
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
