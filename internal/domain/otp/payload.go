package otp

// SimplePayload is the payload for identifier-keyed OTP types (phone,
// email, and any future type that needs nothing beyond the code
// itself).
type SimplePayload struct {
	OTPCode string `json:"code"`
}

func (p *SimplePayload) Code() string { return p.OTPCode }

// LinkPayload is the payload for a magic-link OTP. Unlike phone/email, a
// link OTP is looked up by its code alone once the link is opened (see
// linkStrategy.Key) -- there's no separate identifier available at
// verify time. So whatever the flow needs to resume -- which company a
// chat should be linked to, which chat it is -- has to travel with the
// code as metadata rather than being re-derivable from an identifier.
// CompanyID and ChatID are what today's link-chat flow needs; a future
// link-based flow that needs different metadata should define its own
// payload type via a new Strategy rather than growing this one.
type RegisterUserPayload struct {
	OTPCode   string `json:"code"`
	Roles     []string
	CompanyID string
}

func (p *RegisterUserPayload) Code() string { return p.OTPCode }

type LinkChatCompanyPayload struct {
	OTPCode   string `json:"code"`
	CompanyID string `json:"company_id"`
}

func (p *LinkChatCompanyPayload) Code() string { return p.OTPCode }
