// Package seed provides the initial database seeder: a default company,
// its super admin and simple admin users (if they don't already exist),
// and the baseline RBAC policy set. Run is idempotent and safe to call on
// every startup.
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/hasher"
	"messenger-backend/pkg/logger"
)

// Run seeds the default company, its super admin and simple admin users,
// and the baseline RBAC policy set, returning the (possibly pre-existing)
// default company's ID. bootstrap/init.go uses that ID as the bot ingestion
// company when BOT_COMPANY_ID isn't explicitly configured -- see its doc
// comment.
//
// Config must provide two extra fields:
//
//	SimpleAdminUsername string // defaults to "admin" when empty
//	SimpleAdminPassword string // defaults to the super admin password when empty
func Run(
	ctx context.Context,
	companyRepo repository_contract.CompanyRepository,
	userRepo repository_contract.UserRepository,
	rbacRepo repository_contract.RBACRepository,
	chatRepo repository_contract.ChatRepository,
	hasher hasher.Hasher,
	cfg Config,
	log logger.Logger,
) (uint, error) {
	var chats []AdminChatConfig
	if err := json.Unmarshal([]byte(cfg.SuperAdminChatsRawPayload), &chats); err != nil {
		return 0, err
	}
	log.Info("admin chat payload", logger.Any("data", chats))

	company, err := companyRepo.FindByCode(ctx, cfg.CompanyCode)
	if err != nil && !errors.Is(err, exception.ErrCompanyNotFound) {
		return 0, fmt.Errorf("lookup default company: %w", err)
	}

	if company != nil {
		log.Info("seed: default company already exists, skipping creation", logger.String("code", cfg.CompanyCode))
	} else {
		company = &entity.Company{Name: cfg.CompanyName, Code: cfg.CompanyCode, IsActive: true, Description: "My-init-description-company-for-test :)"}
		if createErr := companyRepo.Create(ctx, company); createErr != nil {
			return 0, fmt.Errorf("create default company: %w", createErr)
		}
		log.Info("seed: default company created", logger.String("code", company.Code), logger.Uint("company_id", company.ID))
	}

	// Super admin
	superAdmin, err := ensureUser(ctx, userRepo, hasher, log, &entity.User{
		Firstname:       "Matin",
		Lastname:        "HAB",
		CompanyID:       company.ID,
		Username:        cfg.SuperAdminUsername,
		Email:           &cfg.SuperAdminEmail,
		Phone:           &cfg.SuperAdminPhone,
		IsActive:        true,
		IsVerifiedPhone: true,
		IsVerifiedEmail: true,
	}, cfg.SuperAdminPassword)
	if err != nil {
		return 0, err
	}

	// Simple admin (email/phone derived from the super admin's)
	// simpleUsername := cfg.SimpleAdminUsername //TODO :
	simpleUsername := strings.Replace(cfg.SuperAdminUsername, "super-admin", "admin", 1)
	if simpleUsername == "" {
		simpleUsername = "admin"
	}

	// simplePassword := cfg.SimpleAdminPassword //TODO :
	simplePassword := cfg.SuperAdminPassword
	if simplePassword == "" {
		simplePassword = cfg.SuperAdminPassword
	}
	// simpleEmail :=  //TODO :
	simpleEmail := strings.Replace(cfg.SuperAdminEmail, "superadmin", "admin", 1)

	// simplePhone :=  //TODO :
	simplePhone := strings.Replace(cfg.SuperAdminPhone, "0916", "0918", 1)

	simpleAdmin, err := ensureUser(ctx, userRepo, hasher, log, &entity.User{
		Firstname:       "Simple",
		Lastname:        "Admin",
		CompanyID:       company.ID,
		Username:        simpleUsername,
		Email:           &simpleEmail,
		Phone:           &simplePhone,
		IsActive:        true,
		IsVerifiedPhone: true,
		IsVerifiedEmail: true,
	}, simplePassword)
	if err != nil {
		return 0, err
	}

	if err := seedPolicies(ctx, company.ID, superAdmin.ID, simpleAdmin.ID, rbacRepo, log); err != nil {
		return 0, fmt.Errorf("seed rbac policies: %w", err)
	}

	// Seed admin chats
	for _, chat := range chats {

		err := chatRepo.Create(ctx, &entity.Chat{
			CompanyID:      &company.ID,
			Platform:       entity.MessengerPlatform(chat.Platform),
			PlatformChatID: chat.PlatformChatID,
			Title:          chat.Title,
			Username:       chat.Username,
			ChatType:       entity.ChatType(chat.ChatType),
			IsPrivate:      chat.IsPrivate,
			IsActive:       true,
		})
		if err != nil {
			log.Warn("seed: failed to create chat or already exists",
				logger.String("platform", chat.Platform),
				logger.String("platform_chat_id", chat.PlatformChatID),
				logger.Err(err),
			)
			continue
		}
		log.Info("seed: admin chat created successfully",
			logger.String("platform", chat.Platform),
			logger.String("platform_chat_id", chat.PlatformChatID),
		)
	}

	return company.ID, nil
}

// ensureUser returns the existing user (looked up by username within the
// company) or creates it with the given raw password.
func ensureUser(
	ctx context.Context,
	userRepo repository_contract.UserRepository,
	h hasher.Hasher,
	log logger.Logger,
	u *entity.User,
	rawPassword string,
) (*entity.User, error) {
	existing, err := userRepo.FindByUsernameInCompany(ctx, u.CompanyID, u.Username)
	if err != nil && !exception.ErrUserNotFound.Is(err) {
		return nil, fmt.Errorf("lookup user %s: %w", u.Username, err)
	}
	if existing != nil {
		log.Info("seed: user already exists, skipping creation", logger.String("username", u.Username))
		return existing, nil
	}

	hash, err := h.Hash(rawPassword)
	if err != nil {
		return nil, fmt.Errorf("hash password for %s: %w", u.Username, err)
	}
	u.PasswordHash = string(hash)

	if err := userRepo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("create user %s: %w", u.Username, err)
	}
	log.Info("seed: user created", logger.String("username", u.Username))
	return u, nil
}

// deriveEmail uses plus-addressing: matin@x.com -> matin+admin@x.com
// (mail still lands in the super admin's inbox, but the address is unique).
func deriveEmail(email, tag string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return email
	}
	return email[:at] + "+" + tag + email[at:]
}

// derivePhone increments the last digit (with carry), keeping the format:
// 09121234567 -> 09121234568, +989121234569 -> +989121234570
func derivePhone(phone string) string {
	b := []byte(phone)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] < '0' || b[i] > '9' {
			continue
		}
		if b[i] == '9' {
			b[i] = '0'
			continue
		}
		b[i]++
		return string(b)
	}
	return phone
}

// seedPolicies is intentionally explicit (a literal policy table) rather
// than generated, so the default authorization surface is easy to audit
// at a glance and to extend as new endpoints are added.
//
// Role -> permission policies use a wildcard domain ("*") so "admin" and
// "user" mean the same thing in every company; only role ASSIGNMENT
// (via AddRoleForUser below) is company-specific -- that's what
// makes admin privileges bounded to one company with no global superuser.
func seedPolicies(ctx context.Context, companyID, superAdminID, adminID uint, rbacRepo repository_contract.RBACRepository, log logger.Logger) error {
	superAdminSubject := fmt.Sprint(superAdminID)
	adminSubject := fmt.Sprint(adminID)
	companyDomain := fmt.Sprint(companyID)

	for _, p := range DefaultAdminPolicies {
		if err := rbacRepo.AddPermission(ctx, p.Role, p.Domain, p.Permission); err != nil {
			return fmt.Errorf("add permission %v for role %s: %w", p.Permission, p.Role, err)
		}
		log.Info("seed: policy added", logger.String("role", p.Role), logger.String("object", p.Permission.Endpoint), logger.String("action", p.Permission.Method))
	}

	if err := rbacRepo.AddRoleForUser(ctx, superAdminSubject, entity.RoleSuperAdmin, companyDomain); err != nil {
		return fmt.Errorf("assign super admin role: %w", err)
	}
	if err := rbacRepo.AddRoleForUser(ctx, adminSubject, entity.RoleAdmin, companyDomain); err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}

	log.Info("seed: roles assigned",
		logger.String("super_admin_id", superAdminSubject),
		logger.String("admin_id", adminSubject),
		logger.String("company_id", companyDomain),
	)

	return rbacRepo.SavePolicy(ctx)
}
