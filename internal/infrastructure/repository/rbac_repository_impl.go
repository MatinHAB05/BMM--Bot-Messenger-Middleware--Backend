package repository

import (
	"context"
	"fmt"

	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/rbac"
)

type rbacRepository struct {
	enforcer rbac.RBACEnforcer
}

func NewRBACRepository(enforcer rbac.RBACEnforcer) repository_contract.RBACRepository {
	return &rbacRepository{
		enforcer: enforcer,
	}
}

// ==========================================
// Role Management
// ==========================================

func (r *rbacRepository) AddRoleForUser(ctx context.Context, userID string, role string, companyID string) error {
	_, err := r.enforcer.GetEnforcer().AddRoleForUser(userID, role, companyID)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) AddRolesForUser(ctx context.Context, userID string, role []string, companyID string) error {
	_, err := r.enforcer.GetEnforcer().AddRolesForUser(userID, role, companyID)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) RemoveRoleFromUser(ctx context.Context, userID string, role string, companyID string) error {
	_, err := r.enforcer.GetEnforcer().DeleteRoleForUser(userID, role, companyID)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) HasRoleForUser(ctx context.Context, userID string, role string, companyID string) (bool, error) {
	has, err := r.enforcer.GetEnforcer().HasRoleForUser(userID, role, companyID)
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}
	return has, nil
}

func (r *rbacRepository) GetRolesForUser(ctx context.Context, userID string, companyID string) ([]string, error) {
	roles, err := r.enforcer.GetEnforcer().GetRolesForUser(userID, companyID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}
	return roles, nil
}

func (r *rbacRepository) GetUsersForRole(ctx context.Context, role string, companyID string) ([]string, error) {
	users, err := r.enforcer.GetEnforcer().GetUsersForRole(role, companyID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}
	return users, nil
}

func (r *rbacRepository) ReplaceRolesForUser(ctx context.Context, userID string, companyID string, roles []string) error {
	// Wholesale replace: drop everything the user currently holds in this
	// company, then (re)apply the requested set as one grouping-policy
	// batch, so callers never have to diff old vs. new roles themselves.
	if _, err := r.enforcer.GetEnforcer().DeleteRolesForUser(userID, companyID); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}

	if len(roles) == 0 {
		return nil
	}

	if _, err := r.enforcer.GetEnforcer().AddRolesForUser(userID, roles, companyID); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) DeleteRolesForUser(ctx context.Context, userID string, companyID string) error {
	_, err := r.enforcer.GetEnforcer().DeleteRolesForUser(userID, companyID)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) DeleteUser(ctx context.Context, userID string) error {
	// Global: removes userID from every g row (every company) and any
	// permission ever granted directly to userID.
	_, err := r.enforcer.GetEnforcer().DeleteUser(userID)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) DeleteRole(ctx context.Context, role string, companyID string) error {
	if role == "" || companyID == "" {
		return fmt.Errorf("%w: role and companyID cannot be empty", exception.ErrBadRequest)
	}

	// The raw enforcer's DeleteRole(role) has no company argument and would
	// wipe the role across every company, so a company-scoped delete has to
	// be composed by hand:
	//   1) detach the role from every user holding it in this company
	//   2) drop every permission (p rule) attached to role+companyID
	users, err := r.enforcer.GetEnforcer().GetUsersForRole(role, companyID)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	for _, user := range users {
		if _, err := r.enforcer.GetEnforcer().DeleteRoleForUser(user, role, companyID); err != nil {
			return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
		}
	}

	if _, err := r.enforcer.GetEnforcer().RemoveFilteredPolicy(0, role, companyID); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}

	return nil
}

func (r *rbacRepository) GetAllRolesByCompany(ctx context.Context, companyID string) ([]string, error) {
	// g rows are (user, role, companyID); companyID sits at index 2.
	rawGrouping, err := r.enforcer.GetEnforcer().GetFilteredGroupingPolicy(2, companyID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	seen := make(map[string]struct{})
	roles := make([]string, 0, len(rawGrouping))
	for _, g := range rawGrouping {
		if len(g) >= 2 {
			if _, ok := seen[g[1]]; !ok {
				seen[g[1]] = struct{}{}
				roles = append(roles, g[1])
			}
		}
	}
	return roles, nil
}

func (r *rbacRepository) GetAllUsersByCompany(ctx context.Context, companyID string) ([]string, error) {
	rawGrouping, err := r.enforcer.GetEnforcer().GetFilteredGroupingPolicy(2, companyID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	seen := make(map[string]struct{})
	users := make([]string, 0, len(rawGrouping))
	for _, g := range rawGrouping {
		if len(g) >= 1 {
			if _, ok := seen[g[0]]; !ok {
				seen[g[0]] = struct{}{}
				users = append(users, g[0])
			}
		}
	}
	return users, nil
}

// ==========================================
// Company Management
// ==========================================

func (r *rbacRepository) GetAllCompanies(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	companies := make([]string, 0)

	// A company can show up either because it has role assignments (g)...
	grouping, err := r.enforcer.GetEnforcer().GetGroupingPolicy()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}
	for _, g := range grouping {
		if len(g) >= 3 {
			if _, ok := seen[g[2]]; !ok {
				seen[g[2]] = struct{}{}
				companies = append(companies, g[2])
			}
		}
	}

	// ...or because it has permissions defined (p), even with no users yet.
	policies, err := r.enforcer.GetEnforcer().GetPolicy()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}
	for _, p := range policies {
		if len(p) >= 2 {
			if _, ok := seen[p[1]]; !ok {
				seen[p[1]] = struct{}{}
				companies = append(companies, p[1])
			}
		}
	}

	return companies, nil
}

func (r *rbacRepository) DeleteCompany(ctx context.Context, companyID string) error {
	if companyID == "" {
		return fmt.Errorf("%w: companyID cannot be empty", exception.ErrBadRequest)
	}

	// g rows: (user, role, companyID) -> companyID at index 2.
	if _, err := r.enforcer.GetEnforcer().RemoveFilteredGroupingPolicy(2, companyID); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}

	// p rows: (role, companyID, obj, act) -> companyID at index 1.
	if _, err := r.enforcer.GetEnforcer().RemoveFilteredPolicy(1, companyID); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}

	return nil
}

// ==========================================
// Permission Management (Mapped)
// ==========================================

func (r *rbacRepository) AddPermission(ctx context.Context, role string, companyID string, perm entity.Permission) error {
	// p = sub (role), dom (companyID), obj (endpoint), act (method)
	_, err := r.enforcer.GetEnforcer().AddPolicy(role, companyID, perm.Endpoint, perm.Method)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) RemovePermission(ctx context.Context, role string, companyID string, perm entity.Permission) error {
	_, err := r.enforcer.GetEnforcer().RemovePolicy(role, companyID, perm.Endpoint, perm.Method)
	if err != nil {
		return fmt.Errorf("%w: %v", exception.ErrRoleSyncFailed, err)
	}
	return nil
}

func (r *rbacRepository) GetPermissionsForRole(ctx context.Context, role string, companyID string) ([]entity.Permission, error) {
	if role == "" || companyID == "" {
		return nil, fmt.Errorf("%w: role and companyID cannot be empty", exception.ErrBadRequest)
	}

	// Casbin: [["admin", "company-1", "/api/v1/users", "GET"], ...]
	rawPolicies, err := r.enforcer.GetEnforcer().GetFilteredPolicy(0, role, companyID)
	if err != nil {
		return nil, exception.ErrRoleSyncFailed
	}

	permissions := make([]entity.Permission, 0, len(rawPolicies))
	for _, policy := range rawPolicies {
		if len(policy) >= 4 {
			permissions = append(permissions, entity.Permission{
				Endpoint: policy[2], // obj
				Method:   policy[3], // act
			})
		}
	}

	return permissions, nil
}

func (r *rbacRepository) GetEffectivePermissionsForUser(ctx context.Context, userID string, companyID string) ([]entity.Permission, error) {
	if userID == "" || companyID == "" {
		return nil, fmt.Errorf("%w: userID and companyID cannot be empty", exception.ErrBadRequest)
	}

	roles, err := r.enforcer.GetEnforcer().GetRolesForUser(userID, companyID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}

	// Check the user's own subject too: Casbin's role manager treats a
	// subject as implicitly linked to itself, so a p rule written directly
	// against userID (bypassing roles) is honored as well.
	subjects := append([]string{userID}, roles...)

	seen := make(map[entity.Permission]struct{})
	permissions := make([]entity.Permission, 0)

	for _, subject := range subjects {
		rawPolicies, err := r.enforcer.GetEnforcer().GetFilteredPolicy(0, subject, companyID)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
		}
		for _, policy := range rawPolicies {
			if len(policy) >= 4 {
				perm := entity.Permission{Endpoint: policy[2], Method: policy[3]}
				if _, ok := seen[perm]; !ok {
					seen[perm] = struct{}{}
					permissions = append(permissions, perm)
				}
			}
		}
	}

	return permissions, nil
}

func (r *rbacRepository) GetAllRolePermissions(ctx context.Context, companyID string) (map[string][]entity.Permission, error) {
	// fieldIndex 1 => match on companyID regardless of role.
	rawPolicies, err := r.enforcer.GetEnforcer().GetFilteredPolicy(1, companyID)
	if err != nil {
		return nil, exception.ErrRoleSyncFailed
	}

	roleMap := make(map[string][]entity.Permission)

	for _, policy := range rawPolicies {
		if len(policy) >= 4 {
			role := policy[0]
			perm := entity.Permission{
				Endpoint: policy[2],
				Method:   policy[3],
			}
			roleMap[role] = append(roleMap[role], perm)
		}
	}

	return roleMap, nil
}

func (r *rbacRepository) GetAllRolePermissionsGroupedByCompany(ctx context.Context) (map[string]map[string][]entity.Permission, error) {
	rawPolicies, err := r.enforcer.GetEnforcer().GetPolicy()
	if err != nil {
		return nil, exception.ErrRoleSyncFailed
	}

	companyMap := make(map[string]map[string][]entity.Permission)

	for _, policy := range rawPolicies {
		if len(policy) >= 4 {
			role, companyID := policy[0], policy[1]
			perm := entity.Permission{
				Endpoint: policy[2],
				Method:   policy[3],
			}
			if companyMap[companyID] == nil {
				companyMap[companyID] = make(map[string][]entity.Permission)
			}
			companyMap[companyID][role] = append(companyMap[companyID][role], perm)
		}
	}

	return companyMap, nil
}

// ==========================================
// Enforcement & Persistence
// ==========================================

func (r *rbacRepository) Enforce(sub string, dom string, obj string, act string) (bool, error) {
	allowed, err := r.enforcer.GetEnforcer().Enforce(sub, dom, obj, act)
	if err != nil {
		return false, fmt.Errorf("%w: %v", exception.ErrInternal, err)
	}
	return allowed, nil
}

func (r *rbacRepository) SavePolicy(ctx context.Context) error {
	if err := r.enforcer.GetEnforcer().SavePolicy(); err != nil {
		return fmt.Errorf("%w: %v", exception.ErrDatabaseOperation, err)
	}
	return nil
}
