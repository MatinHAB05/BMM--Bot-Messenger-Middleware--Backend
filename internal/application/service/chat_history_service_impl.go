package service

import (
	"context"
	"errors"
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
)

// chatHistoryService implements service_contract.ChatHistoryService.
// ChatHistory rows have no companyID column of their own -- their tenant
// is implied by their parent Chat -- so every method here first confirms
// chatID belongs to companyID before touching any message rows.
type chatHistoryService struct {
	chatRepo        repository_contract.ChatRepository
	chatHistoryRepo repository_contract.ChatHistoryRepository
	log             logger.Logger
}

func NewChatHistoryService(chatRepo repository_contract.ChatRepository, chatHistoryRepo repository_contract.ChatHistoryRepository, log logger.Logger) service_contract.ChatHistoryService {
	return &chatHistoryService{
		chatRepo:        chatRepo,
		chatHistoryRepo: chatHistoryRepo,
		log:             log.With(logger.String("component", "chat_history_service")),
	}
}

func (s *chatHistoryService) Create(ctx context.Context, message *service_contract.CreateMessageRequest) (*service_contract.ChatHistoryDetailResponse, error) {
	//todo check retuen err if already exists

	mess := &entity.ChatHistory{
		ChatID:            message.ChatID,
		PlatformMessageID: message.PlatformMessageID,
		SenderID:          message.SenderID,
		SenderName:        message.SenderName,
		Content:           message.Content,
		MediaType:         message.MediaType,
		RawPayload:        message.RawPayload,
		MessageTimestamp:  message.MessageTimestamp,
	}
	if err := s.chatHistoryRepo.Create(ctx, mess); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat registered", logger.Uint("chat_id", mess.ChatID), logger.Int64("platform_message_id", mess.PlatformMessageID))

	resp := service_contract.ToChatHistoryDetailResponse(mess)
	return &resp, nil
}

func (s *chatHistoryService) Upsert(ctx context.Context, message *service_contract.CreateMessageRequest) (*service_contract.ChatHistoryDetailResponse, error) {
	//todo check retuen err if already exists

	mess := &entity.ChatHistory{
		ChatID:            message.ChatID,
		PlatformMessageID: message.PlatformMessageID,
		SenderID:          message.SenderID,
		SenderName:        message.SenderName,
		Content:           message.Content,
		MediaType:         message.MediaType,
		RawPayload:        message.RawPayload,
		MessageTimestamp:  message.MessageTimestamp,
	}
	m, err := s.chatHistoryRepo.FindByPlatformMessgeID(ctx, mess.ChatID, uint(message.PlatformMessageID))

	if err != nil && !errors.Is(err, exception.ErrMessageNotFound) {
		return nil, exception.Wrap(exception.ErrInternal, err)
	} else if err == nil {
		mess.ID = m.ID
		if err := s.chatHistoryRepo.Upsert(ctx, mess); err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
	} else {
		if err := s.chatHistoryRepo.Create(ctx, mess); err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
	}

	s.log.Info("chat registered", logger.Uint("chat_id", mess.ChatID), logger.Int64("platform_message_id", mess.PlatformMessageID))

	resp := service_contract.ToChatHistoryDetailResponse(mess)
	return &resp, nil
}

func (s *chatHistoryService) List(ctx context.Context, companyID, chatID uint, query service_contract.ChatHistoryListQuery) (*service_contract.ChatHistoryListResponse, error) {
	if err := s.requireChatInCompany(ctx, companyID, chatID); err != nil {
		return nil, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filter := repository_contract.ChatHistoryFilter{
		Search:    query.Search,
		MediaType: query.MediaType,
		FromDate:  query.FromDate,
		ToDate:    query.ToDate,
	}

	messages, total, err := s.chatHistoryRepo.List(ctx, chatID, filter, offset, pageSize)
	if err != nil || total == nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	responses := make([]service_contract.ChatHistoryResponse, 0, len(messages))
	for i := range messages {
		responses = append(responses, service_contract.ToChatHistoryResponse(&messages[i]))
	}

	return &service_contract.ChatHistoryListResponse{Messages: responses, Total: *total, Page: page, PageSize: pageSize}, nil
}

func (s *chatHistoryService) GetByIDInCompany(ctx context.Context, companyID, chatID, messageID uint) (*service_contract.ChatHistoryDetailResponse, error) {
	if err := s.requireChatInCompany(ctx, companyID, chatID); err != nil {
		return nil, err
	}

	message, err := s.chatHistoryRepo.FindByID(ctx, chatID, messageID)
	if err != nil {
		if errors.Is(err, exception.ErrMessageNotFound) {
			return nil, exception.ErrMessageNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToChatHistoryDetailResponse(message)
	return &resp, nil
}

func (s *chatHistoryService) GetByID(ctx context.Context, chatID, messageID uint) (*service_contract.ChatHistoryDetailResponse, error) {
	if err := s.requireChat(ctx, chatID); err != nil {
		return nil, err
	}

	message, err := s.chatHistoryRepo.FindByID(ctx, chatID, messageID)
	if err != nil {
		if errors.Is(err, exception.ErrMessageNotFound) {
			return nil, exception.ErrMessageNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToChatHistoryDetailResponse(message)
	return &resp, nil
}

func (s *chatHistoryService) Delete(ctx context.Context, companyID, chatID, messageID uint) error {
	if err := s.requireChatInCompany(ctx, companyID, chatID); err != nil {
		return err
	}

	_, err := s.chatHistoryRepo.FindByID(ctx, chatID, messageID)
	if err != nil {
		if errors.Is(err, exception.ErrMessageNotFound) {
			return exception.ErrMessageNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.chatHistoryRepo.Delete(ctx, chatID, messageID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat history message deleted", logger.Uint("chat_id", chatID), logger.Uint("message_id", messageID))
	return nil
}

func (s *chatHistoryService) ListByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, query service_contract.ChatHistoryListQuery) (*service_contract.ChatHistoryListResponse, error) {
	chat, err := s.requireChatInCompanyByPlatform(ctx, companyID, platform, platformChatID)
	if err != nil {
		return nil, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filter := repository_contract.ChatHistoryFilter{
		Search:    query.Search,
		MediaType: query.MediaType,
		FromDate:  query.FromDate,
		ToDate:    query.ToDate,
	}

	messages, total, err := s.chatHistoryRepo.List(ctx, chat.ID, filter, offset, pageSize)
	if err != nil || total == nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	responses := make([]service_contract.ChatHistoryResponse, 0, len(messages))
	for i := range messages {
		responses = append(responses, service_contract.ToChatHistoryResponse(&messages[i]))
	}

	return &service_contract.ChatHistoryListResponse{Messages: responses, Total: *total, Page: page, PageSize: pageSize}, nil
}

func (s *chatHistoryService) GetByIDInCompanyByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, messageID uint) (*service_contract.ChatHistoryDetailResponse, error) {
	chat, err := s.requireChatInCompanyByPlatform(ctx, companyID, platform, platformChatID)
	if err != nil {
		return nil, err
	}

	message, err := s.chatHistoryRepo.FindByID(ctx, chat.ID, messageID)
	if err != nil {
		if errors.Is(err, exception.ErrMessageNotFound) {
			return nil, exception.ErrMessageNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToChatHistoryDetailResponse(message)
	return &resp, nil
}

func (s *chatHistoryService) GetByIDByPlatformID(ctx context.Context, platform, platformChatID string, messageID uint) (*service_contract.ChatHistoryDetailResponse, error) {
	chat, err := s.requireChatByPlatform(ctx, platform, platformChatID)
	if err != nil {
		return nil, err
	}

	message, err := s.chatHistoryRepo.FindByID(ctx, chat.ID, messageID)
	if err != nil {
		if errors.Is(err, exception.ErrMessageNotFound) {
			return nil, exception.ErrMessageNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToChatHistoryDetailResponse(message)
	return &resp, nil
}

func (s *chatHistoryService) DeleteByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, messageID uint) error {
	chat, err := s.requireChatInCompanyByPlatform(ctx, companyID, platform, platformChatID)
	if err != nil {
		return err
	}

	_, err = s.chatHistoryRepo.FindByID(ctx, chat.ID, messageID)
	if err != nil {
		if errors.Is(err, exception.ErrMessageNotFound) {
			return exception.ErrMessageNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.chatHistoryRepo.Delete(ctx, chat.ID, messageID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat history message deleted via platform ID", logger.Uint("chat_id", chat.ID), logger.String("platform_chat_id", platformChatID), logger.Uint("message_id", messageID))
	return nil
}

func (s *chatHistoryService) requireChatInCompanyByPlatform(ctx context.Context, companyID uint, platform, platformChatID string) (*entity.Chat, error) {
	p := entity.MessengerPlatform(platform)
	chat, err := s.chatRepo.FindByPlatformChatIDInCompany(ctx, companyID, p, platformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	return chat, nil
}

func (s *chatHistoryService) requireChatByPlatform(ctx context.Context, platform, platformChatID string) (*entity.Chat, error) {
	p := entity.MessengerPlatform(platform)
	chat, err := s.chatRepo.FindByPlatformChatID(ctx, p, platformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	return chat, nil
}

func (s *chatHistoryService) requireChatInCompany(ctx context.Context, companyID, chatID uint) error {
	_, err := s.chatRepo.FindByIDInCompany(ctx, companyID, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return exception.ErrChatNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	return nil
}
func (s *chatHistoryService) requireChat(ctx context.Context, chatID uint) error {
	_, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return exception.ErrChatNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	return nil
}
