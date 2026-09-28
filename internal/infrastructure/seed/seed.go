// Package seed provides the initial database seeder: a default company,
// its admin user (if one doesn't already exist), and the baseline RBAC
// policy set. Run is idempotent and safe to call on every startup.
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/pkg/hasher"
	"messenger-backend/pkg/logger"
)

// Run seeds the default company, its admin user, and the baseline RBAC
// policy set, returning the (possibly pre-existing) default company's ID.
// bootstrap/init.go uses that ID as the bot ingestion company when
// BOT_COMPANY_ID isn't explicitly configured -- see its doc comment.
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

	admin, err := userRepo.FindByUsernameInCompany(ctx, company.ID, cfg.SuperAdminUsername)
	if err != nil && !exception.ErrUserNotFound.Is(err) {
		return 0, fmt.Errorf("lookup admin user: %w", err)
	}

	if admin != nil {
		log.Info("seed: admin user already exists, skipping creation", logger.String("username", cfg.SuperAdminUsername))
	} else {
		hash, hashErr := hasher.Hash(cfg.SuperAdminPassword)
		if hashErr != nil {
			return 0, fmt.Errorf("hash admin password: %w", hashErr)
		}

		admin = &entity.User{
			Firstname:       "Matin",
			Lastname:        "HAB",
			CompanyID:       company.ID,
			Username:        cfg.SuperAdminUsername,
			Email:           &cfg.SuperAdminEmail,
			PasswordHash:    string(hash),
			IsActive:        true,
			IsVerifiedPhone: true,
			IsVerifiedEmail: true,
			Phone:           &cfg.SuperAdminPhone,
		}
		if createErr := userRepo.Create(ctx, admin); createErr != nil {
			return 0, fmt.Errorf("create admin user: %w", createErr)
		}
		log.Info("seed: admin user created", logger.String("username", admin.Username))
	}

	if err := seedPolicies(ctx, company.ID, admin.ID, rbacRepo, log); err != nil {
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

// seedPolicies is intentionally explicit (a literal policy table) rather
// than generated, so the default authorization surface is easy to audit
// at a glance and to extend as new endpoints are added.
//
// Role -> permission policies use a wildcard domain ("*") so "admin" and
// "user" mean the same thing in every company; only role ASSIGNMENT
// (via AddRoleForUserInDomain below) is company-specific -- that's what
// makes admin privileges bounded to one company with no global superuser.
func seedPolicies(ctx context.Context, companyID, adminID uint, rbacRepo repository_contract.RBACRepository, log logger.Logger) error {
	adminSubject := fmt.Sprint(adminID)
	companyDomain := fmt.Sprint(companyID)

	for _, p := range DefaultAdminPolicies {
		if err := rbacRepo.AddPermission(ctx, p.Role, p.Domain, p.Permission); err != nil {
			return fmt.Errorf("add permission %v for role %s: %w", p.Permission, p.Role, err)
		}
		log.Info("seed: policy added", logger.String("role", p.Role), logger.String("object", p.Permission.Endpoint), logger.String("action", p.Permission.Method))
	}

	if err := rbacRepo.AddRoleForUser(ctx, adminSubject, entity.RoleSuperAdmin, companyDomain); err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}
	log.Info("seed: admin role assigned", logger.String("user_id", adminSubject), logger.String("company_id", companyDomain))

	return rbacRepo.SavePolicy(ctx)
}
