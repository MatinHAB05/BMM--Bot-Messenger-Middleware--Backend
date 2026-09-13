package service

import (
	"context"
	"errors"
	"fmt"
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/otp"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"

	"gorm.io/datatypes"
)

// chatService implements service_contract.ChatService: company-scoped Chat CRUD,
// plus IngestUpdate, the sole entry point the Telegram/Bale background
// listeners call into. IngestUpdate owns both finding-or-registering the
// Chat row and writing the ChatHistory row, as one unit of business logic
// -- exactly what spec section "Bot Ingestion Flow" asks for, and why the
// listeners themselves stay free of it.
type chatService struct {
	chatRepo        repository_contract.ChatRepository
	chatHistoryRepo repository_contract.ChatHistoryRepository
	otpService      service_contract.OTPService
	companyRepo     repository_contract.CompanyRepository
	log             logger.Logger
}

func NewChatService(
	chatRepo repository_contract.ChatRepository,
	chatHistoryRepo repository_contract.ChatHistoryRepository,

	companyRepo repository_contract.CompanyRepository,
	otpService service_contract.OTPService,
	log logger.Logger) service_contract.ChatService {
	return &chatService{
		chatRepo:        chatRepo,
		chatHistoryRepo: chatHistoryRepo,
		companyRepo:     companyRepo,
		otpService:      otpService,
		log:             log.With(logger.String("component", "chat_service")),
	}
}

func (s *chatService) List(ctx context.Context, companyID uint, query service_contract.ChatListQuery) (*service_contract.ChatListResponse, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	filter := repository_contract.ChatFilter{
		Platform: entity.MessengerPlatform(query.Platform),
		ChatType: entity.ChatType(query.ChatType),
		IsActive: query.IsActive,
	}

	chats, total, err := s.chatRepo.List(ctx, companyID, filter, offset, pageSize)
	if err != nil || (total == nil) {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	responses := make([]service_contract.ChatResponse, 0, len(chats))
	for i := range chats {
		responses = append(responses, service_contract.ToChatResponse(&chats[i]))
	}

	return &service_contract.ChatListResponse{Chats: responses, Total: *total, Page: page, PageSize: pageSize}, nil
}

func (s *chatService) GetByIDInCompany(ctx context.Context, companyID, chatID uint) (*service_contract.ChatResponse, error) {
	chat, err := s.chatRepo.FindByIDInCompany(ctx, companyID, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) GetByID(ctx context.Context, chatID uint) (*service_contract.ChatResponse, error) {
	chat, err := s.chatRepo.FindByID(ctx, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) Create(ctx context.Context, companyID uint, req service_contract.CreateChatRequest) (*service_contract.ChatResponse, error) {
	platform := entity.MessengerPlatform(req.Platform)

	_, err := s.chatRepo.FindByPlatformChatIDInCompany(ctx, companyID, platform, req.PlatformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatAlreadyExists) {
			return nil, exception.ErrChatAlreadyExists
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	chat := &entity.Chat{
		CompanyID:      &companyID,
		Platform:       platform,
		PlatformChatID: req.PlatformChatID,
		Title:          req.Title,
		Username:       req.Username,
		ChatType:       entity.ChatType(req.ChatType),
		IsPrivate:      req.IsPrivate,
		IsActive:       true,
	}
	if err := s.chatRepo.Create(ctx, chat); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat registered", logger.Uint("chat_id", chat.ID), logger.String("platform", req.Platform))

	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) HalfCreate(ctx context.Context, req service_contract.CreateChatRequest) (*service_contract.ChatResponse, error) {
	platform := entity.MessengerPlatform(req.Platform)

	_, err := s.chatRepo.FindByPlatformChatID(ctx, platform, req.PlatformChatID)
	if !(err != nil && errors.Is(err, exception.ErrChatNotFound)) {
		if err == nil {
			return nil, exception.ErrChatAlreadyExists
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	chat := &entity.Chat{
		Platform:       platform,
		PlatformChatID: req.PlatformChatID,
		Title:          req.Title,
		Username:       req.Username,
		ChatType:       entity.ChatType(req.ChatType),
		IsPrivate:      req.IsPrivate,
		IsActive:       true,
	}
	if err := s.chatRepo.HalfCreate(ctx, chat); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat registered", logger.Uint("chat_id", chat.ID), logger.String("platform", req.Platform))

	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) Update(ctx context.Context, companyID, chatID uint, req service_contract.UpdateChatRequest) (*service_contract.ChatResponse, error) {
	chat, err := s.chatRepo.FindByIDInCompany(ctx, companyID, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if req.Title != "" {
		chat.Title = req.Title
	}
	if req.Username != "" {
		chat.Username = req.Username
	}
	if req.IsActive != nil {
		chat.IsActive = *req.IsActive
	}

	if err := s.chatRepo.Update(ctx, chat); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat updated", logger.Uint("chat_id", chatID))

	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) Delete(ctx context.Context, companyID, chatID uint) error {
	_, err := s.chatRepo.FindByIDInCompany(ctx, companyID, chatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return exception.ErrChatNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}
	if err := s.chatRepo.Delete(ctx, companyID, chatID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat deactivated (soft-deleted)", logger.Uint("chat_id", chatID))
	return nil
}

// IngestUpdate finds or registers the Chat for companyID (the bot's
// associated company -- see bootstrap/init.go for how that's resolved),
// then persists the message as a ChatHistory row. Called directly by the
// Telegram/Bale listener callbacks with nothing but I/O-adapted data.
func (s *chatService) IngestUpdate(ctx context.Context, companyID uint, update service_contract.IngestedUpdate) error {
	platform := entity.MessengerPlatform(update.Platform)

	chat, err := s.chatRepo.FindByPlatformChatIDInCompany(ctx, companyID, platform, update.PlatformChatID)
	if err != nil && !errors.Is(err, exception.ErrChatNotFound) {
		return fmt.Errorf("lookup chat: %w", err)
	}

	if errors.Is(err, exception.ErrChatNotFound) {
		chat = &entity.Chat{
			CompanyID:      &companyID,
			Platform:       platform,
			PlatformChatID: update.PlatformChatID,
			Title:          update.ChatTitle,
			Username:       update.ChatUsername,
			ChatType:       entity.ChatType(update.ChatType),
			IsPrivate:      update.IsPrivateChat,
			IsActive:       true,
		}
		if err := s.chatRepo.Create(ctx, chat); err != nil {
			return fmt.Errorf("register chat: %w", err)
		}
		s.log.Info("chat auto-registered from incoming update",
			logger.String("platform", update.Platform), logger.String("platform_chat_id", update.PlatformChatID))
	} else if refreshChatMetadata(chat, update) {
		if err := s.chatRepo.Update(ctx, chat); err != nil {
			s.log.Warn("failed to refresh chat metadata", logger.Err(err), logger.Uint("chat_id", chat.ID))
		}
	}

	message := &entity.ChatHistory{
		ChatID:            chat.ID,
		PlatformMessageID: update.PlatformMessageID,
		SenderID:          update.SenderID,
		SenderName:        update.SenderName,
		Content:           update.Content,
		MediaType:         update.MediaType,
		RawPayload:        datatypes.JSON(update.RawPayload),
		MessageTimestamp:  update.MessageTimestamp,
	}
	if err := s.chatHistoryRepo.Create(ctx, message); err != nil {
		return fmt.Errorf("persist chat history: %w", err)
	}

	return nil
}

func (s *chatService) GetByPlatformIDInCompany(ctx context.Context, companyID uint, platform, platformChatID string) (*service_contract.ChatResponse, error) {
	p := entity.MessengerPlatform(platform)
	chat, err := s.chatRepo.FindByPlatformChatIDInCompany(ctx, companyID, p, platformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) GetByPlatformID(ctx context.Context, platform, platformChatID string) (*service_contract.ChatResponse, error) {
	p := entity.MessengerPlatform(platform)
	chat, err := s.chatRepo.FindByPlatformChatID(ctx, p, platformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) UpdateByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string, req service_contract.UpdateChatRequest) (*service_contract.ChatResponse, error) {
	p := entity.MessengerPlatform(platform)
	chat, err := s.chatRepo.FindByPlatformChatIDInCompany(ctx, companyID, p, platformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return nil, exception.ErrChatNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if req.Title != "" {
		chat.Title = req.Title
	}
	if req.Username != "" {
		chat.Username = req.Username
	}
	if req.IsActive != nil {
		chat.IsActive = *req.IsActive
	}

	if err := s.chatRepo.Update(ctx, chat); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat updated via platform ID", logger.Uint("chat_id", chat.ID), logger.String("platform_chat_id", platformChatID))

	resp := service_contract.ToChatResponse(chat)
	return &resp, nil
}

func (s *chatService) DeleteByPlatformID(ctx context.Context, companyID uint, platform, platformChatID string) error {
	p := entity.MessengerPlatform(platform)
	chat, err := s.chatRepo.FindByPlatformChatIDInCompany(ctx, companyID, p, platformChatID)
	if err != nil {
		if errors.Is(err, exception.ErrChatNotFound) {
			return exception.ErrChatNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.chatRepo.Delete(ctx, companyID, chat.ID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("chat deactivated (soft-deleted) via platform ID", logger.Uint("chat_id", chat.ID), logger.String("platform_chat_id", platformChatID))
	return nil
}

// refreshChatMetadata keeps a Chat's display fields (title/username can
// change over time) in sync with the latest observed update, reporting
// whether anything actually changed so the caller can skip a no-op write.
func refreshChatMetadata(chat *entity.Chat, update service_contract.IngestedUpdate) bool {
	changed := false
	if update.ChatTitle != "" && chat.Title != update.ChatTitle {
		chat.Title = update.ChatTitle
		changed = true
	}
	if update.ChatUsername != "" && chat.Username != update.ChatUsername {
		chat.Username = update.ChatUsername
		changed = true
	}
	return changed
}

func (s *chatService) SendOTP(ctx context.Context, companyCode string) (*service_contract.SendLinkChatOTPResponse, error) {
	_, err := s.companyRepo.FindByCode(ctx, companyCode)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	res, err := s.otpService.SendOTP(ctx, companyCode, otp.TypeLink, nil)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	return &service_contract.SendLinkChatOTPResponse{
		Message:         res.Message,
		ExpiresInSecond: res.ExpiresInSecond,
		Code:            res.Code,
	}, nil
}
func (s *chatService) FeatChatWithOTP(ctx context.Context, companyCode, code, platformChatID, platform string) (*service_contract.ChatResponse, error) {
	chat, err := s.chatRepo.FindByPlatformChatID(ctx, entity.MessengerPlatform(platform), platformChatID)

	s.log.Debug("chat begin", logger.Any("chat", chat))

	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	} else if chat.CompanyID != nil {
		s.log.Warn("chat was already regestred by a company", logger.String("chat-id", platformChatID), logger.Any("current_company_id", chat.CompanyID), logger.String("target_company_code", companyCode))
		return nil, exception.ErrChatCompanyAlreadyRegistered
	}

	com, err := s.companyRepo.FindByCode(ctx, companyCode)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	_, err = s.otpService.VerifyOTP(ctx, companyCode, otp.TypeLink, code)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	err = s.otpService.InvalidateOTP(ctx, companyCode, otp.TypeLink, code)

	new_chat := chat
	new_chat.CompanyID = &com.ID
	s.log.Debug("before update : ", logger.Any("new_chat", new_chat))

	err = s.chatRepo.Update(ctx, chat)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Debug("after update : ", logger.Any("new_chat", new_chat))

	resp := service_contract.ToChatResponse(new_chat)
	s.log.Debug("resp : ", logger.Any("resp", resp))

	return &resp, nil
}
