package service_contract

import (
	"context"
)

type EmailService interface {
	SendEmail(ctx context.Context, req SendEmailRequest) (*bool, error)
	SendOTPEmail(ctx context.Context, req SendOTPEmailRequest) (*bool, error)
}

type SendEmailRequest struct {
	To          []string `json:"to" binding:"required,email"`
	Subject     string   `json:"subject" binding:"required"`
	TextBody    string   `json:"text_body,omitempty"`
	HTMLBody    string   `json:"html_body,omitempty"`
	EmbedFiles  []string `json:"embed_files,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type SendOTPEmailRequest struct {
	To         []string `json:"to" binding:"required,email"`
	OTP        string
	TTL string
}
