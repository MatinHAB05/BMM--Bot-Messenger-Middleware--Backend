// Package otp contains the OTP (One-Time Password) domain module: the
// Strategy-pattern contract that lets each OTP type (phone, email, link,
// and anything added later) define its own key format, TTL, code format,
// and payload shape, while OTPService and OTPRepository stay completely
// generic. Adding a new OTP type is purely additive -- implement
// Strategy, register an instance with OTPService -- and never requires
// touching this package, the repository, or the service (Open/Closed
// Principle).
package otp

import (
	"fmt"
	"time"

	"messenger-backend/internal/domain/exception"
)

// Type identifies a category of OTP. It's a plain string rather than an
// int-backed enum so it round-trips cleanly through JSON, Redis keys,
// and HTTP request bodies (e.g. the `type` field on
// SendAuthOTPRequest/VerifyAuthOTPRequest) with no extra glue code.
type Type string

const (
	TypePhone                   Type = "phone"
	TypeEmail                   Type = "email"
	TypeLink                    Type = "link"
	TypeLinkJustChnnel          Type = "link-channel"
	TypeRegistrionUserToCompany Type = "register-user-to-company"
)

// String implements fmt.Stringer so Type formats naturally in log
// fields and %s-style key formatting.
func (t Type) String() string { return string(t) }

// ParseType validates a caller-supplied string (typically an HTTP
// request's `type` field) against the OTP types a running OTPService
// actually has strategies for (see OTPService.RegisteredTypes). Doing
// the lookup against a caller-supplied list rather than a hardcoded set
// of the three built-in types means a deployment that registers an
// additional Strategy gets validation for it for free.
func ParseType(s string, registered []Type) (Type, error) {
	for _, t := range registered {
		if string(t) == s {
			return t, nil
		}
	}
	return "", fmt.Errorf("%w: %q", exception.ErrOTPUnsupportedType, s)
}

// Payload is the data stored alongside an OTP code. Every payload must
// be able to hand back its own code for comparison against
// caller-submitted input; beyond that, a payload can carry whatever
// extra fields its Strategy defines (see LinkPayload.CompanyID). That's
// what makes storage here "not restricted to a basic string" -- payloads
// are marshaled to JSON by OTPService before reaching OTPRepository, so
// a concrete payload only needs to be an ordinary JSON-taggable struct.
type Payload interface {
	// Code returns the OTP secret this payload carries, for the service
	// layer to compare against caller-submitted input.
	Code() string
}

// Strategy is the Strategy-pattern contract each OTP type implements.
// It owns everything that varies by type -- key formatting, TTL, code
// generation, and payload construction/rehydration -- so OTPService and
// OTPRepository can stay entirely type-agnostic.
type Strategy interface {
	// Type reports which OTP type this strategy implements.
	Type() Type

	// Key computes the storage key for a pending OTP. Most strategies
	// key off identifier (e.g. a phone number or email address); a
	// magic-link OTP instead keys off code itself, since once a code is
	// embedded in a URL it IS the lookup handle -- no other identifier
	// is available when the link is opened. Implementations use
	// whichever argument is meaningful to them and ignore the other.
	Key(identifier, code string) string
	KeyWithID(identifier string) string

	// DefaultTTL is how long an OTP of this type lives.
	DefaultTTL() time.Duration

	// GenerateCode produces a new OTP secret in this type's own format
	// -- e.g. a short numeric code for phone/email (fit for a human to
	// type) or a longer URL-safe token for a link (fit for a URL, never
	// typed by hand).
	GenerateCode() (string, error)

	// NewPayload builds this strategy's concrete Payload around a
	// freshly generated code plus caller-supplied metadata (e.g.
	// {"company_id": 42} for a link OTP). Implementations should ignore
	// metadata keys they don't understand and error only when a key
	// they DO understand is present but malformed.
	NewPayload(code string, metadata map[string]any) (Payload, error)

	// EmptyPayload returns a zero-value instance of this strategy's
	// concrete Payload type, for the service layer to json.Unmarshal
	// stored bytes into. encoding/json needs a concrete target type,
	// which is why this can't just live on the generic Payload
	// interface.
	EmptyPayload() Payload
}
