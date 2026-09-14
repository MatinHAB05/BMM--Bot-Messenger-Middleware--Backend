package apimiddleware

import (
	"errors"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/paseto"
	repository_contract "messenger-backend/internal/domain/repository"
	"messenger-backend/internal/domain/tokencontext"
	"messenger-backend/pkg/logger"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthnMiddleware gin.HandlerFunc

const (
	authorizationHeaderKey  = "Authorization"
	authorizationTypeBearer = "bearer"
)

// Auth verifies the PASETO access token on every protected request:
//  1. it must parse and decrypt successfully against our symmetric key,
//  2. it must not be expired,
//  3. it must be a TokenType of "access" (a refresh token must not be
//     usable to authenticate a normal API request), and
//  4. its jti must NOT be present in the Redis revocation blacklist
//     (blacklist:access_token:<jti>), which POST /auth/logout populates.
//
// On success, the verified payload is attached to the Gin context for
// downstream middleware (RBAC) and handlers to read via tokencontext.
func Auth(tokenMaker paseto.Maker, tokenBlacklistRepo repository_contract.AuthnTokenRepository, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(authorizationHeaderKey)
		if header == "" {
			respondError(c, exception.ErrMissingToken)
			return
		}

		fields := strings.Fields(header)
		if len(fields) != 2 || !strings.EqualFold(fields[0], authorizationTypeBearer) {
			respondError(c, exception.ErrMissingToken)
			return
		}
		tokenString := fields[1]

		payload, err := tokenMaker.VerifyToken(tokenString)
		if err != nil {
			if errors.Is(err, paseto.ErrExpiredToken) {
				respondError(c, exception.ErrTokenExpired)
				return
			}
			respondError(c, exception.ErrTokenInvalid)
			return
		}

		if payload.TokenType != paseto.AccessToken {
			respondError(c, exception.ErrTokenInvalid)
			return
		}

		isBlacklisted, err := tokenBlacklistRepo.IsBlacklisted(c.Request.Context(), payload.ID.String())
		if err != nil {
			log.Error(err, "failed to check token blacklist", logger.String("jti", payload.ID.String()))
			respondError(c, exception.ErrInternal)
			return
		}
		if isBlacklisted {
			respondError(c, exception.ErrTokenRevoked)
			return
		}

		tokencontext.SetPayload(c, payload)
		c.Next()
	}
}
