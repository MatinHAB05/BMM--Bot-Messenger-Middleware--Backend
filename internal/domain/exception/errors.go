// Package exception defines the application's custom error vocabulary.
// A single AppError type carries a stable machine-readable Code, a
// human-readable Message, the HTTP status it should map to, and an
// optionally wrapped lower-level error. Handlers in the presentation
// layer never leak raw errors to clients -- they translate them into
// one of these sentinels first.
package exception

import (
	"errors"
	"net/http"
)

// ==========================================
// 1. Raw Domain & Infrastructure Errors
// ==========================================
var (
	// User domain errors
	ErrDomainUserNotFound              = errors.New("user not found")
	ErrDomainUserAlreadyExists         = errors.New("user already exists")
	ErrDomainUserEmailAlreadyExists    = errors.New("user email already exists")
	ErrDomainUserPhoneAlreadyExists    = errors.New("user phone already exists")
	ErrDomainUserUsernameAlreadyExists = errors.New("user username already exists")
	ErrDomainUserInactive              = errors.New("user account is deactivated")

	// Auth & OTP domain errors
	ErrDomainInvalidCredentials   = errors.New("invalid credentials")
	ErrDomainTokenInvalid         = errors.New("token is invalid")
	ErrDomainTokenExpired         = errors.New("token has expired")
	ErrDomainTokenRevoked         = errors.New("token has been revoked")
	ErrDomainMissingToken         = errors.New("missing authorization token")
	ErrDomainRefreshTokenInvalid  = errors.New("invalid or expired refresh token")
	ErrDomainTokenNotFound        = errors.New("token not found")
	ErrDomainOTPInvalid           = errors.New("otp code invalid or expired")
	ErrDomainOTPNotFound          = errors.New("otp not found or expired")
	ErrDomainOTPUnsupportedType   = errors.New("unsupported otp type")
	ErrDomainOTPInvalidMetadata   = errors.New("invalid otp metadata")
	ErrDomainUnverifiedCredential = errors.New("unverified credential")
	ErrDomainInvalidPayload       = errors.New("invalid or unparseable payload")
	ErrDomainMiddlewareAssertion  = errors.New("middleware assertion failed")

	// Company domain errors
	ErrDomainCompanyNotFound      = errors.New("company not found")
	ErrDomainCompanyInactive      = errors.New("company is deactivated")
	ErrDomainCompanyAlreadyExists = errors.New("company already exists")

	// Chat, Channel, Message & Attachment domain errors
	ErrDomainChannelNotFound              = errors.New("channel not found")
	ErrDomainChatNotFound                 = errors.New("chat not found")
	ErrDomainChatAlreadyExists            = errors.New("chat already exists")
	ErrDomainChatCompanyAlreadyRegistered = errors.New("chat is already registered with this company")
	ErrDomainMessageNotFound              = errors.New("message not found")
	ErrDomainSentBaleMsgNotFound          = errors.New("sent bale msg not found")
	ErrDomainMediaGroupMsgNotFound        = errors.New("media group message not found")
	ErrDomainAttachmentNotFound           = errors.New("attachment not found")

	// Messenger / platform delivery domain errors
	ErrDomainPlatformForbidden    = errors.New("messenger platform forbidden")
	ErrDomainPlatformBadRequest   = errors.New("messenger platform bad request")
	ErrDomainPlatformUnauthorized = errors.New("messenger platform unauthorized")
	ErrDomainPlatformRateLimited  = errors.New("messenger platform rate limited")
	ErrDomainPlatformNotFound     = errors.New("messenger platform target not found")
	ErrDomainPlatformConflict     = errors.New("messenger platform conflict")

	// RBAC / Policy domain errors
	ErrDomainRoleNotFound       = errors.New("role not found")
	ErrDomainPermissionNotFound = errors.New("permission not found")
	ErrDomainEnforcerNil        = errors.New("enforcer instance is nil")
	ErrDomainRoleSyncFailed     = errors.New("role sync failed")

	// General Infrastructure & System errors
	ErrDomainDatabaseOperation   = errors.New("database operation failed")
	ErrDomainCacheOperation      = errors.New("cache operation failed")
	ErrDomainForbidden           = errors.New("action forbidden")
	ErrDomainRateLimited         = errors.New("rate limit exceeded")
	ErrDomainInternal            = errors.New("internal error")
	ErrDomainBadRequest          = errors.New("bad request")
	ErrDomainUnsupportedPlatform = errors.New("unsupported platform")
	ErrDomainHandlerRouting      = errors.New("handler routing failed")
	ErrDomainComingSoon          = errors.New("feature is not completed yet")
)

// ==========================================
// 2. Application Sentinel AppErrors (Synced 1-to-1)
// ==========================================
var (
	// User
	ErrUserNotFound              = &AppError{Code: "USER_NOT_FOUND", Message: "user not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainUserNotFound}
	ErrUserAlreadyExists         = &AppError{Code: "USER_ALREADY_EXISTS", Message: "user already exists", HTTPStatus: http.StatusConflict, Err: ErrDomainUserAlreadyExists}
	ErrUserEmailAlreadyExists    = &AppError{Code: "USER_EMAIL_ALREADY_EXISTS", Message: "email already in use", HTTPStatus: http.StatusConflict, Err: ErrDomainUserEmailAlreadyExists}
	ErrUserPhoneAlreadyExists    = &AppError{Code: "USER_PHONE_ALREADY_EXISTS", Message: "phone number already in use", HTTPStatus: http.StatusConflict, Err: ErrDomainUserPhoneAlreadyExists}
	ErrUserUsernameAlreadyExists = &AppError{Code: "USER_USERNAME_ALREADY_EXISTS", Message: "username already in use", HTTPStatus: http.StatusConflict, Err: ErrDomainUserUsernameAlreadyExists}
	ErrUserInactive              = &AppError{Code: "USER_INACTIVE", Message: "user account is deactivated", HTTPStatus: http.StatusForbidden, Err: ErrDomainUserInactive}

	// Auth & OTP
	ErrInvalidCredentials          = &AppError{Code: "INVALID_CREDENTIALS", Message: "invalid username or password", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainInvalidCredentials}
	ErrTokenInvalid                = &AppError{Code: "TOKEN_INVALID", Message: "token is invalid", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainTokenInvalid}
	ErrTokenExpired                = &AppError{Code: "TOKEN_EXPIRED", Message: "token has expired", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainTokenExpired}
	ErrTokenRevoked                = &AppError{Code: "TOKEN_REVOKED", Message: "token has been revoked", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainTokenRevoked}
	ErrMissingToken                = &AppError{Code: "MISSING_TOKEN", Message: "authorization token is missing", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainMissingToken}
	ErrMiddlewareTokensAssetion    = &AppError{Code: "MIDDLEWARE_ASSERTION_FAILED", Message: "failed to retrieve token info from middleware context", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainMiddlewareAssertion}
	ErrRefreshTokenInvalid         = &AppError{Code: "REFRESH_TOKEN_INVALID", Message: "refresh token is invalid, expired, or already used", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainRefreshTokenInvalid}
	ErrTokenNotFound               = &AppError{Code: "TOKEN_NOT_FOUND", Message: "token not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainTokenNotFound}
	ErrOTPInvalid                  = &AppError{Code: "OTP_INVALID", Message: "the provided OTP code is invalid or has expired", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainOTPInvalid}
	ErrOTPNotFound                 = &AppError{Code: "OTP_NOT_FOUND", Message: "the requested OTP code was not found or has expired", HTTPStatus: http.StatusNotFound, Err: ErrDomainOTPNotFound}
	ErrOTPUnsupportedType          = &AppError{Code: "OTP_UNSUPPORTED_TYPE", Message: "unsupported OTP type", HTTPStatus: http.StatusBadRequest, Err: ErrDomainOTPUnsupportedType}
	ErrOTPInvalidMetadata          = &AppError{Code: "OTP_INVALID_METADATA", Message: "invalid OTP metadata provided", HTTPStatus: http.StatusBadRequest, Err: ErrDomainOTPInvalidMetadata}
	ErrInvalidPayload              = &AppError{Code: "INVALID_PAYLOAD", Message: "invalid request payload format or type assertion failure", HTTPStatus: http.StatusBadRequest, Err: ErrDomainInvalidPayload}
	ErrEmptyEmailPhone             = &AppError{Code: "EMPTY_PHONE_EMAIL", Message: "at least one of email or phone must be provided", HTTPStatus: http.StatusBadRequest, Err: ErrDomainBadRequest}
	ErrNotVerifiedCredentials      = &AppError{Code: "NOT_VERIFIED_CREDENTIALS", Message: "credentials (email or phone) are not verified", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainUnverifiedCredential}
	ErrNotVerifiedEmailCredentials = &AppError{Code: "NOT_VERIFIED_EMAIL_CREDENTIAL", Message: "email credential is not verified", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainUnverifiedCredential}
	ErrNotVerifiedPhoneCredentials = &AppError{Code: "NOT_VERIFIED_PHONE_CREDENTIAL", Message: "phone credential is not verified", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainUnverifiedCredential}
	ErrNotVerifiedEmailOTP         = &AppError{Code: "NOT_VERIFIED_EMAIL_OTP", Message: "email OTP is not verified", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainUnverifiedCredential}
	ErrNotVerifiedPhoneOTP         = &AppError{Code: "NOT_VERIFIED_PHONE_OTP", Message: "phone OTP is not verified", HTTPStatus: http.StatusUnauthorized, Err: ErrDomainUnverifiedCredential}

	// Company
	ErrCompanyNotFound      = &AppError{Code: "COMPANY_NOT_FOUND", Message: "company not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainCompanyNotFound}
	ErrCompanyInactive      = &AppError{Code: "COMPANY_INACTIVE", Message: "company is deactivated", HTTPStatus: http.StatusForbidden, Err: ErrDomainCompanyInactive}
	ErrCompanyAlreadyExists = &AppError{Code: "COMPANY_ALREADY_EXISTS", Message: "company code or name already exists", HTTPStatus: http.StatusConflict, Err: ErrDomainCompanyAlreadyExists}

	// Chat, Channel, Message & Attachment
	ErrChannelNotFound              = &AppError{Code: "CHANNEL_NOT_FOUND", Message: "channel not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainChannelNotFound}
	ErrChatNotFound                 = &AppError{Code: "CHAT_NOT_FOUND", Message: "chat not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainChatNotFound}
	ErrChatAlreadyExists            = &AppError{Code: "CHAT_ALREADY_EXISTS", Message: "this chat is already registered", HTTPStatus: http.StatusConflict, Err: ErrDomainChatAlreadyExists}
	ErrChatCompanyAlreadyRegistered = &AppError{Code: "CHAT_ALREADY_REG_WITH_COMPANY", Message: "this chat is already registered for your company", HTTPStatus: http.StatusConflict, Err: ErrDomainChatCompanyAlreadyRegistered}
	ErrMessageNotFound              = &AppError{Code: "MESSAGE_NOT_FOUND", Message: "message not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainMessageNotFound}
	ErrSentBaleMsgNotFound          = &AppError{Code: "SENT_BALE_MSG_NOT_FOUND", Message: "sent bale message not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainSentBaleMsgNotFound}
	ErrMediaGroupMsgNotFound        = &AppError{Code: "MEDIA_GROUP_MSG_NOT_FOUND", Message: "media group message not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainMediaGroupMsgNotFound}
	ErrAttachmentNotFound           = &AppError{Code: "ATTACHMENT_NOT_FOUND", Message: "attachment not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainAttachmentNotFound}

	// Messenger / platform delivery (synced with pkg/messenger's
	// engine-agnostic error sentinels -- see broadcastService.mapMessengerErr)
	ErrPlatformForbidden    = &AppError{Code: "PLATFORM_FORBIDDEN", Message: "the messaging platform denied this action (e.g. the bot was blocked or removed)", HTTPStatus: http.StatusForbidden, Err: ErrDomainPlatformForbidden}
	ErrPlatformBadRequest   = &AppError{Code: "PLATFORM_BAD_REQUEST", Message: "the messaging platform rejected the request as malformed", HTTPStatus: http.StatusBadRequest, Err: ErrDomainPlatformBadRequest}
	ErrPlatformUnauthorized = &AppError{Code: "PLATFORM_UNAUTHORIZED", Message: "the messaging platform rejected this bot's credentials", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainPlatformUnauthorized}
	ErrPlatformRateLimited  = &AppError{Code: "PLATFORM_RATE_LIMITED", Message: "the messaging platform is rate-limiting this bot", HTTPStatus: http.StatusTooManyRequests, Err: ErrDomainPlatformRateLimited}
	ErrPlatformNotFound     = &AppError{Code: "PLATFORM_TARGET_NOT_FOUND", Message: "the messaging platform reports this chat or message no longer exists", HTTPStatus: http.StatusNotFound, Err: ErrDomainPlatformNotFound}
	ErrPlatformConflict     = &AppError{Code: "PLATFORM_CONFLICT", Message: "the messaging platform reports a conflicting request", HTTPStatus: http.StatusConflict, Err: ErrDomainPlatformConflict}

	// RBAC
	ErrRoleNotFound       = &AppError{Code: "ROLE_NOT_FOUND", Message: "role not found or has no permissions", HTTPStatus: http.StatusNotFound, Err: ErrDomainRoleNotFound}
	ErrPermissionNotFound = &AppError{Code: "PERMISSION_NOT_FOUND", Message: "permission not found", HTTPStatus: http.StatusNotFound, Err: ErrDomainPermissionNotFound}
	ErrEnforcerNil        = &AppError{Code: "ENFORCER_NIL", Message: "enforcer instance is not initialized", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainEnforcerNil}
	ErrRoleSyncFailed     = &AppError{Code: "ROLE_SYNC_FAILED", Message: "failed to synchronize user roles with the policy store", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainRoleSyncFailed}

	// General & Infrastructure
	ErrDatabaseOperation   = &AppError{Code: "DATABASE_ERROR", Message: "database operation failed", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainDatabaseOperation}
	ErrCacheOperation      = &AppError{Code: "CACHE_ERROR", Message: "cache operation failed", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainCacheOperation}
	ErrForbidden           = &AppError{Code: "FORBIDDEN", Message: "you do not have permission to perform this action", HTTPStatus: http.StatusForbidden, Err: ErrDomainForbidden}
	ErrRateLimited         = &AppError{Code: "RATE_LIMITED", Message: "too many requests, please try again later", HTTPStatus: http.StatusTooManyRequests, Err: ErrDomainRateLimited}
	ErrInternal            = &AppError{Code: "INTERNAL_ERROR", Message: "internal server error", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainInternal}
	ErrBadRequest          = &AppError{Code: "BAD_REQUEST", Message: "invalid request payload", HTTPStatus: http.StatusBadRequest, Err: ErrDomainBadRequest}
	ErrBadLoginRequest     = &AppError{Code: "BAD_LOGIN_REQUEST", Message: "exactly one login option must be requested", HTTPStatus: http.StatusBadRequest, Err: ErrDomainBadRequest}
	ErrUnsupportedPlatform = &AppError{Code: "UNSUPPORTED_PLATFORM", Message: "one or more requested platforms are not supported", HTTPStatus: http.StatusBadRequest, Err: ErrDomainUnsupportedPlatform}
	ErrSendMessagePlatform = &AppError{Code: "SEND_MESSAGE_PLATFORM_ERROR", Message: "failed to send message via platform provider", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainInternal}
	ErrFalsyHandlerRouting = &AppError{Code: "HANDLER_ROUTING_FAILED", Message: "failed to route handlers", HTTPStatus: http.StatusInternalServerError, Err: ErrDomainHandlerRouting}
	ErrComingSoon          = &AppError{Code: "COMING_SOON", Message: "this feature is not yet completed", HTTPStatus: http.StatusMethodNotAllowed, Err: ErrDomainComingSoon}
)

// ==========================================
// 3. AppError Definition & Helper Methods
// ==========================================
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Is lets errors.Is match against sentinel values even after Wrap has produced
// a new *AppError carrying additional context, by comparing stable Codes.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Wrap attaches underlying error context to a sentinel AppError without
// mutating the shared sentinel value itself.
func Wrap(base *AppError, err error) *AppError {
	return &AppError{
		Code:       base.Code,
		Message:    base.Message,
		HTTPStatus: base.HTTPStatus,
		Err:        err,
	}
}
