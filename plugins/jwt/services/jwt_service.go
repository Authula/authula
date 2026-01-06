package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/GoBetterAuth/go-better-auth/services"
)

// JWTServiceImpl is the concrete implementation of the JWTService interface
type JWTServiceImpl struct {
	cacheService     CacheService
	blacklistService BlacklistService
	sessionService   services.SessionService
}

// NewJWTService creates a new JWT service implementation
func NewJWTService(
	cacheService CacheService,
	blacklistService BlacklistService,
	sessionService services.SessionService,
) services.JWTService {
	return &JWTServiceImpl{
		cacheService:     cacheService,
		blacklistService: blacklistService,
		sessionService:   sessionService,
	}
}

// ValidateToken validates a JWT token and ensures the referenced session is still active
func (s *JWTServiceImpl) ValidateToken(token string) (userID string, err error) {
	jwkSet, err := s.cacheService.GetJWKSWithFallback(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get JWKS: %w", err)
	}

	parsedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(jwkSet), jwt.WithValidate(true))
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	jti, ok := parsedToken.JwtID()
	if ok && jti != "" && s.blacklistService != nil {
		isBlacklisted, err := s.blacklistService.IsBlacklisted(context.Background(), jti)
		if err != nil {
			// Don't fail validation on blacklist check error, but continue
		} else if isBlacklisted {
			return "", errors.New("token has been revoked")
		}
	}

	var tokenType string
	if err := parsedToken.Get("type", &tokenType); err != nil {
		return "", errors.New("missing token type claim")
	}

	if tokenType != "access" {
		return "", errors.New("invalid token type")
	}

	var extractedUserID string
	if err := parsedToken.Get("user_id", &extractedUserID); err != nil {
		return "", errors.New("missing user_id claim")
	}

	if extractedUserID == "" {
		return "", errors.New("missing user_id claim")
	}

	var sessionID string
	if err := parsedToken.Get("session_id", &sessionID); err != nil {
		return "", errors.New("missing session_id claim")
	}

	if sessionID == "" {
		return "", errors.New("missing session_id claim")
	}

	if s.blacklistService != nil {
		isBlacklisted, err := s.blacklistService.IsBlacklisted(context.Background(), "session:"+sessionID)
		if err != nil {
			// Don't fail validation on blacklist check error, but continue
		} else if isBlacklisted {
			return "", errors.New("session has been revoked")
		}
	}

	// Ensure the session is still active
	session, err := s.sessionService.GetByID(context.Background(), sessionID)
	if err != nil || session == nil {
		return "", errors.New("session not found or invalid")
	}

	return extractedUserID, nil
}
