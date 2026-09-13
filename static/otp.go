package static

import (
	"bytes"
)

func (sf *StaticFiles) ExecuteOTPTemplate(otpCode string, ttl string) (*bytes.Buffer, error) {
	var htmlBuffer bytes.Buffer
	if err := sf.otpHTMLTemplate.Execute(&htmlBuffer, map[string]any{
		"OTPCode": otpCode,
		"TTL":     ttl,
	}); err != nil {
		return nil, err
	}
	return &htmlBuffer, nil
}
