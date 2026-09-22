package service

import (
	"bytes"
	"context"
	"fmt"
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

type broadcastService struct {
	clients            map[string]messenger.MessengerClient
	chatRepo           repository_contract.ChatRepository
	chatHisRepo        repository_contract.ChatHistoryRepository
	sentbalemsgService service_contract.SentBaleMsgService
	log                logger.Logger
	cfg                BroadcastConfig
	pool               *ants.Pool
}

func NewBroadcastService(
	clients []messenger.MessengerClient,
	chatRepo repository_contract.ChatRepository,
	chatHisRepo repository_contract.ChatHistoryRepository,
	sentbalemsgService service_contract.SentBaleMsgService,
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
// than BroadcastHandler (which already validates type/size on the way
// in) -- e.g. anything invoking BroadcastService directly.
func validateBroadcastAttachment(a service_contract.BroadcastAttachment) error {
	switch a.Type {
	case service_contract.BroadcastAttachmentPhoto,
		service_contract.BroadcastAttachmentVideo,
		service_contract.BroadcastAttachmentVoice,
		service_contract.BroadcastAttachmentDocument,
		service_contract.BroadcastAttachmentAnimation:
	default:
		return fmt.Errorf("unsupported attachment type %q", a.Type)
	}
	if len(a.Data) == 0 {
		return fmt.Errorf("attachment data is empty")
	}
	return nil
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

	handle := func(ctx context.Context, job broadcastJob) jobResult {
		client := s.clients[job.platform]
		target := targetFromChat(job.chat)

		var (
			mes *messenger.MessageUpdate
			err error
		)

		if req.Attachment != nil {
			// Every job in this broadcast reads the same buffered bytes
			// concurrently, and an io.Reader can only be drained once --
			// so each job gets its own bytes.Reader over req.Attachment.Data
			// rather than sharing one.
			mes, err = client.SendAttachment(ctx, job.chat.PlatformChatID, req.Message, messenger.Attachment{
				Type:     messenger.AttachmentType(req.Attachment.Type),
				FileName: req.Attachment.FileName,
				Data:     bytes.NewReader(req.Attachment.Data),
			})
		} else {
			mes, err = client.SendMessage(ctx, job.chat.PlatformChatID, req.Message)
		}

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
			// Attachments: , //todo
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

		err = s.sentbalemsgService.SetSentBaleMsg(ctx, job.chat.PlatformChatID, req.Message) // ? : FUCK BALE!
		if err != nil {
			result.Success = false
			result.Error = append(result.Error, err.Error())
		}
		return jobResult{result: result, target: target}
	}

	results, targets := s.runJobs(ctx, platformChats, handle)

	return &service_contract.BroadcastResponse{
		Results: results,
		Targets: targets,
	}, nil
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
