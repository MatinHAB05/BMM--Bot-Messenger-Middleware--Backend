package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// TODO : mq :)
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
	pool               *ants.Pool // per-chat fan-out (runJobs)
	batchPool          *ants.Pool // per-broadcast-UUID fan-out (DeleteBroadcastBatch) -- kept separate from pool so a batch job blocked waiting on its own chat-level jobs can never exhaust the workers those chat-level jobs need; see DeleteBroadcastBatch's doc comment.
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
	batchPool *ants.Pool,
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
		batchPool:          batchPool,
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

// GetBroadcastPlatforms returns the distinct platforms that the given
// broadcast UUID was actually delivered to, by querying the DB with no
// platform filter so every matching chat row is returned.
func (s *broadcastService) GetBroadcastPlatforms(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID) ([]string, error) {
	// passing nil/empty platforms = no IN-filter → returns ALL platforms
	platformChats, _, err := s.chatRepo.GetAllChatsContainsBroadcastMsgUUID(ctx, companyID, broadcastMsgUUID, nil)
	if err != nil {
		return nil, err
	}
	platforms := make([]string, 0, len(platformChats))
	for p := range platformChats {
		platforms = append(platforms, p)
	}
	return platforms, nil
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

// mapMessengerErr classifies an error returned by a messenger.MessengerClient
// call (SendMessage, SendAttachment, EditMessageText, DeleteMessage) into
// this application's own exception.AppError vocabulary, so a broadcast
// failure carries a stable Code/HTTPStatus like everything else this
// service returns, instead of an opaque platform-adapter error string.
//
// The classification itself is done against pkg/messenger's
// engine-agnostic sentinels (messenger.ErrorForbidden and its
// siblings) -- every concrete engine adapter (telegram, bale, ...)
// already maps its own SDK's errors onto those, so this is the one
// place in the service layer that needs to know about them. An error
// that doesn't match any sentinel (a non-platform error, e.g. a
// context deadline) falls back to the pre-existing generic
// ErrSendMessagePlatform, unchanged from before this mapping existed.
// A nil err returns nil.
func mapMessengerErr(err error) *exception.AppError {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, messenger.ErrorForbidden):
		return exception.ErrPlatformForbidden
	case errors.Is(err, messenger.ErrorBadRequest):
		return exception.ErrPlatformBadRequest
	case errors.Is(err, messenger.ErrorUnauthorized):
		return exception.ErrPlatformUnauthorized
	case errors.Is(err, messenger.ErrorTooManyRequests):
		return exception.ErrPlatformRateLimited
	case errors.Is(err, messenger.ErrorNotFound):
		return exception.ErrPlatformNotFound
	case errors.Is(err, messenger.ErrorConflict):
		return exception.ErrPlatformConflict
	default:
		return exception.ErrSendMessagePlatform
	}
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
		appErr := mapMessengerErr(err)
		s.log.Error(appErr, "broadcast send failed",
			logger.String("platform", job.platform),
			logger.String("target_id", job.chat.PlatformChatID),
		)
		return jobResult{
			result: service_contract.BroadcastResult{
				Platform: job.platform,
				Success:  false,
				Error:    []string{appErr.Error()},
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
			appErr := mapMessengerErr(err)
			s.log.Error(appErr, "broadcast attachment send failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
				logger.String("file", f.fileName),
			)
			result.Success = false
			result.Error = append(result.Error, fmt.Sprintf("%s: %s", f.fileName, appErr.Error()))
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
			HasAttachments:    true,
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

		// Each attachment is a real, separate platform message (see the
		// file loop above) -- so it needs its own "sent" record, not one
		// shared record for the whole batch. The hash ties the record to
		// exactly what was sent for THIS message: the broadcast's caption
		// plus this file's own bytes, so two different attachments (or the
		// same attachment resent under a different caption) never collide
		// on the same value.
		sentHash := attachmentSentMsgHash(caption, f.data)
		if err := s.sentbalemsgService.SetSentBaleMsg(ctx, job.chat.PlatformChatID, sentHash); err != nil { // ? : FUCK BALE!
			result.Success = false
			result.Error = append(result.Error, fmt.Sprintf("%s: %s", f.fileName, err.Error()))
		}
	}

	return jobResult{result: result, target: target}
}

// attachmentSentMsgHash produces a stable, content-addressed identifier for
// one broadcast attachment message: sha256 of the broadcast caption
// concatenated with this file's raw bytes, hex-encoded. Used in place of
// the plain caption so SetSentBaleMsg gets a value unique to each attachment
// message instead of the same caption repeated for every file in the batch.
func attachmentSentMsgHash(caption string, data []byte) string {
	h := sha256.New()
	h.Write([]byte(caption))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *broadcastService) DeleteBroadcast(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, req service_contract.DeleteBroadcastRequest) error {
	if len(req.Platforms) == 0 {
		return exception.Wrap(exception.ErrInternal, fmt.Errorf("no platforms specified"))
	}
	if err := s.validatePlatforms(req.Platforms); err != nil {
		return err
	}

	return s.deleteBroadcastByUUID(ctx, companyID, broadcastMsgUUID, req.Platforms)
}

// DeleteBroadcastBatch deletes several broadcasts (each identified by its
// own UUID) in one call. Every UUID gets the full deleteBroadcastByUUID
// treatment independently -- its own chat lookup, its own per-chat/
// per-message fan-out -- submitted to s.batchPool, a pool dedicated to
// this batch level and kept separate from s.pool (which deleteBroadcastByUUID's
// own runJobs call uses for its per-chat jobs). Sharing one pool across
// both levels would deadlock: an outer batch job would hold a worker for
// its entire duration while waiting on inner chat-level jobs that need a
// free worker from that same pool to run at all. Two separate pools rule
// that out entirely, regardless of batch size or pool capacity.
//
// A failure on one broadcast UUID does not stop the rest from being
// processed; every failure is collected and reported together at the end.
func (s *broadcastService) DeleteBroadcastBatch(ctx context.Context, companyID uint, req service_contract.DeleteBroadcastBatchRequest) error {
	if len(req.Platforms) == 0 {
		return exception.Wrap(exception.ErrInternal, fmt.Errorf("no platforms specified"))
	}
	if len(req.BroadcastIDS) == 0 {
		return exception.Wrap(exception.ErrInternal, fmt.Errorf("no broadcast ids specified"))
	}
	if err := s.validatePlatforms(req.Platforms); err != nil {
		return err
	}

	var (
		mu       sync.Mutex
		failures int
		errs     []string
	)

	var wg sync.WaitGroup

	for _, broadcastMsgUUID := range req.BroadcastIDS {
		broadcastMsgUUID := broadcastMsgUUID // capture per-iteration value
		wg.Add(1)

		err := s.batchPool.Submit(func() {
			defer wg.Done()

			if err := s.deleteBroadcastByUUID(ctx, companyID, broadcastMsgUUID, req.Platforms); err != nil {
				s.log.Error(err, "broadcast batch delete: broadcast failed",
					logger.String("broadcast_uuid", broadcastMsgUUID.String()),
				)
				mu.Lock()
				failures++
				errs = append(errs, fmt.Sprintf("%s: %s", broadcastMsgUUID, err.Error()))
				mu.Unlock()
			}
		})

		if err != nil {
			wg.Done()
			s.log.Error(err, "failed to submit broadcast batch delete job to ants pool",
				logger.String("broadcast_uuid", broadcastMsgUUID.String()),
			)
			mu.Lock()
			failures++
			errs = append(errs, fmt.Sprintf("%s: failed to submit job: %s", broadcastMsgUUID, err.Error()))
			mu.Unlock()
		}
	}

	wg.Wait()

	if failures > 0 {
		return exception.Wrap(
			exception.ErrInternal,
			fmt.Errorf("%d/%d broadcasts failed to delete: %s", failures, len(req.BroadcastIDS), strings.Join(errs, "; ")),
		)
	}
	return nil
}

// deleteBroadcastByUUID deletes every message belonging to one broadcast
// (one UUID) across every target chat: this is DeleteBroadcast's original
// body, factored out so DeleteBroadcast and DeleteBroadcastBatch share it
// instead of duplicating the fan-out/cleanup logic.
func (s *broadcastService) deleteBroadcastByUUID(ctx context.Context, companyID uint, broadcastMsgUUID uuid.UUID, platforms []string) error {
	platformChats, _, err := s.chatRepo.GetAllChatsContainsBroadcastMsgUUID(ctx, companyID, broadcastMsgUUID, platforms)
	if err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	handle := func(ctx context.Context, job broadcastJob) jobResult {
		client := s.clients[job.platform]
		target := targetFromChat(job.chat)

		// A multi-attachment broadcast sent N real, separate platform
		// messages to this chat -- all sharing broadcastMsgUUID. Every one
		// of them has to be deleted individually; GetByBroadcastMsgID
		// (singular) would silently leave the rest behind.
		brmsgs, err := s.chatHisRepo.GetAllByBroadcastMsgID(ctx, broadcastMsgUUID, job.chat.ID)
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

		result := service_contract.BroadcastResult{
			Platform: job.platform,
			Success:  true,
		}

		for _, brmsg := range brmsgs {
			if err := client.DeleteMessage(ctx, job.chat.PlatformChatID, int(brmsg.PlatformMessageID)); err != nil {
				appErr := mapMessengerErr(err)
				result.Success = false
				result.Error = append(result.Error, appErr.Error())
				s.log.Error(appErr, "broadcast delete: platform delete failed",
					logger.String("platform", job.platform),
					logger.String("target_id", job.chat.PlatformChatID),
				)
				// keep going -- one failed file shouldn't stop the rest
				// of this chat's messages from being cleaned up
				continue
			}

			s.log.Info("broadcast message deleted",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)

			if err := s.chatHisRepo.Delete(ctx, job.chat.ID, brmsg.ID); err != nil {
				result.Success = false
				result.Error = append(result.Error, err.Error())
				s.log.Error(err, "broadcast delete: history cleanup failed",
					logger.String("platform", job.platform),
					logger.String("target_id", job.chat.PlatformChatID),
				)
				s.log.Warn("message deleted on platform but history row not cleaned up; retry/reconciliation needed",
					logger.String("platform", job.platform),
					logger.String("target_id", job.chat.PlatformChatID),
				)
			}
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

		// Same one-broadcastUUID-to-many-messages situation as Delete: a
		// multi-attachment broadcast left several rows for this chat. Only
		// the first one (ordered by id ASC) ever carried text/caption --
		// see sendAttachmentBroadcastJob's i==0 rule -- so that's the only
		// one there's anything to edit on; the rest were sent without a
		// caption and have nothing to update.
		brmsgs, err := s.chatHisRepo.GetAllByBroadcastMsgID(ctx, broadcastMsgUUID, job.chat.ID)
		if err != nil {
			s.log.Error(err, "broadcast update: history lookup failed",
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

		brmsg := &brmsgs[0]

		if _, err := client.EditMessageText(ctx, job.chat.PlatformChatID, int(brmsg.PlatformMessageID), req.NewMessge.Content, brmsg.HasAttachments); err != nil {
			appErr := mapMessengerErr(err)
			s.log.Error(appErr, "broadcast update: platform edit failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			return jobResult{
				result: service_contract.BroadcastResult{
					Platform: job.platform,
					Success:  false,
					Error:    []string{appErr.Error()},
				},
				target: target,
			}
		}

		s.log.Info("broadcast message updated",
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
			s.log.Error(err, "broadcast update: history persist failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			s.log.Warn("message edited on platform but history row not updated; retry/reconciliation needed",
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
