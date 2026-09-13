package repository_contract

import (
	"context"

	"messenger-backend/internal/domain/entity"
)

// RBACRepository defines persistence and enforcement operations for a
// multi-tenant, domain-aware RBAC model:
//
//	sub (r/p) => userID
//	dom (r/p) => companyID   (a user can hold different roles per company)
//	obj (r/p) => endpoint
//	act (r/p) => http method
//
// Roles, role assignments and permissions are always scoped to a single
// companyID. The same role name in two different companies is treated as
// two completely independent roles.
type RBACRepository interface {
	// ==========================================
	// Role Management (scoped to a company)
	// ==========================================

	AddRoleForUser(ctx context.Context, userID string, role string, companyID string) error
	AddRolesForUser(ctx context.Context, userID string, role []string, companyID string) error
	RemoveRoleFromUser(ctx context.Context, userID string, role string, companyID string) error
	HasRoleForUser(ctx context.Context, userID string, role string, companyID string) (bool, error)
	GetRolesForUser(ctx context.Context, userID string, companyID string) ([]string, error)
	GetUsersForRole(ctx context.Context, role string, companyID string) ([]string, error)

	// ReplaceRolesForUser atomically overwrites every role userID holds in
	// companyID with roles.
	ReplaceRolesForUser(ctx context.Context, userID string, companyID string, roles []string) error

	// DeleteRolesForUser strips every role userID holds in companyID only
	// (e.g. user removed from that company). Roles in other companies are
	// untouched.
	DeleteRolesForUser(ctx context.Context, userID string, companyID string) error

	// DeleteUser purges userID from every role in every company (e.g. the
	// user account itself was deleted).
	DeleteUser(ctx context.Context, userID string) error

	// DeleteRole removes role from companyID entirely: every user holding it
	// there loses it, and every permission attached to role+companyID is
	// removed. The same role name in other companies is untouched.
	DeleteRole(ctx context.Context, role string, companyID string) error

	GetAllRolesByCompany(ctx context.Context, companyID string) ([]string, error)
	GetAllUsersByCompany(ctx context.Context, companyID string) ([]string, error)

	// ==========================================
	// Company Management
	// ==========================================

	GetAllCompanies(ctx context.Context) ([]string, error)

	// DeleteCompany wipes every role assignment AND every permission that
	// belongs to companyID (e.g. a company was deleted/deactivated).
	DeleteCompany(ctx context.Context, companyID string) error

	// ==========================================
	// Permission Management (Mapped, attached to a role within a company)
	// ==========================================

	AddPermission(ctx context.Context, role string, companyID string, permission entity.Permission) error
	RemovePermission(ctx context.Context, role string, companyID string, permission entity.Permission) error

	GetPermissionsForRole(ctx context.Context, role string, companyID string) ([]entity.Permission, error)

	// GetEffectivePermissionsForUser resolves every permission userID has in
	// companyID, combining permissions inherited from all of the user's
	// roles there with any permission granted directly to the userID.
	GetEffectivePermissionsForUser(ctx context.Context, userID string, companyID string) ([]entity.Permission, error)

	GetAllRolePermissions(ctx context.Context, companyID string) (map[string][]entity.Permission, error)

	// GetAllRolePermissionsGroupedByCompany returns companyID -> role ->
	// permissions for every company, system-wide (e.g. a super-admin view).
	GetAllRolePermissionsGroupedByCompany(ctx context.Context) (map[string]map[string][]entity.Permission, error)

	// ==========================================
	// Enforcement & Persistence
	// ==========================================

	Enforce(sub string, dom string, obj string, act string) (bool, error)
	SavePolicy(ctx context.Context) error
}
