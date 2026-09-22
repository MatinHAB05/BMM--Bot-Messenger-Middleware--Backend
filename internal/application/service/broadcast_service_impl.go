package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger"

	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"gorm.io/datatypes"
)

// todo : mq :)
type BroadcastConfig struct {
	JobTimeout time.Duration
	// AttachmentBucket is the object storage (MinIO/S3) bucket broadcast
	// attachments are uploaded to before being sent to each platform.
	// Required whenever a Broadcast request includes an attachment.
	AttachmentBucket string
}

type BroadcastTarget struct {
	Name      string
	Platform  string
	IsPrivate bool
	Type      string
}

type broadcastJob struct {
	platform string
	chat     entity.Chat
}

type jobResult struct {
	result service_contract.BroadcastResult
	target service_contract.BroadcastTarget
}

// preparedAttachmentFile is one broadcast attachment file after it has
// been uploaded to object storage: storagePath is the resulting bucket
// key, and data is kept around (in addition to being in storage) so it
// can be re-read for every target chat/platform without re-downloading
// it from MinIO per send.
type preparedAttachmentFile struct {
	fileName    string
	contentType string
	size        int64
	storagePath string
	data        []byte
}

type broadcastService struct {
	clients            map[string]messenger.MessengerClient
	chatRepo           repository_contract.ChatRepository
	chatHisRepo        repository_contract.ChatHistoryRepository
	sentbalemsgService service_contract.SentBaleMsgService
	storageRepo        repository_contract.StorageRepository
	log                logger.Logger
	cfg                BroadcastConfig
	pool               *ants.Pool
}

func NewBroadcastService(
	clients []messenger.MessengerClient,
	chatRepo repository_contract.ChatRepository,
	chatHisRepo repository_contract.ChatHistoryRepository,
	sentbalemsgService service_contract.SentBaleMsgService,
	storageRepo repository_contract.StorageRepository,
	log logger.Logger,
	cfg BroadcastConfig,
	pool *ants.Pool,
) service_contract.BroadcastService {
	registry := make(map[string]messenger.MessengerClient, len(clients))
	for _, c := range clients {
		registry[c.Platform()] = c
	}

	return &broadcastService{
		clients:            registry,
		chatRepo:           chatRepo,
		chatHisRepo:        chatHisRepo,
		sentbalemsgService: sentbalemsgService,
		storageRepo:        storageRepo,
		log:                log.With(logger.String("component", "broadcast_service")),
		cfg:                cfg,
		pool:               pool,
	}
}

func (s *broadcastService) validatePlatforms(platforms []string) error {
	for _, platform := range platforms {
		if _, ok := s.clients[platform]; !ok {
			return exception.Wrap(
				exception.ErrUnsupportedPlatform,
				fmt.Errorf("no client registered for platform %q", platform),
			)
		}
	}
	return nil
}

// validateBroadcastAttachment defends the service against callers other
// than BroadcastHandler (which already validates type/size/count on the
// way in) -- e.g. anything invoking BroadcastService directly.
func validateBroadcastAttachment(a service_contract.BroadcastAttachment) error {
	switch a.Type {
	case service_contract.BroadcastAttachmentPhoto,
		service_contract.BroadcastAttachmentVideo,
		service_contract.BroadcastAttachmentVoice,
		service_contract.BroadcastAttachmentAudio,
		service_contract.BroadcastAttachmentDocument,
		service_contract.BroadcastAttachmentAnimation:
	default:
		return fmt.Errorf("unsupported attachment type %q", a.Type)
	}
	if len(a.Files) == 0 {
		return fmt.Errorf("attachment must include at least one file")
	}
	for _, f := range a.Files {
		if len(f.Data) == 0 {
			return fmt.Errorf("attachment %q has no data", f.FileName)
		}
	}
	return nil
}

// sanitizeObjectKeySegment keeps a user-supplied file name from breaking
// the "/"-delimited object key it's embedded in.
func sanitizeObjectKeySegment(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" {
		return "file"
	}
	return name
}

// uploadBroadcastAttachments persists every file in a broadcast's
// attachment batch to object storage once (not once per target chat --
// the bytes are identical for every send), keyed under the broadcast's
// own UUID so re-running/inspecting a specific broadcast's files later is
// straightforward.
func (s *broadcastService) uploadBroadcastAttachments(ctx context.Context, broadcastUUID uuid.UUID, files []service_contract.BroadcastAttachmentFile) ([]preparedAttachmentFile, error) {
	payloads := make([]repository_contract.FileUploadPayload, len(files))
	prepared := make([]preparedAttachmentFile, len(files))

	for i, f := range files {
		objectKey := fmt.Sprintf("broadcasts/%s/%d-%s", broadcastUUID, i, sanitizeObjectKeySegment(f.FileName))

		payloads[i] = repository_contract.FileUploadPayload{
			ObjectKey:   objectKey,
			Reader:      bytes.NewReader(f.Data),
			Size:        int64(len(f.Data)),
			ContentType: f.ContentType,
		}
		prepared[i] = preparedAttachmentFile{
			fileName:    f.FileName,
			contentType: f.ContentType,
			size:        int64(len(f.Data)),
			storagePath: objectKey,
			data:        f.Data,
		}
	}

	if _, err := s.storageRepo.UploadFilesBatch(ctx, s.cfg.AttachmentBucket, payloads); err != nil {
		return nil, fmt.Errorf("upload broadcast attachments: %w", err)
	}

	return prepared, nil
}

func (s *broadcastService) runJobs(
	ctx context.Context,
	platformChats map[string][]entity.Chat,
	handle func(ctx context.Context, job broadcastJob) jobResult,
) ([]service_contract.BroadcastResult, []service_contract.BroadcastTarget) {
	var totalJobs int
	for _, chats := range platformChats {
		totalJobs += len(chats)
	}

	if totalJobs == 0 {
		return nil, nil
	}

	resultChan := make(chan jobResult, totalJobs)
	var wg sync.WaitGroup

Loop:
	for platform, chats := range platformChats {
		for _, chat := range chats {
			select {
			case <-ctx.Done():
				break Loop
			default:
			}

			j := broadcastJob{platform: platform, chat: chat}
			wg.Add(1)

			err := s.pool.Submit(func() {
				defer wg.Done()

				jobCtx := ctx
				if s.cfg.JobTimeout > 0 {
					var cancel context.CancelFunc
					jobCtx, cancel = context.WithTimeout(ctx, s.cfg.JobTimeout)
					defer cancel()
				}

				res := handle(jobCtx, j)

				select {
				case <-ctx.Done():
				case resultChan <- res:
				}
			})

			if err != nil {
				wg.Done()
				s.log.Error(err, "failed to submit job to ants pool", logger.String("platform", platform))
			}
		}
	}

	wg.Wait()
	close(resultChan)

	var results []service_contract.BroadcastResult
	var targets []service_contract.BroadcastTarget
	for r := range resultChan {
		results = append(results, r.result)
		targets = append(targets, r.target)
	}

	return results, targets
}

func targetFromChat(chat entity.Chat) service_contract.BroadcastTarget {
	return service_contract.BroadcastTarget{
		Name:      chat.Title,
		Platform:  string(chat.Platform),
		IsPrivate: chat.IsPrivate,
		Type:      string(chat.ChatType),
	}
}

func (s *broadcastService) Broadcast(ctx context.Context, companyID uint, req service_contract.BroadcastRequest) (*service_contract.BroadcastResponse, error) {
	if len(req.Platforms) == 0 {
		return nil, exception.Wrap(exception.ErrInternal, fmt.Errorf("no platforms specified"))
	}
	if req.Message == "" && req.Attachment == nil {
		return nil, exception.Wrap(exception.ErrInternal, fmt.Errorf("message or attachment is required"))
	}
	if req.Attachment != nil {
		if s.storageRepo == nil || s.cfg.AttachmentBucket == "" {
			return nil, exception.Wrap(exception.ErrInternal, fmt.Errorf("broadcast attachment storage is not configured"))
		}
		if err := validateBroadcastAttachment(*req.Attachment); err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
	}

	if err := s.validatePlatforms(req.Platforms); err != nil {
		return nil, err
	}

	platformChats, _, err := s.chatRepo.ListAll(ctx, companyID, req.Platforms)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	// ? we can use v7 (after enabale pool random) but we are fine for now
	broadcastUUID := uuid.New()

	// Attachments are uploaded to object storage exactly once here, ahead
	// of the fan-out below -- every target chat/platform reuses the same
	// storagePath and buffered bytes rather than each job re-uploading.
	var preparedFiles []preparedAttachmentFile
	if req.Attachment != nil {
		preparedFiles, err = s.uploadBroadcastAttachments(ctx, broadcastUUID, req.Attachment.Files)
		if err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
	}

	handle := func(ctx context.Context, job broadcastJob) jobResult {
		target := targetFromChat(job.chat)
		if req.Attachment == nil {
			return s.sendTextBroadcastJob(ctx, job, target, req.Message, broadcastUUID)
		}
		return s.sendAttachmentBroadcastJob(ctx, job, target, req.Message, req.Attachment.Type, preparedFiles, broadcastUUID)
	}

	results, targets := s.runJobs(ctx, platformChats, handle)

	return &service_contract.BroadcastResponse{
		Results: results,
		Targets: targets,
	}, nil
}

// sendTextBroadcastJob delivers a plain text message to one target chat.
// This is exactly Broadcast's original (pre-attachment) per-job body,
// factored out unchanged so text-only broadcasts behave identically to
// before.
func (s *broadcastService) sendTextBroadcastJob(ctx context.Context, job broadcastJob, target service_contract.BroadcastTarget, message string, broadcastUUID uuid.UUID) jobResult {
	client := s.clients[job.platform]

	mes, err := client.SendMessage(ctx, job.chat.PlatformChatID, message)
	if err != nil {
		s.log.Error(err, "broadcast send failed",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
		)
		return jobResult{
			result: service_contract.BroadcastResult{
				Platform: job.platform,
				Success:  false,
				Error:    []string{err.Error()},
			},
			target: target,
		}
	}

	s.log.Info("broadcast delivered",
		logger.String("platform", job.platform),
		logger.String("target_id", job.chat.PlatformChatID),
	)

	result := service_contract.BroadcastResult{
		Platform: job.platform,
		Success:  true,
	}

	if err := s.chatHisRepo.Create(ctx, &entity.ChatHistory{
		ChatID:            job.chat.ID,
		PlatformMessageID: mes.PlatformMessageID,
		SenderID:          mes.SenderID,
		SenderName:        mes.SenderName,
		Content:           mes.Content,
		MediaType:         mes.MediaType,
		RawPayload:        datatypes.JSON(mes.RawPayload),
		MessageTimestamp:  mes.Timestamp,
		IsBroadcast:       true,
		BroadcastUUID:     &broadcastUUID,
	}); err != nil {
		result.Success = false
		result.Error = append(result.Error, err.Error())
		s.log.Error(err, "broadcast history persist failed",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
		)
		s.log.Warn("message delivered but history not persisted; retry/reconciliation needed",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
		)
	}

	if err := s.sentbalemsgService.SetSentBaleMsg(ctx, job.chat.PlatformChatID, message); err != nil { // ? : FUCK BALE!
		result.Success = false
		result.Error = append(result.Error, err.Error())
	}

	return jobResult{result: result, target: target}
}

// sendAttachmentBroadcastJob delivers every file in the attachment batch
// (all the same type: N photos, N videos, ...) to one target chat, one
// platform Send call per file, and persists one ChatHistory row per file
// -- with its Attachment relation populated so it shows up as a real
// attachment on that message, not just a media_type label.
//
// Only the first file carries the caption (req.Message); the rest are
// sent without one, matching how a Telegram/Bale album shows a single
// caption rather than repeating it under every item.
func (s *broadcastService) sendAttachmentBroadcastJob(
	ctx context.Context,
	job broadcastJob,
	target service_contract.BroadcastTarget,
	caption string,
	attachmentType service_contract.BroadcastAttachmentType,
	files []preparedAttachmentFile,
	broadcastUUID uuid.UUID,
) jobResult {
	client := s.clients[job.platform]
	result := service_contract.BroadcastResult{Platform: job.platform, Success: true}

	for i, f := range files {
		fileCaption := ""
		if i == 0 {
			fileCaption = caption
		}

		mes, err := client.SendAttachment(ctx, job.chat.PlatformChatID, fileCaption, messenger.Attachment{
			Type:     messenger.AttachmentType(attachmentType),
			FileName: f.fileName,
			Data:     bytes.NewReader(f.data), // fresh reader: every job/file reads the same buffered bytes independently
		})
		if err != nil {
			s.log.Error(err, "broadcast attachment send failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
				logger.String("file", f.fileName),
			)
			result.Success = false
			result.Error = append(result.Error, fmt.Sprintf("%s: %s", f.fileName, err.Error()))
			continue
		}

		s.log.Info("broadcast attachment delivered",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
			logger.String("file", f.fileName),
		)

		history := &entity.ChatHistory{
			ChatID:            job.chat.ID,
			PlatformMessageID: mes.PlatformMessageID,
			SenderID:          mes.SenderID,
			SenderName:        mes.SenderName,
			Content:           mes.Content,
			MediaType:         mes.MediaType,
			RawPayload:        datatypes.JSON(mes.RawPayload),
			MessageTimestamp:  mes.Timestamp,
			IsBroadcast:       true,
			BroadcastUUID:     &broadcastUUID,
		}

		// Passing Attachments here (rather than persisting them
		// separately) lets gorm's default associations-on-create behavior
		// insert the attachments row wired to this exact ChatHistory via
		// its foreignKey -- no second write needed.
		if mes.Attachment != nil {
			history.Attachments = []entity.Attachment{
				{
					PlatformFileID:          mes.Attachment.PlatformFileID,
					FileType:                entity.AttachmentFileType(attachmentType),
					FileName:                f.fileName,
					MimeType:                f.contentType,
					FileSize:                f.size,
					ThumbnailPlatformFileID: mes.Attachment.ThumbnailPlatformFileID,
					StoragePath:             f.storagePath,
					Width:                   mes.Attachment.Width,
					Height:                  mes.Attachment.Height,
					Duration:                mes.Attachment.Duration,
				},
			}
		}

		if err := s.chatHisRepo.Create(ctx, history); err != nil {
			result.Success = false
			result.Error = append(result.Error, fmt.Sprintf("%s: %s", f.fileName, err.Error()))
			s.log.Error(err, "broadcast history persist failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
				logger.String("file", f.fileName),
			)
			s.log.Warn("attachment delivered but history not persisted; retry/reconciliation needed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
				logger.String("file", f.fileName),
			)
		}
	}

	if err := s.sentbalemsgService.SetSentBaleMsg(ctx, job.chat.PlatformChatID, caption); err != nil { // ? : FUCK BALE!
		result.Success = false
		result.Error = append(result.Error, err.Error())
	}

	return jobResult{result: result, target: target}
}

func (s *broadcastService) DeleteBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req service_contract.DeleteBroadcastRequest) error {
	if len(req.Platforms) == 0 {
		return exception.Wrap(exception.ErrInternal, fmt.Errorf("no platforms specified"))
	}
	if err := s.validatePlatforms(req.Platforms); err != nil {
		return err
	}

	platformChats, _, err := s.chatRepo.GetAllChatsContainsBroadcastMsgUUID(ctx, companyID, broadcastMsgUUID, req.Platforms)
	if err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	handle := func(ctx context.Context, job broadcastJob) jobResult {
		client := s.clients[job.platform]
		target := targetFromChat(job.chat)

		brmsg, err := s.chatHisRepo.GetByBroadcastMsgID(ctx, broadcastMsgUUID, job.chat.ID)
		if err != nil {
			s.log.Error(err, "broadcast delete: history lookup failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			return jobResult{
				result: service_contract.BroadcastResult{
					Platform: job.platform,
					Success:  false,
					Error:    []string{err.Error()},
				},
				target: target,
			}
		}

		if err := client.DeleteMessage(ctx, job.chat.PlatformChatID, int(brmsg.PlatformMessageID)); err != nil {
			s.log.Error(err, "broadcast delete: platform delete failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			return jobResult{
				result: service_contract.BroadcastResult{
					Platform: job.platform,
					Success:  false,
					Error:    []string{err.Error()},
				},
				target: target,
			}
		}

		s.log.Info("broadcast message deleted",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
		)

		result := service_contract.BroadcastResult{
			Platform: job.platform,
			Success:  true,
		}

		if err := s.chatHisRepo.Delete(ctx, job.chat.ID, brmsg.ID); err != nil {
			result.Success = false
			result.Error = []string{err.Error()}
			s.log.Error(err, "broadcast delete: history cleanup failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			s.log.Warn("message deleted on platform but history row not cleaned up; retry/reconciliation needed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
		}

		return jobResult{result: result, target: target}
	}

	results, _ := s.runJobs(ctx, platformChats, handle)

	var failures int
	for _, r := range results {
		if !r.Success {
			failures++
		}
	}
	if failures > 0 {
		return exception.Wrap(
			exception.ErrInternal,
			fmt.Errorf("%d/%d broadcast delete targets failed", failures, len(results)),
		)
	}
	return nil
}

func (s *broadcastService) UpdateBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req service_contract.UpdateBroadcastRequest) error {
	if len(req.Platforms) == 0 {
		return exception.Wrap(exception.ErrInternal, fmt.Errorf("no platforms specified"))
	}
	if err := s.validatePlatforms(req.Platforms); err != nil {
		return err
	}

	platformChats, _, err := s.chatRepo.GetAllChatsContainsBroadcastMsgUUID(ctx, companyID, broadcastMsgUUID, req.Platforms)
	if err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	handle := func(ctx context.Context, job broadcastJob) jobResult {
		client := s.clients[job.platform]
		target := targetFromChat(job.chat)

		brmsg, err := s.chatHisRepo.GetByBroadcastMsgID(ctx, broadcastMsgUUID, job.chat.ID)
		if err != nil {
			s.log.Error(err, "broadcast delete: history lookup failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			return jobResult{
				result: service_contract.BroadcastResult{
					Platform: job.platform,
					Success:  false,
					Error:    []string{err.Error()},
				},
				target: target,
			}
		}

		if _, err := client.EditMessageText(ctx, job.chat.PlatformChatID, int(brmsg.PlatformMessageID), req.NewMessge.Content); err != nil {
			s.log.Error(err, "broadcast delete: platform delete failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			return jobResult{
				result: service_contract.BroadcastResult{
					Platform: job.platform,
					Success:  false,
					Error:    []string{err.Error()},
				},
				target: target,
			}
		}

		s.log.Info("broadcast message deleted",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
		)

		result := service_contract.BroadcastResult{
			Platform: job.platform,
			Success:  true,
		}

		////
		brmsg.Content = req.NewMessge.Content
		////

		if err := s.chatHisRepo.Upsert(ctx, brmsg); err != nil {
			result.Success = false
			result.Error = []string{err.Error()}
			s.log.Error(err, "broadcast delete: history cleanup failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			s.log.Warn("message deleted on platform but history row not cleaned up; retry/reconciliation needed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
		}

		return jobResult{result: result, target: target}
	}

	results, _ := s.runJobs(ctx, platformChats, handle)

	var failures int
	for _, r := range results {
		if !r.Success {
			failures++
		}
	}
	if failures > 0 {
		return exception.Wrap(
			exception.ErrInternal,
			fmt.Errorf("%d/%d broadcast delete targets failed", failures, len(results)),
		)
	}
	return nil
}
