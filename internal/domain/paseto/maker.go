package paseto

import (
	"errors"
	"fmt"
	"time"

	gopaseto "aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken is returned when a token fails to parse or decrypt.
	ErrInvalidToken = errors.New("token is invalid")
	// ErrExpiredToken is returned when a token parses correctly but its
	// validity window has passed.
	ErrExpiredToken = errors.New("token has expired")
)

// Maker defines the behaviour required to issue and verify PASETO tokens.
// It is deliberately narrow so the concrete crypto library stays an
// implementation detail of this package.
type Maker interface {
	CreateToken(userID, companyID, username string, tokenType TokenType, duration time.Duration) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}

// PasetoMaker issues and verifies PASETO v4.local (symmetric-key encrypted)
// tokens. v4.local is used -- rather than v4.public -- because a single
// backend both mints and verifies its own tokens, so there is no need to
// distribute a public key to a third-party verifier.
type PasetoMaker struct {
	symmetricKey gopaseto.V4SymmetricKey
	parser       gopaseto.Parser
}

// NewPasetoMaker builds a Maker from a 32-byte, hex-encoded symmetric key
// (see PASETO_SYMMETRIC_KEY in the environment configuration). Generate one
// with `openssl rand -hex 32`.
func NewPasetoMaker(hexKey string) (Maker, error) {
	key, err := gopaseto.V4SymmetricKeyFromHex(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid paseto symmetric key: %w", err)
	}

	return &PasetoMaker{
		symmetricKey: key,
		parser:       gopaseto.NewParser(), // NotExpired rule is enabled by default
	}, nil
}

func (m *PasetoMaker) CreateToken(userID, companyID, username string, tokenType TokenType, duration time.Duration) (string, *Payload, error) {
	now := time.Now()
	// ? we can use v7 (after enabale pool random) but we are fine for now (espically for token payloads!)
	payload := &Payload{
		ID:        uuid.New(),
		UserID:    userID,
		CompanyID: companyID,
		Username:  username,
		TokenType: tokenType,
		IssuedAt:  now,
		ExpiredAt: now.Add(duration),
	}

	token := gopaseto.NewToken()
	token.SetIssuedAt(payload.IssuedAt)
	token.SetNotBefore(payload.IssuedAt)
	token.SetExpiration(payload.ExpiredAt)
	token.SetJti(payload.ID.String())
	token.SetString("user_id", payload.UserID)
	token.SetString("company_id", payload.CompanyID)
	token.SetString("username", payload.Username)
	token.SetString("token_type", string(payload.TokenType))

	encrypted := token.V4Encrypt(m.symmetricKey, nil)

	return encrypted, payload, nil
}

func (m *PasetoMaker) VerifyToken(tokenString string) (*Payload, error) {
	token, err := m.parser.ParseV4Local(m.symmetricKey, tokenString, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	jti, err := token.GetJti()
	if err != nil {
		return nil, ErrInvalidToken
	}
	id, err := uuid.Parse(jti)
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := token.GetString("user_id")
	if err != nil {
		return nil, ErrInvalidToken
	}
	companyID, err := token.GetString("company_id")
	if err != nil {
		return nil, ErrInvalidToken
	}
	username, err := token.GetString("username")
	if err != nil {
		return nil, ErrInvalidToken
	}
	tokenType, err := token.GetString("token_type")
	if err != nil {
		return nil, ErrInvalidToken
	}
	issuedAt, err := token.GetIssuedAt()
	if err != nil {
		return nil, ErrInvalidToken
	}
	expiredAt, err := token.GetExpiration()
	if err != nil {
		return nil, ErrInvalidToken
	}

	payload := &Payload{
		ID:        id,
		UserID:    userID,
		CompanyID: companyID,
		Username:  username,
		TokenType: TokenType(tokenType),
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}

	// Belt-and-braces: the parser already enforces NotExpired, but we
	// re-check explicitly since callers depend on ErrExpiredToken being
	// distinguishable from a structurally invalid token.
	if payload.IsExpired() {
		return nil, ErrExpiredToken
	}

	return payload, nil
}
