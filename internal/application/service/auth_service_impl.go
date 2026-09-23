package service

import (
	"context"
	"errors"
	"fmt"
	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/entity"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/otp"
	"messenger-backend/internal/domain/paseto"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/infrastructure/database"
	"messenger-backend/pkg/logger"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthServiceConfig struct {
	AppEnv          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	OTPTTL          time.Duration
}

// authService implements service_contract.AuthService. It owns the full token
// lifecycle: PASETO issuance, Redis-backed refresh sessions
// (refresh_token:<user_id>:<token_id>), and access-token revocation via a
// Redis blacklist keyed by the token's jti.
type authService struct {
	userRepo    repository_contract.UserRepository
	companyRepo repository_contract.CompanyRepository
	rbacRepo    repository_contract.RBACRepository
	tokenRepo   repository_contract.AuthnTokenRepository
	otpService  service_contract.OTPService
	tokenMaker  paseto.Maker
	trxManager  database.TrxManager

	log logger.Logger
	*AuthServiceConfig
}

func NewAuthService(
	userRepo repository_contract.UserRepository,
	companyRepo repository_contract.CompanyRepository,
	rbacRepo repository_contract.RBACRepository,
	tokenRepo repository_contract.AuthnTokenRepository,
	otpService service_contract.OTPService,
	trxManager database.TrxManager,
	tokenMaker paseto.Maker,
	log logger.Logger,
	cfg *AuthServiceConfig,
) service_contract.AuthService {
	return &authService{
		userRepo:          userRepo,
		companyRepo:       companyRepo,
		tokenRepo:         tokenRepo,
		tokenMaker:        tokenMaker,
		otpService:        otpService,
		rbacRepo:          rbacRepo,
		trxManager:        trxManager,
		log:               log.With(logger.String("component", "auth_service")),
		AuthServiceConfig: cfg,
	}
}

func (s *authService) Login(ctx context.Context, req service_contract.LoginRequest) (*service_contract.TokenPairResponse, error) {
	fmt.Println("login-req", req.Default, req.EmailOption, req.PhoneOption)
	if !isValidLoginRequest(req) {
		return nil, exception.ErrBadLoginRequest
	}

	switch {
	case req.Default != nil:
		return s.loginWithPassword(ctx, req.Default.Username, req.Default.Password)

	case req.EmailOption != nil:
		return s.loginWithOTP(ctx, req.EmailOption.Email, otp.TypeEmail)

	case req.PhoneOption != nil:
		return s.loginWithOTP(ctx, req.PhoneOption.Phone, otp.TypePhone)

	default:
		return nil, exception.ErrBadLoginRequest
	}
}

// چک می‌کند که فقط و دقیقاً یکی از فیلدها پر باشد
func isValidLoginRequest(req service_contract.LoginRequest) bool {
	count := 0
	if req.Default != nil {
		count++
	}
	if req.EmailOption != nil {
		count++
	}
	if req.PhoneOption != nil {
		count++
	}
	return count == 1
}

func (s *authService) loginWithPassword(ctx context.Context, username, password string) (*service_contract.TokenPairResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound // حفظ خطای دقیق جهت دیباگ
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if err := validateUserStatus(user); err != nil {
		return nil, err
	}

	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, exception.ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, user.ID, user.CompanyID, user.Username)
}

func (s *authService) loginWithOTP(ctx context.Context, identifier string, otpType otp.Type) (*service_contract.TokenPairResponse, error) {
	user, err := s.userRepo.FindByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.ErrUserNotFound // حفظ خطای دقیق جهت دیباگ
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if err := validateUserStatus(user); err != nil {
		return nil, err
	}

	switch otpType {
	case otp.TypeEmail:
		if !user.IsVerifiedEmail {
			return nil, exception.ErrNotVerifiedEmailCredentials
		}
	case otp.TypePhone:
		if !user.IsVerifiedPhone {
			return nil, exception.ErrNotVerifiedPhoneCredentials
		}
	}

	isVerified, err := s.otpService.InvalidateIsVerified(ctx, identifier, otpType)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	if isVerified == nil || !*isVerified {
		if otpType == otp.TypeEmail {
			return nil, exception.ErrNotVerifiedEmailOTP
		}
		return nil, exception.ErrNotVerifiedPhoneOTP
	}

	return s.issueTokenPair(ctx, user.ID, user.CompanyID, user.Username)
}

func validateUserStatus(user *entity.User) error {
	if user == nil {
		return exception.ErrUserNotFound
	}
	if !user.IsActive {
		return exception.ErrUserInactive
	}
	return nil
}

func (s *authService) RegisterMeWithCompany(ctx context.Context, req service_contract.RegisterWithCompanyRequest) (*service_contract.UserResponse, *service_contract.CompanyResponse, error) {
	var c *entity.Company
	var u *entity.User
	err := s.trxManager.WithTransaction(ctx, func(trxCtx context.Context) error {
		c = &entity.Company{
			Name:     req.Company.Name,
			Code:     req.Company.Code,
			IsActive: true,
		}
		if err := s.companyRepo.Create(ctx, c); err != nil {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Me.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		u = &entity.User{
			PasswordHash: string(hash),
			IsActive:     true,
			CompanyID:    c.ID,
		}

		if req.Me.Email == "" && req.Me.Phone == "" {
			return exception.ErrEmptyEmailPhone
		}

		if _, err := s.checkCrdentionls(ctx, req.Me.Email, req.Me.Phone, req.Me.Username, u); err != nil {
			return err
		}

		if err := s.userRepo.Create(ctx, u); err != nil {
			return err
		}

		if err := s.rbacRepo.AddRoleForUser(ctx, strconv.FormatUint(uint64(u.ID), 10), entity.RoleSuperAdmin, strconv.FormatUint(uint64(c.ID), 10)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	usr, com := service_contract.ToUserResponse(u, []string{entity.RoleSuperAdmin}), service_contract.ToCompanyResponse(c)
	return &usr, &com, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*service_contract.TokenPairResponse, error) {
	payload, err := s.tokenMaker.VerifyToken(refreshToken)
	if err != nil {
		return nil, exception.Wrap(exception.ErrRefreshTokenInvalid, err)
	}
	if payload.TokenType != paseto.RefreshToken {
		return nil, exception.ErrRefreshTokenInvalid
	}

	stored, err := s.tokenRepo.GetRefreshSession(ctx, payload.UserID, payload.ID.String())
	if err != nil || stored != refreshToken {
		return nil, exception.ErrRefreshTokenInvalid
	}

	// Rotate: purge the used refresh session before issuing a new pair so
	// a stolen-then-replayed refresh token cannot be redeemed twice.
	if sessionKey, err := s.tokenRepo.DeleteRefreshSession(ctx, payload.UserID, payload.ID.String()); err != nil {
		s.log.Warn("failed to purge rotated refresh session", logger.Err(err), logger.String("key", sessionKey))
	}

	companyID, err := parseUint(payload.CompanyID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrRefreshTokenInvalid, err)
	}

	userID, err := parseUint(payload.UserID)
	if err != nil {
		return nil, exception.Wrap(exception.ErrRefreshTokenInvalid, err)
	}
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

	return s.issueTokenPair(ctx, user.ID, user.CompanyID, user.Username)
}

func (s *authService) Logout(ctx context.Context, accessToken string) error {
	payload, err := s.tokenMaker.VerifyToken(accessToken)
	if err != nil {
		return exception.Wrap(exception.ErrTokenInvalid, err)
	}

	// Blacklist the access token's jti for however long it would otherwise
	// remain valid -- this is what AuthMiddleware checks on every request.
	remaining := time.Until(payload.ExpiredAt)
	if remaining > 0 {
		if err := s.tokenRepo.BlacklistAccessToken(ctx, payload.ID.String(), remaining); err != nil {
			return exception.Wrap(exception.ErrInternal, fmt.Errorf("blacklist access token: %w", err))
		}
	}

	// Best-effort: purge every refresh session for this user so logout
	// terminates the whole session rather than only this access token.
	if err := s.tokenRepo.DeleteUserRefreshSessions(ctx, payload.UserID); err != nil {
		s.log.Warn("refresh session scan failed during logout", logger.Err(err), logger.String("user_id", payload.UserID))
	}

	s.log.Info("user logged out", logger.String("user_id", payload.UserID), logger.String("jti", payload.ID.String()))
	return nil
}

func (s *authService) RegisterWithOTP(ctx context.Context, req service_contract.VerifyRegisterWithOTPRequest) (*service_contract.UserResponse, *service_contract.CompanyResponse, error) {
	rawPayload, err := s.otpService.VerifyOTP(ctx, "=does not matter=", otp.TypeRegistrionUserToCompany, req.Code)
	if err != nil {
		return nil, nil, exception.Wrap(exception.ErrInternal, err)
	}
	payload, ok := rawPayload.(*otp.RegisterUserPayload)
	if !ok {
		s.log.Warn("fail to type assert raw-payload for regiser-user-otp", logger.Any("raw-payload", rawPayload))
		return nil, nil, exception.Wrap(exception.ErrInternal, err)
	}

	var c *entity.Company
	var u *entity.User
	err = s.trxManager.WithTransaction(ctx, func(trxCtx context.Context) error {
		com_id, err := strconv.ParseUint(payload.CompanyID, 10, 64)
		if err != nil {
			return err
		}
		if c, err = s.companyRepo.FindByID(ctx, uint(com_id)); err != nil {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Me.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		u = &entity.User{
			PasswordHash: string(hash),
			IsActive:     true,
			CompanyID:    c.ID,
		}

		if req.Me.Email == "" && req.Me.Phone == "" {
			return exception.ErrEmptyEmailPhone
		}

		if _, err := s.checkCrdentionls(ctx, req.Me.Email, req.Me.Phone, req.Me.Username, u); err != nil {
			return err
		}

		if err := s.userRepo.Create(ctx, u); err != nil {
			return err
		}

		if err := s.rbacRepo.AddRolesForUser(ctx, strconv.FormatUint(uint64(u.ID), 10), payload.Roles, strconv.FormatUint(uint64(c.ID), 10)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	usr, com := service_contract.ToUserResponse(u, payload.Roles), service_contract.ToCompanyResponse(c)
	return &usr, &com, nil
}

func (s *authService) issueTokenPair(ctx context.Context, userID, companyID uint, username string) (*service_contract.TokenPairResponse, error) {
	userIDStr := fmt.Sprint(userID)
	companyIDStr := fmt.Sprint(companyID)

	accessToken, accessPayload, err := s.tokenMaker.CreateToken(userIDStr, companyIDStr, username, paseto.AccessToken, s.AuthServiceConfig.AccessTokenTTL)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	refreshToken, refreshPayload, err := s.tokenMaker.CreateToken(userIDStr, companyIDStr, username, paseto.RefreshToken, s.AuthServiceConfig.RefreshTokenTTL)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	if err := s.tokenRepo.SaveRefreshSession(ctx, userIDStr, refreshPayload.ID.String(), refreshToken, s.RefreshTokenTTL); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, fmt.Errorf("persist refresh session: %w", err))
	}

	s.log.Info("token pair issued", logger.String("user_id", userIDStr), logger.String("company_id", companyIDStr), logger.String("username", username))
	return &service_contract.TokenPairResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt.Format(time.RFC3339),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt.Format(time.RFC3339),
	}, nil
}

func (s *authService) checkCrdentionls(ctx context.Context, email, phone, username string, user *entity.User) (*bool, error) {
	// check username uniqess
	_, err := s.userRepo.FindByIdentifier(ctx, username)
	if err == nil {
		return nil, exception.ErrUserUsernameAlreadyExists
	} else if !errors.Is(err, exception.ErrUserNotFound) {
		return nil, exception.Wrap(exception.ErrInternal, err)
	} else {
		user.Username = username
	}

	// check email uniqess & verification
	if email != "" {
		_, err = s.userRepo.FindByIdentifier(ctx, email)
		if err == nil {
			return nil, exception.ErrUserEmailAlreadyExists
		} else if !errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
		is, err := s.otpService.InvalidateIsVerified(ctx, email, otp.TypeEmail)
		if err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		} else if !*is {
			return nil, exception.ErrNotVerifiedEmailCredentials
		}
		user.Email = &email
		user.IsVerifiedEmail = true
	} else {
		user.Email = nil
		user.IsVerifiedEmail = false
	}

	// check phone uniqess & verification
	if phone != "" {
		_, err = s.userRepo.FindByIdentifier(ctx, phone)
		if err == nil {
			return nil, exception.ErrUserPhoneAlreadyExists
		} else if !errors.Is(err, exception.ErrUserNotFound) {
			return nil, exception.Wrap(exception.ErrInternal, err)
		}
		is, err := s.otpService.InvalidateIsVerified(ctx, phone, otp.TypePhone)
		if err != nil {
			return nil, exception.Wrap(exception.ErrInternal, err)
		} else if !*is {
			return nil, exception.ErrNotVerifiedPhoneCredentials
		}
		user.Phone = &phone
		user.IsVerifiedPhone = true
	} else {
		user.Phone = nil
		user.IsVerifiedPhone = false
	}
	ok := true
	return &ok, nil
}

//TODO

func (s *authService) SendOTP(ctx context.Context, identifier, otpType string) (*service_contract.SendAuthOTPResponse, error) {
	if !(otpType == otp.TypeEmail.String() || otpType == otp.TypePhone.String()) {
		return nil, exception.ErrBadRequest
	}

	//TODO : real send otp

	r, err := s.otpService.SendOTP(ctx, identifier, otp.Type(otpType), nil)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	resp := &service_contract.SendAuthOTPResponse{
		Code:            r.Code,
		Message:         r.Message,
		ExpiresInSecond: r.ExpiresInSecond,
	}

	return resp, nil
}

func (s *authService) VerifyOTP(ctx context.Context, identifier, otpType, code string) (*bool, error) {
	if !(otpType == otp.TypeEmail.String() || otpType == otp.TypePhone.String()) {
		return nil, exception.ErrBadRequest
	}

	err := s.otpService.InvalidateOTPAndSetVerified(ctx, identifier, otp.Type(otpType), code)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	ok := true
	return &ok, nil
}

// // SendOTP generates a 6-digit numeric code and stores it in Redis via
// // OTPRepository, keyed by (identifier, otpType) with the configured TTL.
// //
// // NOTE: actual delivery (an email or SMS provider) is intentionally not
// // wired up here -- this is the integration point for that. Outside of
// // production, the generated code is both logged and returned in the
// // response so the OTP flow is testable end-to-end without a provider
// // configured; in production it is only logged, never returned to the
// // caller.
// func (s *authService) SendOTP(ctx context.Context, identifier, otpType string) (*service_contract.SendAuthOTPResponse, error) {
// 	user, err := s.userRepo.FindByIdentifier(ctx, identifier)
// 	if err != nil {
// 		if errors.Is(err, exception.ErrUserNotFound) {
// 			return nil, exception.ErrUserNotFound
// 		}
// 		return nil, exception.Wrap(exception.ErrInternal, err)
// 	}

// 	if !user.IsActive {
// 		return nil, exception.ErrUserInactive
// 	}

// 	code, err := s.generateOTPCode()
// 	if err != nil {
// 		return nil, exception.Wrap(exception.ErrInternal, err)
// 	}

// 	if err := s.otpRepo.SaveOTP(ctx, identifier, otpType, code, s.AuthServiceConfig.OTPTTL); err != nil {
// 		return nil, exception.Wrap(exception.ErrInternal, err)
// 	}

// 	resp := &service_contract.SendAuthOTPResponse{
// 		Message:         "otp sent",
// 		ExpiresInSecond: int(s.OTPTTL.Seconds()),
// 	}

// 	if s.AuthServiceConfig.AppEnv == "production" {
// 		s.log.Info("otp generated", logger.String("identifier", identifier), logger.String("type", otpType))
// 	} else {
// 		resp.Code = code
// 		s.log.Info("otp generated (non-production build, code included in response)",
// 			logger.String("identifier", identifier), logger.String("type", otpType))
// 	}

// 	return resp, nil
// }

// // VerifyOTP authenticates an existing user identified by email or phone.
// // It intentionally does NOT self-register a brand-new user/company on an
// // identifier with no match: there's no company to assign a fresh account
// // to from just an email or phone number, so OTP-based "registration" here
// // means activating a user an admin already created via POST /users (see
// // that DTO's doc comment) -- not open self-signup.
// func (s *authService) VerifyOTP(ctx context.Context, identifier, otpType, code string) (*service_contract.TokenPairResponse, error) {
// 	valid, err := s.otpRepo.VerifyOTP(ctx, identifier, otpType, code)
// 	if err != nil {
// 		return nil, exception.Wrap(exception.ErrInternal, err)
// 	}
// 	if !valid {
// 		return nil, exception.ErrOTPInvalid
// 	}

// 	// Single-use: burn the code once it's been successfully verified.
// 	if err := s.otpRepo.DeleteOTP(ctx, identifier, otpType); err != nil {
// 		s.log.Warn("failed to delete verified otp", logger.Err(err), logger.String("identifier", identifier))
// 	}

// 	user, err := s.userRepo.FindByIdentifier(ctx, identifier)
// 	if err != nil {
// 		if errors.Is(err, exception.ErrUserNotFound) {
// 			return nil, exception.ErrUserNotFound
// 		}
// 		return nil, exception.Wrap(exception.ErrInternal, err)
// 	}

// 	if !user.IsActive {
// 		return nil, exception.ErrUserInactive
// 	}

// 	return s.issueTokenPair(ctx, user.ID, user.CompanyID, user.Username)
// }
