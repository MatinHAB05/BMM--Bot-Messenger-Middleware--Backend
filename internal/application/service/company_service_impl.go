package service

import (
	"context"
	"errors"
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/otp"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
	"strconv"
)

// companyService implements service_contract.CompanyService. There is no
// cross-company management: Update/Delete verify the caller's own
// companyID (from their token) matches the path id before touching
// anything, since there is no global superuser role in this system.
type companyService struct {
	companyRepo repository_contract.CompanyRepository
	otpService  service_contract.OTPService
	log         logger.Logger
}

func NewCompanyService(
	companyRepo repository_contract.CompanyRepository,
	otpService service_contract.OTPService,
	log logger.Logger,
) service_contract.CompanyService {
	return &companyService{
		companyRepo: companyRepo,
		otpService:  otpService,
		log:         log.With(logger.String("component", "company_service")),
	}
}

func (s *companyService) Create(ctx context.Context, req service_contract.CreateCompanyRequest) (*service_contract.CompanyResponse, error) {
	_, err := s.companyRepo.FindByCode(ctx, req.Code)
	if err != nil {
		if errors.Is(err, exception.ErrChatAlreadyExists) {
			return nil, exception.ErrChatAlreadyExists
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	com := &entity.Company{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		IsActive:    true,
	}
	if err := s.companyRepo.Create(ctx, com); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("company registered", logger.Uint("company_id", com.ID), logger.String("code", com.Code))

	resp := service_contract.ToCompanyResponse(com)
	return &resp, nil
}

func (s *companyService) Me(ctx context.Context, companyID uint) (*service_contract.CompanyResponse, error) {
	company, err := s.companyRepo.FindByID(ctx, companyID)
	if err != nil {
		if errors.Is(err, exception.ErrCompanyNotFound) {
			return nil, exception.ErrCompanyNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := service_contract.ToCompanyResponse(company)
	return &resp, nil
}

func (s *companyService) Update(ctx context.Context, callerCompanyID, targetCompanyID uint, req service_contract.UpdateCompanyRequest) (*service_contract.CompanyResponse, error) {
	if callerCompanyID != targetCompanyID {
		// No global superuser: admin privileges never cross a company
		// boundary. Reported as "not found" rather than "forbidden" so a
		// caller can't use this endpoint to confirm another company's id
		// exists.
		return nil, exception.ErrCompanyNotFound
	}

	company, err := s.companyRepo.FindByID(ctx, targetCompanyID)
	if err != nil {
		if errors.Is(err, exception.ErrCompanyNotFound) {
			return nil, exception.ErrCompanyNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if req.Name != "" {
		company.Name = req.Name
	}
	if req.Code != "" {
		company.Code = req.Code
	}
	if req.IsActive != nil {
		company.IsActive = *req.IsActive
	}
	company.Description = req.Description

	if err := s.companyRepo.Update(ctx, company); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("company updated", logger.Uint("company_id", company.ID))

	resp := service_contract.ToCompanyResponse(company)
	return &resp, nil
}

func (s *companyService) Delete(ctx context.Context, callerCompanyID, targetCompanyID uint) error {
	if callerCompanyID != targetCompanyID {
		return exception.ErrCompanyNotFound
	}

	_, err := s.companyRepo.FindByID(ctx, targetCompanyID)
	if err != nil {
		if errors.Is(err, exception.ErrCompanyNotFound) {
			return exception.ErrCompanyNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.companyRepo.Delete(ctx, targetCompanyID); err != nil {
		if errors.Is(err, exception.ErrCompanyNotFound) {
			return exception.ErrCompanyNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("company deactivated (soft-deleted)", logger.Uint("company_id", targetCompanyID))
	return nil
}

func (s *companyService) SendRegistionrWithOTP(ctx context.Context, companyID uint, req service_contract.SendRegistionrWithOTPRequest) (*service_contract.SendRegistionrWithOTPResponse, error) {
	r, err := s.otpService.SendOTP(ctx, "=dose not matter=", otp.TypeRegistrionUserToCompany, map[string]any{
		"roles": req.Roles, "company_id": strconv.FormatUint(uint64(companyID), 10),
	})
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := service_contract.SendRegistionrWithOTPResponse{
		Message:         r.Message,
		ExpiresInSecond: r.ExpiresInSecond,
		Code:            r.Code,
	}
	return &resp, nil
}
