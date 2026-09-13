// internal/application/service/rbac_service_impl.go
package service

import (
	"context"
	"strings"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/logger"
)

type rbacService struct {
	rbacRepo repository_contract.RBACRepository
	log      logger.Logger
}

func NewRBACService(rbacRepo repository_contract.RBACRepository, log logger.Logger) service_contract.RBACService {
	return &rbacService{
		rbacRepo: rbacRepo,
		log:      log.With(logger.String("component", "rbac_service")),
	}
}

// ==========================================
// Role Management
// ==========================================

func (s *rbacService) AddRoleForUser(ctx context.Context, req service_contract.AddRoleForUserRequest) error {
	if err := s.rbacRepo.AddRoleForUser(ctx, req.User, req.Role, req.CompanyID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("role added for user",
		logger.String("user", req.User),
		logger.String("role", req.Role),
		logger.String("company_id", req.CompanyID),
	)
	return nil
}

func (s *rbacService) RemoveRoleFromUser(ctx context.Context, req service_contract.RemoveRoleFromUserRequest) error {
	if err := s.rbacRepo.RemoveRoleFromUser(ctx, req.User, req.Role, req.CompanyID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("role removed from user",
		logger.String("user", req.User),
		logger.String("role", req.Role),
		logger.String("company_id", req.CompanyID),
	)
	return nil
}

func (s *rbacService) HasRoleForUser(ctx context.Context, user string, role string, companyID string) (bool, error) {
	has, err := s.rbacRepo.HasRoleForUser(ctx, user, role, companyID)
	if err != nil {
		return false, exception.Wrap(exception.ErrInternal, err)
	}
	return has, nil
}

func (s *rbacService) GetRolesForUser(ctx context.Context, user string, companyID string) ([]string, error) {
	roles, err := s.rbacRepo.GetRolesForUser(ctx, user, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return roles, nil
}

func (s *rbacService) GetUsersForRole(ctx context.Context, role string, companyID string) ([]string, error) {
	users, err := s.rbacRepo.GetUsersForRole(ctx, role, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return users, nil
}

func (s *rbacService) ReplaceRolesForUser(ctx context.Context, req service_contract.ReplaceRolesForUserRequest) error {
	if err := s.rbacRepo.ReplaceRolesForUser(ctx, req.User, req.CompanyID, req.Roles); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("roles replaced for user",
		logger.String("user", req.User),
		logger.String("company_id", req.CompanyID),
		logger.String("roles", strings.Join(req.Roles, ",")),
	)
	return nil
}

func (s *rbacService) DeleteRolesForUser(ctx context.Context, user string, companyID string) error {
	if err := s.rbacRepo.DeleteRolesForUser(ctx, user, companyID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("all roles removed for user in company",
		logger.String("user", user),
		logger.String("company_id", companyID),
	)
	return nil
}

func (s *rbacService) DeleteUser(ctx context.Context, user string) error {
	if err := s.rbacRepo.DeleteUser(ctx, user); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("user purged from rbac", logger.String("user", user))
	return nil
}

func (s *rbacService) DeleteRole(ctx context.Context, role string, companyID string) error {
	if err := s.rbacRepo.DeleteRole(ctx, role, companyID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("role deleted",
		logger.String("role", role),
		logger.String("company_id", companyID),
	)
	return nil
}

func (s *rbacService) GetAllRolesByCompany(ctx context.Context, companyID string) ([]string, error) {
	roles, err := s.rbacRepo.GetAllRolesByCompany(ctx, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return roles, nil
}

func (s *rbacService) GetAllUsersByCompany(ctx context.Context, companyID string) ([]string, error) {
	users, err := s.rbacRepo.GetAllUsersByCompany(ctx, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return users, nil
}

// ==========================================
// Company Management
// ==========================================

func (s *rbacService) GetAllCompanies(ctx context.Context) ([]string, error) {
	companies, err := s.rbacRepo.GetAllCompanies(ctx)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return companies, nil
}

func (s *rbacService) DeleteCompany(ctx context.Context, companyID string) error {
	if err := s.rbacRepo.DeleteCompany(ctx, companyID); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("company rbac data deleted", logger.String("company_id", companyID))
	return nil
}

// ==========================================
// Permission Management
// ==========================================

func (s *rbacService) AddPermission(ctx context.Context, req service_contract.AddPermissionRequest) error {
	permEntity := entity.Permission{
		Endpoint: req.Endpoint,
		Method:   req.Method,
	}
	if err := s.rbacRepo.AddPermission(ctx, req.Role, req.CompanyID, permEntity); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("permission added to role",
		logger.String("role", req.Role),
		logger.String("company_id", req.CompanyID),
		logger.String("endpoint", req.Endpoint),
		logger.String("method", req.Method),
	)
	return nil
}

func (s *rbacService) RemovePermission(ctx context.Context, req service_contract.RemovePermissionRequest) error {
	permEntity := entity.Permission{
		Endpoint: req.Endpoint,
		Method:   req.Method,
	}
	if err := s.rbacRepo.RemovePermission(ctx, req.Role, req.CompanyID, permEntity); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("permission removed from role",
		logger.String("role", req.Role),
		logger.String("company_id", req.CompanyID),
		logger.String("endpoint", req.Endpoint),
		logger.String("method", req.Method),
	)
	return nil
}

func (s *rbacService) GetPermissionsForRole(ctx context.Context, role string, companyID string) ([]service_contract.PermissionResponse, error) {
	perms, err := s.rbacRepo.GetPermissionsForRole(ctx, role, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return service_contract.ToPermissionResponses(perms), nil
}

func (s *rbacService) GetEffectivePermissionsForUser(ctx context.Context, user string, companyID string) ([]service_contract.PermissionResponse, error) {
	perms, err := s.rbacRepo.GetEffectivePermissionsForUser(ctx, user, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return service_contract.ToPermissionResponses(perms), nil
}

func (s *rbacService) GetAllRolePermissions(ctx context.Context, companyID string) (map[string][]service_contract.PermissionResponse, error) {
	rolePermsMap, err := s.rbacRepo.GetAllRolePermissions(ctx, companyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return service_contract.ToRolePermissionsMapResponse(rolePermsMap), nil
}

func (s *rbacService) GetAllRolePermissionsGroupedByCompany(ctx context.Context) (map[string]map[string][]service_contract.PermissionResponse, error) {
	companyMap, err := s.rbacRepo.GetAllRolePermissionsGroupedByCompany(ctx)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return service_contract.ToCompanyRolePermissionsMapResponse(companyMap), nil
}

// ==========================================
// Enforcement & Persistence
// ==========================================

func (s *rbacService) Enforce(ctx context.Context, req service_contract.EnforceRequest) (*service_contract.EnforceResponse, error) {
	allowed, err := s.rbacRepo.Enforce(req.Subject, req.CompanyID, req.Endpoint, req.Method)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return &service_contract.EnforceResponse{Allowed: allowed}, nil
}

func (s *rbacService) SavePolicy(ctx context.Context) error {
	if err := s.rbacRepo.SavePolicy(ctx); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	s.log.Info("rbac policy saved successfully")
	return nil
}
