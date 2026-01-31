package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// JWTServiceImpl is the concrete implementation of the JWTService interface
type JWTServiceImpl struct {
	logger           models.Logger
	tokenService     services.TokenService
	keyService       KeyService
	cacheService     CacheService
	blacklistService BlacklistService
	sessionService   services.SessionService
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
	algorithm        types.JWTAlgorithm
}

// NewJWTService creates a new JWT service implementation
func NewJWTService(
	logger models.Logger,
	sessionService services.SessionService,
	tokenService services.TokenService,
	keyService KeyService,
	cacheService CacheService,
	blacklistService BlacklistService,
	expiresIn time.Duration,
	refreshExpiresIn time.Duration,
) services.JWTService {
	return &JWTServiceImpl{
		logger:           logger,
		sessionService:   sessionService,
		tokenService:     tokenService,
		keyService:       keyService,
		cacheService:     cacheService,
		blacklistService: blacklistService,
		expiresIn:        expiresIn,
		refreshExpiresIn: refreshExpiresIn,
	}
}

// GenerateTokens creates access and refresh JWT tokens tied to a session
func (s *JWTServiceImpl) GenerateTokens(ctx context.Context, userID string, sessionID string) (*types.TokenPair, error) {
	if sessionID == "" {
		return nil, errors.New("session id is required to generate tokens")
	}

	jwksKey, err := s.keyService.GetActiveKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	privateKeyPEM, err := s.tokenService.Decrypt(jwksKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	privKey, err := jwk.ParseKey([]byte(privateKeyPEM), jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Set the Key ID (kid) on the key so it's included in the JWT header
	if err := privKey.Set(jwk.KeyIDKey, jwksKey.ID); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}

	keyAlgorithm := s.detectAlgorithmFromKey(privKey)

	now := time.Now()
	jti := uuid.New().String()

	accessClaims := jwt.New()
	accessClaims.Set(jwt.SubjectKey, userID)
	accessClaims.Set(jwt.IssuedAtKey, now)
	accessClaims.Set(jwt.ExpirationKey, now.Add(s.expiresIn))
	accessClaims.Set(jwt.JwtIDKey, jti)
	accessClaims.Set("user_id", userID)
	accessClaims.Set("session_id", sessionID)
	accessClaims.Set("type", types.JWTTokenTypeAccess.String())

	accessTokenBytes, err := jwt.Sign(accessClaims, jwt.WithKey(keyAlgorithm, privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := jwt.New()
	refreshClaims.Set(jwt.SubjectKey, userID)
	refreshClaims.Set(jwt.IssuedAtKey, now)
	refreshClaims.Set(jwt.ExpirationKey, now.Add(s.refreshExpiresIn))
	refreshClaims.Set(jwt.JwtIDKey, jti)
	refreshClaims.Set("user_id", userID)
	refreshClaims.Set("session_id", sessionID)
	refreshClaims.Set("type", types.JWTTokenTypeRefresh.String())

	refreshTokenBytes, err := jwt.Sign(refreshClaims, jwt.WithKey(keyAlgorithm, privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &types.TokenPair{
		AccessToken:  string(accessTokenBytes),
		RefreshToken: string(refreshTokenBytes),
		ExpiresIn:    s.expiresIn,
		TokenType:    "Bearer",
	}, nil
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
		slog.Debug("parsedToken", "token", parsedToken)
		return "", errors.New("missing token type claim")
	}

	if tokenType != types.JWTTokenTypeAccess.String() {
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

func (s *JWTServiceImpl) detectAlgorithmFromKey(k jwk.Key) jwa.SignatureAlgorithm {
	if alg, ok := k.Algorithm(); ok {
		if sigAlg, ok := alg.(jwa.SignatureAlgorithm); ok {
			return sigAlg
		}
	}

	keyType := k.KeyType().String()
	var detectedAlg jwa.SignatureAlgorithm
	switch keyType {
	case "OKP":
		detectedAlg = jwa.EdDSA()
	case "RSA":
		detectedAlg = jwa.RS256()
	case "EC":
		detectedAlg = jwa.ES256()
	case "oct":
		detectedAlg = jwa.HS256()
	default:
		detectedAlg = jwa.EdDSA()
	}

	return detectedAlg
}
