package paseto

import (
	"time"

	"github.com/google/uuid"
)

// TokenType distinguishes access tokens from refresh tokens so the same
// Maker/Payload machinery can issue and verify both.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Payload is the decrypted content of a PASETO token.
type Payload struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	CompanyID string    `json:"company_id"`
	Username  string    `json:"username"`
	TokenType TokenType `json:"token_type"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

func (p *Payload) IsExpired() bool {
	return time.Now().After(p.ExpiredAt)
}
