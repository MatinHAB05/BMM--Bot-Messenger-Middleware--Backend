package repository_contract

import (
	"context"
	"time"
)

// OTPRepository is the storage-agnostic contract for persisting OTP
// payloads. It deals only in opaque, pre-serialized bytes under a key --
// encoding a Payload into bytes and decoding bytes back into a concrete
// Payload is the otp.Strategy's / OTPService's job, not the
// repository's. That split is what lets a single implementation (Redis
// today; in-memory for tests, or anything else tomorrow) support every
// OTP type's payload -- including ones that don't exist yet -- without
// ever needing to change.
//
// This replaces the previous, identifier/otpType-specific OTPRepository:
// key formatting now belongs to otp.Strategy (each type formats its key
// differently -- see otp.Strategy.Key), so the repository itself no
// longer needs to know about identifiers or types at all.
type OTPRepository interface {
	// Save stores payload under key with the given TTL, replacing
	// whatever was previously stored there.
	Save(ctx context.Context, key string, payload []byte, ttl time.Duration) error

	// Get retrieves the raw payload bytes stored under key.
	// Returns exception.ErrOTPNotFound if key doesn't exist or expired.
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes key, e.g. to burn a code after single use.
	// Returns exception.ErrOTPNotFound if key didn't exist.
	Delete(ctx context.Context, key string) error
	DeleteVerified(ctx context.Context, key string) error
	SetVerfied(ctx context.Context, key string, ttl time.Duration) error
	IsVerified(ctx context.Context, key string) (*bool, error)
	// Exists reports whether key is currently present.
	Exists(ctx context.Context, key string) (bool, error)

	// TTL returns the remaining time-to-live for key.
	// Returns exception.ErrOTPNotFound if key doesn't exist.
	TTL(ctx context.Context, key string) (time.Duration, error)
}
