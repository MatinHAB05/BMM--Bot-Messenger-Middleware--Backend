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
	"strconv"
)

//todo : use hasher interface

// userService implements service_contract.UserService. Every method is scoped to
// a companyID -- there is no path to read or mutate a user outside the
// caller's own company. Role assignment is never stored on the User row;
// it is always read from and written to the Casbin enforcer's
// domain-scoped RBAC API (domain == company id as a string), whose GORM
// adapter is the single source of truth for authorization policy.
type userService struct {
	userRepo   repository_contract.UserRepository
	otpService service_contract.OTPService
	enforcer   repository_contract.RBACRepository
	log        logger.Logger
}

func NewUserService(
	userRepo repository_contract.UserRepository,
	otpService service_contract.OTPService,
	enforcer repository_contract.RBACRepository,
	log logger.Logger,
) service_contract.UserService {
	return &userService{
		userRepo:   userRepo,
		otpService: otpService,
		enforcer:   enforcer,
		log:        log.With(logger.String("component", "user_service")),
	}
}

func (s *userService) Me(ctx context.Context, companyID uint, userID string) (*service_contract.UserResponse, error) {
	id, err := parseUint(userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrUserNotFound, err)
	}

	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, id)
	if err != nil || user == nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	roles, err := s.rolesFor(ctx, companyID, userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToUserResponse(user, roles)
	return &resp, nil
}

func (s *userService) GetByID(ctx context.Context, companyID uint, userID string) (*service_contract.UserResponse, error) {
	id, err := parseUint(userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrUserNotFound, err)
	}
	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, id)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	roles, err := s.rolesFor(ctx, companyID, userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	resp := service_contract.ToUserResponse(user, roles)
	return &resp, nil
}

func (s *userService) List(ctx context.Context, companyID uint, page, pageSize int) (*service_contract.UserListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	users, total, err := s.userRepo.List(ctx, companyID, offset, pageSize)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	responses := make([]service_contract.UserResponse, 0, len(users))
	for i := range users {
		roles, err := s.rolesFor(ctx, companyID, strconv.FormatUint(uint64(users[i].ID), 10))
		if err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
		responses = append(responses, service_contract.ToUserResponse(&users[i], roles))
	}

	return &service_contract.UserListResponse{
		Users:    responses,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

/* Create ?!??!
func (s *userService) Create(ctx context.Context, companyID uint, req service_contract.CreateUserRequest) (*service_contract.UserResponse, error) {
	exists, err := s.userRepo.ExistsByUsername(ctx, companyID, req.Username)
	if err != nil || exists == nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	if *exists {
		return nil, exception.ErrUserAlreadyExists
	}

	var passwordHash string

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
		passwordHash = string(hash)
	}
	// An empty passwordHash is valid: it means this user was invited
	// passwordless and will activate via POST /auth/otp/send + /verify
	// against their Email or Phone instead of POST /auth/login.

	user := &entity.User{
		CompanyID:    companyID,
		Username:     req.Username,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: passwordHash,
		IsActive:     true,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.syncRoles(ctx, companyID, fmt.Sprint(user.ID), req.Roles); err != nil {
		return nil, exception.Wrap(exception.ErrRoleSyncFailed, err)
	}

	s.log.Info("user created", logger.String("username", user.Username), logger.Uint("user_id", user.ID), logger.Any("roles", req.Roles))

	resp := service_contract.ToUserResponse(user, req.Roles)
	return &resp, nil
}
*/

func (s *userService) Update(ctx context.Context, companyID uint, userID string, req service_contract.UpdateUserRequest) (*service_contract.UserResponse, error) {
	id, err := parseUint(userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrUserNotFound, err)
	}
	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, id)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("user updated", logger.String("user_id", userID))

	roles, err := s.rolesFor(ctx, companyID, userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := service_contract.ToUserResponse(user, roles)
	return &resp, nil
}

func (s *userService) UpdateEmail(ctx context.Context, companyID uint, userID uint, req service_contract.UpdateUserEmailRequest) (*service_contract.UserResponse, error) {
	// id, err := parseUint(userID)
	// if err != nil {
	// 	return nil, exception.Wrap(exception.ErrUserNotFound, err)
	// }
	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, userID)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if !user.IsActive {
		return nil, exception.ErrUserInactive
	}

	if _, err := s.checkCrdentionls(ctx, req.Email, "email", user); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("user updated", logger.Uint("user_id", userID))

	resp := service_contract.ToUserResponse(user, nil)
	return &resp, nil
}

func (s *userService) UpdatePhone(ctx context.Context, companyID uint, userID uint, req service_contract.UpdateUserPhoneRequest) (*service_contract.UserResponse, error) {
	// id, err := parseUint(userID)
	// if err != nil {
	// 	return nil, exception.Wrap(exception.ErrUserNotFound, err)
	// }
	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, userID)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if !user.IsActive {
		return nil, exception.ErrUserInactive
	}

	if _, err := s.checkCrdentionls(ctx, req.Phone, "phone", user); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("user updated", logger.Uint("user_id", userID))

	resp := service_contract.ToUserResponse(user, nil)
	return &resp, nil
}

func (s *userService) UpdateUsername(ctx context.Context, companyID uint, userID uint, req service_contract.UpdateUserUsernameRequest) (*service_contract.UserResponse, error) {
	// id, err := parseUint(userID)
	// if err != nil {
	// 	return nil, exception.Wrap(exception.ErrUserNotFound, err)
	// }
	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, userID)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if !user.IsActive {
		return nil, exception.ErrUserInactive
	}

	if _, err := s.checkCrdentionls(ctx, req.Username, "username", user); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	s.log.Info("user updated", logger.Uint("user_id", userID))

	resp := service_contract.ToUserResponse(user, nil)
	return &resp, nil
}

func (s *userService) UpdateRoles(ctx context.Context, companyID uint, userID string, roles []string) (*service_contract.UserResponse, error) {
	id, err := parseUint(userID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrUserNotFound, err)
	}
	user, err := s.userRepo.FindByIDInCompany(ctx, companyID, id)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.syncRoles(ctx, companyID, userID, roles); err != nil {
		return nil, exception.Wrap(exception.ErrRoleSyncFailed, err)
	}

	s.log.Info("user roles updated", logger.String("user_id", userID), logger.Any("roles", roles))

	resp := service_contract.ToUserResponse(user, roles)
	return &resp, nil
}

func (s *userService) Delete(ctx context.Context, companyID uint, userID string) error {
	id, err := parseUint(userID)
	if err != nil {
		return exception.Wrap(exception.ErrUserNotFound, err)
	}

	_, err = s.userRepo.FindByIDInCompany(ctx, companyID, id)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return exception.ErrUserNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.userRepo.Delete(ctx, companyID, id); err != nil {
		return exception.Wrap(exception.ErrInternal, err)
	}
	if err := s.enforcer.DeleteRolesForUser(ctx, userID, fmt.Sprint(companyID)); err != nil {
		s.log.Warn("failed to clear casbin roles for deleted user", logger.Err(err), logger.String("user_id", userID))
	}

	s.log.Info("user deleted", logger.String("user_id", userID))
	return nil
}

// rolesFor returns the Casbin roles currently assigned to subject within
// companyID's domain.
func (s *userService) rolesFor(ctx context.Context, companyID uint, subject string) ([]string, error) {
	roles, err := s.enforcer.GetRolesForUser(ctx, subject, strconv.FormatUint(uint64(companyID), 10))
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// syncRoles replaces subject's entire role set, within companyID's
// domain, with roles.
func (s *userService) syncRoles(ctx context.Context, companyID uint, subject string, roles []string) error {
	if err := s.enforcer.DeleteRolesForUser(ctx, subject, strconv.FormatUint(uint64(companyID), 10)); err != nil {
		return fmt.Errorf("clear existing roles: %w", err)
	}
	for _, role := range roles {
		if err := s.enforcer.AddRoleForUser(ctx, subject, role, strconv.FormatUint(uint64(companyID), 10)); err != nil {
			return fmt.Errorf("assign role %q: %w", role, err)
		}
	}
	return nil
}

func (s *userService) checkCrdentionls(ctx context.Context, identifier string, idType string, user *entity.User) (*bool, error) {
	// check username uniqess
	switch idType {
	case "username":
		_, err := s.userRepo.FindByIdentifier(ctx, identifier)
		if err == nil {
			return nil, exception.ErrUserUsernameAlreadyExists
		} else if !errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.Wrap(exception.ErrInternal, err)
		} else {
			user.Username = identifier
		}
	case "email":
		// check email uniqess & verification
		if identifier != "" {
			_, err := s.userRepo.FindByIdentifier(ctx, identifier)
			if err == nil {
				return nil, exception.ErrUserEmailAlreadyExists
			} else if !errors.Is(err, exception.ErrUserNotFound) {
				return nil, exception.Wrap(exception.ErrInternal, err)
			}
			is, err := s.otpService.InvalidateIsVerified(ctx, identifier, otp.TypeEmail)
			if err != nil {
				return nil, exception.Wrap(exception.ErrInternal, err)
			} else if !*is {
				return nil, exception.ErrNotVerifiedEmailCredentials
			}
			user.Email = &identifier
			user.IsVerifiedEmail = true
		} else {
			user.Email = nil
			user.IsVerifiedEmail = false
		}

	case "phone":

		// check phone uniqess & verification
		if identifier != "" {
			_, err := s.userRepo.FindByIdentifier(ctx, identifier)
			if err == nil {
				return nil, exception.ErrUserPhoneAlreadyExists
			} else if !errors.Is(err, exception.ErrUserNotFound) {
				return nil, exception.Wrap(exception.ErrInternal, err)
			}
			is, err := s.otpService.InvalidateIsVerified(ctx, identifier, otp.TypePhone)
			if err != nil {
				return nil, exception.Wrap(exception.ErrInternal, err)
			} else if !*is {
				return nil, exception.ErrNotVerifiedPhoneCredentials
			}
			user.Phone = &identifier
			user.IsVerifiedPhone = true
		} else {
			user.Phone = nil
			user.IsVerifiedPhone = false
		}

	default:
		return nil, exception.ErrInternal
	}

	ok := true
	return &ok, nil
}
