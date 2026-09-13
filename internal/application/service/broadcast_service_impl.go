package service

import (
	"context"
	"fmt"
	"sync"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
	"messenger-backend/pkg/messenger"

	"gorm.io/datatypes"
)

// todo : mq :)
type BroadcastConfig struct {
	WorkerCount int
	QueueSize   int
}

type broadcastJob struct {
	platform string
	chat     entity.Chat
}

type broadcastService struct {
	clients     map[string]messenger.MessengerClient
	chatRepo    repository_contract.ChatRepository
	chatHisRepo repository_contract.ChatHistoryRepository
	log         logger.Logger
	cfg         BroadcastConfig
}

func NewBroadcastService(
	clients []messenger.MessengerClient,
	chatRepo repository_contract.ChatRepository,
	chatHisRepo repository_contract.ChatHistoryRepository,
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
		clients:     registry,
		chatRepo:    chatRepo,
		chatHisRepo: chatHisRepo,
		log:         log.With(logger.String("component", "broadcast_service")),
		cfg:         cfg,
	}
}

func (s *broadcastService) Broadcast(ctx context.Context, companyID uint, req service_contract.BroadcastRequest) (*service_contract.BroadcastResponse, error) {
	for _, platform := range req.Platforms {
		if _, ok := s.clients[platform]; !ok {
			return nil, exception.Wrap(
				exception.ErrUnsupportedPlatform,
				fmt.Errorf("no client registered for platform %q", platform),
			)
		}
	}

	platformChats, _, err := s.chatRepo.ListAll(ctx, companyID, req.Platforms)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	jobQueue := make(chan broadcastJob, s.cfg.QueueSize)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []service_contract.BroadcastResult
		targets []struct {
			Name      string
			Platform  string
			IsPrivate bool
			Type      string
		}
	)

	for i := 0; i < s.cfg.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobQueue {
				client := s.clients[job.platform]

				mes, err := client.SendMessage(ctx, job.chat.PlatformChatID, req.Message)

				result := service_contract.BroadcastResult{
					Platform: job.platform,
					Success:  err == nil,
				}

				target := struct {
					Name      string
					Platform  string
					IsPrivate bool
					Type      string
				}{
					Name:      job.chat.Title,
					Platform:  string(job.chat.Platform),
					IsPrivate: job.chat.IsPrivate,
					Type:      string(job.chat.ChatType),
				}

				if err != nil {
					result.Error = err.Error()
					s.log.Error(err, "broadcast delivery failed",
						logger.String("platform", job.platform),
						logger.String("target_id", job.chat.PlatformChatID),
					)
				} else {
					s.log.Info("broadcast delivered",
						logger.String("platform", job.platform),
						logger.String("target_id", job.chat.PlatformChatID),
					)

					err = s.chatHisRepo.Create(ctx, &entity.ChatHistory{
						ChatID:            job.chat.ID,
						PlatformMessageID: mes.PlatformMessageID,
						SenderID:          mes.SenderID,
						SenderName:        mes.SenderName,
						Content:           mes.Content,
						MediaType:         mes.MediaType,
						RawPayload:        datatypes.JSON(mes.RawPayload),
						MessageTimestamp:  mes.Timestamp,
					})

					if err != nil {
						result.Error = err.Error()
						s.log.Error(err, "broadcast delivery failed",
							logger.String("platform", job.platform),
							logger.String("target_id", job.chat.PlatformChatID),
						)

						//todo : mq to retry
						s.log.Warn("broadcast delivery failed - telgram recived message but db didnt",
							logger.String("platform", job.platform),
							logger.String("target_id", job.chat.PlatformChatID),
						)
					}
				}

				mu.Lock()
				results = append(results, result)
				targets = append(targets, target)
				mu.Unlock()
			}
		}()
	}

	go func() {
		for platform, chats := range platformChats {
			for _, chat := range chats {
				select {
				case <-ctx.Done():
					close(jobQueue)
					return
				case jobQueue <- broadcastJob{platform: platform, chat: chat}:
				}
			}
		}
		close(jobQueue)
	}()

	wg.Wait()

	return &service_contract.BroadcastResponse{
		Results: results,
		Targets: targets,
	}, nil
}
