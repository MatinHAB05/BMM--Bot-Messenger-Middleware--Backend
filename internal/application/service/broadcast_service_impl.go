package service

import (
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
	"gorm.io/datatypes"
)

// todo : mq :)
type BroadcastConfig struct {
	WorkerCount int
	QueueSize   int
	// JobTimeout bounds how long a single per-recipient job (one
	// SendMessage/DeleteMessge + one repo call) is allowed to run.
	// Zero disables the timeout. Without this, one slow/hanging
	// client call can tie up a worker (and, in the worst case, the
	// whole pool) for the lifetime of the parent context.
	JobTimeout time.Duration
}

// service_contract.BroadcastTarget describes a single recipient a broadcast/delete
// operation was attempted against. Structurally identical to (and
// therefore assignable to) the anonymous struct type used by
// service_contract.BroadcastResponse.Targets.
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

// jobResult is what a worker produces for a single job. Feeding these
// through a channel to a single collector goroutine avoids the need
// for a mutex around shared result/target slices.
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
}

func NewBroadcastService(
	clients []messenger.MessengerClient,
	chatRepo repository_contract.ChatRepository,
	chatHisRepo repository_contract.ChatHistoryRepository,
	sentbalemsgService service_contract.SentBaleMsgService,
	log logger.Logger,
	cfg BroadcastConfig,
) service_contract.BroadcastService {
	registry := make(map[string]messenger.MessengerClient, len(clients))
	for _, c := range clients {
		registry[c.Platform()] = c
	}

	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 5
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}

	return &broadcastService{
		clients:            registry,
		chatRepo:           chatRepo,
		chatHisRepo:        chatHisRepo,
		sentbalemsgService: sentbalemsgService,
		log:                log.With(logger.String("component", "broadcast_service")),
		cfg:                cfg,
	}
}

// validatePlatforms checks that every requested platform has a
// registered client, shared by both Broadcast and DeleteBroadcast.
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

// runJobs contains the worker-pool plumbing shared by Broadcast and
// DeleteBroadcast: it fans platformChats out into jobs, runs
// s.cfg.WorkerCount workers that each call handle for every job, and
// collects the results without any shared-slice mutex.
func (s *broadcastService) runJobs(
	ctx context.Context,
	platformChats map[string][]entity.Chat,
	handle func(ctx context.Context, job broadcastJob) jobResult,
) ([]service_contract.BroadcastResult, []service_contract.BroadcastTarget) {
	jobQueue := make(chan broadcastJob, s.cfg.QueueSize)
	resultQueue := make(chan jobResult, s.cfg.QueueSize)

	var wg sync.WaitGroup
	for i := 0; i < s.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobQueue {
				jobCtx := ctx
				if s.cfg.JobTimeout > 0 {
					var cancel context.CancelFunc
					jobCtx, cancel = context.WithTimeout(ctx, s.cfg.JobTimeout)
					resultQueue <- handle(jobCtx, job)
					cancel()
					continue
				}
				resultQueue <- handle(jobCtx, job)
			}
		}()
	}

	// Producer: feed jobs, respecting cancellation.
	go func() {
		defer close(jobQueue)
		for platform, chats := range platformChats {
			for _, chat := range chats {
				select {
				case <-ctx.Done():
					return
				case jobQueue <- broadcastJob{platform: platform, chat: chat}:
				}
			}
		}
	}()

	// Close resultQueue once every worker has finished, so the
	// collector loop below terminates.
	go func() {
		wg.Wait()
		close(resultQueue)
	}()

	var results []service_contract.BroadcastResult
	var targets []service_contract.BroadcastTarget
	for r := range resultQueue {
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

	if err := s.validatePlatforms(req.Platforms); err != nil {
		return nil, err
	}

	platformChats, _, err := s.chatRepo.ListAll(ctx, companyID, req.Platforms)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	// One UUID per broadcast call, shared by every recipient's
	// history row. This is what lets DeleteBroadcast later find and
	// unsend every message that belongs to this broadcast — each
	// history row must NOT get its own random UUID.
	broadcastUUID := uuid.New()

	handle := func(ctx context.Context, job broadcastJob) jobResult {
		client := s.clients[job.platform]
		target := targetFromChat(job.chat)

		mes, err := client.SendMessage(ctx, job.chat.PlatformChatID, req.Message)
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
			// The message was actually delivered on the platform;
			// only persisting the history row failed. Surface it as
			// a partial failure rather than a clean success, since
			// this row won't be reachable by DeleteBroadcast later.
			result.Success = false
			result.Error = append(result.Error, err.Error())
			s.log.Error(err, "broadcast history persist failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			// todo: mq to retry persisting history for delivered-but-unrecorded messages
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
			// The platform message was deleted; only the local
			// history row failed to update. Surface as a partial
			// failure so it can be reconciled later.
			result.Success = false
			result.Error = []string{err.Error()}
			s.log.Error(err, "broadcast delete: history cleanup failed",
				logger.String("platform", job.platform),
				logger.String("target_id", job.chat.PlatformChatID),
			)
			// todo: mq to retry cleaning up history for delivered-delete-but-unrecorded rows
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
