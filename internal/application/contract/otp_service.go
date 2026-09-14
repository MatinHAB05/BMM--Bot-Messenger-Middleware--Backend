package service_contract

import (
	"context"
	"time"

	"messenger-backend/internal/domain/otp"
)

// SendOTPResult is the caller-facing summary of a freshly sent OTP. Code
// mirrors the existing convention seen in SendAuthOTPResponse /
// SendLinkChatOTPResponse: populated outside production (no delivery
// provider wired in yet) so a flow is testable end-to-end, omitted in
// production where the code should only ever reach the identifier it
// was sent to.
type SendOTPResult struct {
	Message         string `json:"message"`
	ExpiresInSecond int    `json:"expires_in_seconds"`
	Code            string `json:"code,omitempty"`
}

// OTPService is the single, type-agnostic entry point for every OTP
// flow (phone, email, link, and any type registered later). It's
// "centralized" in the sense the design calls for: any caller --
// AuthService, ChatService, or something written next year -- goes
// through this one interface, and which concrete behavior runs for a
// given otpType is resolved internally via the otp.Strategy registered
// for it. Callers never branch on type themselves.
type OTPService interface {
	// SendOTP generates a new code for (identifier, otpType) via that
	// type's Strategy, persists it, and returns a caller-facing
	// summary. metadata is passed through to Strategy.NewPayload
	// verbatim (e.g. {"company_id": companyID} for otp.TypeLink); pass
	// nil for types that don't need any.
	SendOTP(ctx context.Context, identifier string, otpType otp.Type, metadata map[string]any) (*SendOTPResult, error)

	// VerifyOTP checks code against whatever is stored for
	// (identifier, otpType) and returns the full Payload on success, so
	// callers can pull out type-specific fields (e.g. a LinkPayload's
	// CompanyID) without OTPService needing to know about them. It does
	// NOT delete the stored OTP -- callers wanting single-use semantics
	// should follow a successful VerifyOTP with InvalidateOTP.
	VerifyOTP(ctx context.Context, identifier string, otpType otp.Type, code string) (otp.Payload, error)

	// InvalidateOTP burns a pending OTP early -- typically called right
	// after a successful VerifyOTP for single-use semantics, or to let
	// a caller cancel a code they no longer want honored.
	GetAndInvalidateOTP(ctx context.Context, identifier string, otpType otp.Type, code string) (otp.Payload, error)
	InvalidateOTP(ctx context.Context, identifier string, otpType otp.Type, code string) error
	InvalidateOTPAndSetVerified(ctx context.Context, identifier string, otpType otp.Type, code string) error
	IsVerified(ctx context.Context, identifier string, otpType otp.Type) (*bool, error)
	InvalidateIsVerified(ctx context.Context, identifier string, otpType otp.Type) (*bool, error)

	// TTL reports how much longer a pending OTP has before it expires.
	TTL(ctx context.Context, identifier string, otpType otp.Type, code string) (time.Duration, error)

	// RegisteredTypes lists the OTP types this service currently has a
	// Strategy for -- useful for validating a request's `type` field
	// (see otp.ParseType) without hardcoding the type list at call sites.
	RegisteredTypes() []otp.Type
}
