package otp

import (
	"fmt"
	"time"
)

// identifierKeyedStrategy implements the OTP types (phone, email) that
// key off the identifier being verified and need nothing more than the
// code itself in their payload. Phone and Email differ only in Type()
// and key prefix, so they share this one implementation rather than
// being hand-duplicated.
type identifierKeyedStrategy struct {
	otpType    Type
	ttl        time.Duration
	codeLength int
	gen        CodeGenerator
}

// NewPhoneStrategy builds the Strategy for otp:phone:{phone} OTPs.
func NewPhoneStrategy(ttl time.Duration, codeLength int, gen CodeGenerator) Strategy {
	return &identifierKeyedStrategy{otpType: TypePhone, ttl: ttl, codeLength: codeLength, gen: gen}
}

// NewEmailStrategy builds the Strategy for otp:email:{email} OTPs.
func NewEmailStrategy(ttl time.Duration, codeLength int, gen CodeGenerator) Strategy {
	return &identifierKeyedStrategy{otpType: TypeEmail, ttl: ttl, codeLength: codeLength, gen: gen}
}

func (s *identifierKeyedStrategy) Type() Type { return s.otpType }

func (s *identifierKeyedStrategy) Key(identifier, _ string) string {
	return fmt.Sprintf("otp:%s:%s", s.otpType, identifier)
}

func (s *identifierKeyedStrategy) KeyWithID(identifier string) string {
	return fmt.Sprintf("otp:%s:%s", s.otpType, identifier)
}

func (s *identifierKeyedStrategy) DefaultTTL() time.Duration { return s.ttl }

func (s *identifierKeyedStrategy) GenerateCode() (string, error) {
	return s.gen.Numeric(s.codeLength)
}

func (s *identifierKeyedStrategy) EmptyPayload() Payload { return &SimplePayload{} }

func (s *identifierKeyedStrategy) NewPayload(code string, _ map[string]any) (Payload, error) {
	return &SimplePayload{OTPCode: code}, nil
}

// ////////////////////////////////////////

type linkgroupchatcompanyStrategy struct {
	otpType    Type
	ttl        time.Duration
	codeLength int
	gen        CodeGenerator
}

// NewPhoneStrategy builds the Strategy for otp:phone:{phone} OTPs.

func NewLinkGroupChatCompanyStrategy(ttl time.Duration, codeLength int, gen CodeGenerator) Strategy {
	return &linkgroupchatcompanyStrategy{otpType: TypeLink, ttl: ttl, codeLength: codeLength, gen: gen}
}
func (s *linkgroupchatcompanyStrategy) Type() Type { return s.otpType }

func (s *linkgroupchatcompanyStrategy) Key(_, code string) string {
	return fmt.Sprintf("otp:%s:%s:%s", s.otpType, "group", code)
}

func (s *linkgroupchatcompanyStrategy) KeyWithID(identifier string) string {
	return fmt.Sprintf("otp:%s:%s:%s", s.otpType, "group", identifier)
}

func (s *linkgroupchatcompanyStrategy) DefaultTTL() time.Duration { return s.ttl }

func (s *linkgroupchatcompanyStrategy) GenerateCode() (string, error) {
	return s.gen.RandBase58String(s.codeLength)
}

func (s *linkgroupchatcompanyStrategy) EmptyPayload() Payload { return &LinkChatCompanyPayload{} }

func (s *linkgroupchatcompanyStrategy) NewPayload(code string, metadatas map[string]any) (Payload, error) {
	return &LinkChatCompanyPayload{OTPCode: code, CompanyID: metadatas["company_id"].(string)}, nil
}

// ////////////////////////////////////////

// ////////////////////////////////////////

type linkchannelchatcompanyStrategy struct {
	otpType    Type
	ttl        time.Duration
	codeLength int
	gen        CodeGenerator
}

// NewPhoneStrategy builds the Strategy for otp:phone:{phone} OTPs.

func NewLinkChannelChatCompanyStrategy(ttl time.Duration, codeLength int, gen CodeGenerator) Strategy {
	return &linkchannelchatcompanyStrategy{otpType: TypeLinkJustChnnel, ttl: ttl, codeLength: codeLength , gen: gen}
}
func (s *linkchannelchatcompanyStrategy) Type() Type { return s.otpType }

func (s *linkchannelchatcompanyStrategy) Key(_, code string) string {
	return fmt.Sprintf("otp:%s:%s:%s", s.otpType, "channel", code)
}

func (s *linkchannelchatcompanyStrategy) KeyWithID(identifier string) string {
	return fmt.Sprintf("otp:%s:%s:%s", s.otpType, "channel", identifier)
}

func (s *linkchannelchatcompanyStrategy) DefaultTTL() time.Duration { return s.ttl }

func (s *linkchannelchatcompanyStrategy) GenerateCode() (string, error) {
	return s.gen.RandBase58String(s.codeLength)
}

func (s *linkchannelchatcompanyStrategy) EmptyPayload() Payload { return &LinkChatCompanyPayload{} }

func (s *linkchannelchatcompanyStrategy) NewPayload(code string, metadatas map[string]any) (Payload, error) {
	return &LinkChatCompanyPayload{OTPCode: code, CompanyID: metadatas["company_id"].(string)}, nil
}

// ////////////////////////////////////////

type registerUserStrategy struct {
	otpType    Type
	ttl        time.Duration
	codeLength int
	gen        CodeGenerator
}

// NewPhoneStrategy builds the Strategy for otp:phone:{phone} OTPs.
func NewRegisterUserStrategy(ttl time.Duration, codeLength int, gen CodeGenerator) Strategy {
	return &registerUserStrategy{otpType: TypeRegistrionUserToCompany, ttl: ttl, codeLength: codeLength, gen: gen}
}
func (s *registerUserStrategy) Type() Type { return s.otpType }

func (s *registerUserStrategy) Key(_, code string) string {
	return fmt.Sprintf("otp:%s:%s", s.otpType, code)
}

func (s *registerUserStrategy) KeyWithID(identifier string) string {
	return fmt.Sprintf("otp:%s:%s", s.otpType, identifier)
}

func (s *registerUserStrategy) DefaultTTL() time.Duration { return s.ttl }

func (s *registerUserStrategy) GenerateCode() (string, error) {
	return s.gen.RandBase58String(s.codeLength)
}

func (s *registerUserStrategy) EmptyPayload() Payload { return &RegisterUserPayload{} }

func (s *registerUserStrategy) NewPayload(code string, metadatas map[string]any) (Payload, error) {
	return &RegisterUserPayload{OTPCode: code, Roles: metadatas["roles"].([]string), CompanyID: metadatas["company_id"].(string)}, nil
}

// toUint accepts the numeric types a caller realistically passes for an
// ID: a plain uint from Go code calling the service directly, or a
// float64 from a map decoded out of a JSON request body.
func toUint(raw any) (uint, error) {
	switch v := raw.(type) {
	case uint:
		return v, nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("must be non-negative, got %d", v)
		}
		return uint(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("must be non-negative, got %v", v)
		}
		return uint(v), nil
	default:
		return 0, fmt.Errorf("must be numeric, got %T", raw)
	}
}
