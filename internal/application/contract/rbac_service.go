// internal/application/contract/rbac_service.go
package service_contract

import (
	"context"
	"messenger-backend/internal/domain/entity"
)

// Request & Response DTOs

type AddRoleForUserRequest struct {
	User      string `json:"user"`
	Role      string `json:"role"`
	CompanyID string `json:"company_id"`
}

type RemoveRoleFromUserRequest struct {
	User      string `json:"user"`
	Role      string `json:"role"`
	CompanyID string `json:"company_id"`
}

type ReplaceRolesForUserRequest struct {
	User      string   `json:"user"`
	CompanyID string   `json:"company_id"`
	Roles     []string `json:"roles" binding:"required,min=1"`
}

type AddPermissionRequest struct {
	Role      string `json:"role"`
	CompanyID string `json:"company_id"`
	Endpoint  string `json:"endpoint"`
	Method    string `json:"method"`
}

type RemovePermissionRequest struct {
	Role      string `json:"role"`
	CompanyID string `json:"company_id"`
	Endpoint  string `json:"endpoint"`
	Method    string `json:"method"`
}

type PermissionResponse struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
}

type EnforceRequest struct {
	Subject   string `json:"subject"`
	CompanyID string `json:"company_id"`
	Endpoint  string `json:"endpoint"`
	Method    string `json:"method"`
}

type EnforceResponse struct {
	Allowed bool `json:"allowed"`
}

// Service Interface

type RBACService interface {
	// Role Management
	AddRoleForUser(ctx context.Context, req AddRoleForUserRequest) error
	RemoveRoleFromUser(ctx context.Context, req RemoveRoleFromUserRequest) error
	HasRoleForUser(ctx context.Context, user string, role string, companyID string) (bool, error)
	GetRolesForUser(ctx context.Context, user string, companyID string) ([]string, error)
	GetUsersForRole(ctx context.Context, role string, companyID string) ([]string, error)
	ReplaceRolesForUser(ctx context.Context, req ReplaceRolesForUserRequest) error
	DeleteRolesForUser(ctx context.Context, user string, companyID string) error
	DeleteUser(ctx context.Context, user string) error
	DeleteRole(ctx context.Context, role string, companyID string) error
	GetAllRolesByCompany(ctx context.Context, companyID string) ([]string, error)
	GetAllUsersByCompany(ctx context.Context, companyID string) ([]string, error)

	// Company Management
	GetAllCompanies(ctx context.Context) ([]string, error)
	DeleteCompany(ctx context.Context, companyID string) error

	// Permission Management
	AddPermission(ctx context.Context, req AddPermissionRequest) error
	RemovePermission(ctx context.Context, req RemovePermissionRequest) error
	GetPermissionsForRole(ctx context.Context, role string, companyID string) ([]PermissionResponse, error)
	GetEffectivePermissionsForUser(ctx context.Context, user string, companyID string) ([]PermissionResponse, error)
	GetAllRolePermissions(ctx context.Context, companyID string) (map[string][]PermissionResponse, error)
	GetAllRolePermissionsGroupedByCompany(ctx context.Context) (map[string]map[string][]PermissionResponse, error)

	// Enforcement & Persistence
	Enforce(ctx context.Context, req EnforceRequest) (*EnforceResponse, error)
	SavePolicy(ctx context.Context) error
}

// Mappers

func ToPermissionResponse(perm entity.Permission) PermissionResponse {
	return PermissionResponse{
		Endpoint: perm.Endpoint,
		Method:   perm.Method,
	}
}

func ToPermissionResponses(permissions []entity.Permission) []PermissionResponse {
	responses := make([]PermissionResponse, len(permissions))
	for i, perm := range permissions {
		responses[i] = ToPermissionResponse(perm)
	}
	return responses
}

func ToRolePermissionsMapResponse(rolePerms map[string][]entity.Permission) map[string][]PermissionResponse {
	result := make(map[string][]PermissionResponse, len(rolePerms))
	for role, perms := range rolePerms {
		result[role] = ToPermissionResponses(perms)
	}
	return result
}

func ToCompanyRolePermissionsMapResponse(companyRolePerms map[string]map[string][]entity.Permission) map[string]map[string][]PermissionResponse {
	result := make(map[string]map[string][]PermissionResponse, len(companyRolePerms))
	for companyID, rolePerms := range companyRolePerms {
		result[companyID] = ToRolePermissionsMapResponse(rolePerms)
	}
	return result
}
