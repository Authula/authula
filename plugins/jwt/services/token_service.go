package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
	"github.com/Authula/authula/services"
)

type tokenService struct {
	logger           models.Logger
	coreTokenService services.TokenService
	keyService       KeyService
	cacheService     CacheService
	blacklistService BlacklistService
	sessionService   services.SessionService
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
}

func NewJWTService(
	logger models.Logger,
	sessionService services.SessionService,
	coreTokenService services.TokenService,
	keyService KeyService,
	cacheService CacheService,
	blacklistService BlacklistService,
	expiresIn time.Duration,
	refreshExpiresIn time.Duration,
) services.JWTService {
	return &tokenService{
		logger:           logger,
		sessionService:   sessionService,
		coreTokenService: coreTokenService,
		keyService:       keyService,
		cacheService:     cacheService,
		blacklistService: blacklistService,
		expiresIn:        expiresIn,
		refreshExpiresIn: refreshExpiresIn,
	}
}

func (s *tokenService) detectAlgorithmFromKey(k jwk.Key) jwa.SignatureAlgorithm {
	return jwa.EdDSA()
}

func (s *tokenService) GenerateUserToken(ctx context.Context, userID string, sessionID string) (*types.TokenPair, error) {
	if sessionID == "" {
		return nil, errors.New("session id is required to generate tokens")
	}

	jwksKey, err := s.keyService.GetActiveKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	privateKeyPEM, err := s.coreTokenService.Decrypt(jwksKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	privKey, err := jwk.ParseKey([]byte(privateKeyPEM), jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	if err := privKey.Set(jwk.KeyIDKey, jwksKey.ID); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}

	keyAlgorithm := s.detectAlgorithmFromKey(privKey)
	now := time.Now()
	jti := uuid.New().String()

	accessClaims := jwt.New()
	if err := accessClaims.Set(jwt.SubjectKey, userID); err != nil {
		return nil, fmt.Errorf("failed to set subject: %w", err)
	}
	if err := accessClaims.Set(jwt.IssuedAtKey, now); err != nil {
		return nil, fmt.Errorf("failed to set issued at: %w", err)
	}
	if err := accessClaims.Set(jwt.ExpirationKey, now.Add(s.expiresIn)); err != nil {
		return nil, fmt.Errorf("failed to set expiration: %w", err)
	}
	if err := accessClaims.Set(jwt.JwtIDKey, jti); err != nil {
		return nil, fmt.Errorf("failed to set JWT ID: %w", err)
	}
	if err := accessClaims.Set("user_id", userID); err != nil {
		return nil, fmt.Errorf("failed to set user_id: %w", err)
	}
	if err := accessClaims.Set("session_id", sessionID); err != nil {
		return nil, fmt.Errorf("failed to set session_id: %w", err)
	}
	if err := accessClaims.Set("token_type", types.JWTTokenTypeAccess.String()); err != nil {
		return nil, fmt.Errorf("failed to set token_type: %w", err)
	}
	if err := accessClaims.Set("actor_type", "user"); err != nil {
		return nil, fmt.Errorf("failed to set actor_type: %w", err)
	}

	accessTokenBytes, err := jwt.Sign(accessClaims, jwt.WithKey(keyAlgorithm, privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := jwt.New()
	if err := refreshClaims.Set(jwt.SubjectKey, userID); err != nil {
		return nil, fmt.Errorf("failed to set subject in refresh token: %w", err)
	}
	if err := refreshClaims.Set(jwt.IssuedAtKey, now); err != nil {
		return nil, fmt.Errorf("failed to set issued at in refresh token: %w", err)
	}
	if err := refreshClaims.Set(jwt.ExpirationKey, now.Add(s.refreshExpiresIn)); err != nil {
		return nil, fmt.Errorf("failed to set expiration in refresh token: %w", err)
	}
	if err := refreshClaims.Set(jwt.JwtIDKey, jti); err != nil {
		return nil, fmt.Errorf("failed to set JWT ID in refresh token: %w", err)
	}
	if err := refreshClaims.Set("user_id", userID); err != nil {
		return nil, fmt.Errorf("failed to set user_id in refresh token: %w", err)
	}
	if err := refreshClaims.Set("session_id", sessionID); err != nil {
		return nil, fmt.Errorf("failed to set session_id in refresh token: %w", err)
	}
	if err := refreshClaims.Set("token_type", types.JWTTokenTypeRefresh.String()); err != nil {
		return nil, fmt.Errorf("failed to set token_type in refresh token: %w", err)
	}
	if err := refreshClaims.Set("actor_type", "user"); err != nil {
		return nil, fmt.Errorf("failed to set actor_type in refresh token: %w", err)
	}

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

func (s *tokenService) GenerateMachineToken(ctx context.Context, clientID string, organizationID string, scopes []string) (*types.TokenPair, error) {
	jwksKey, err := s.keyService.GetActiveKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	privateKeyPEM, err := s.coreTokenService.Decrypt(jwksKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	privKey, err := jwk.ParseKey([]byte(privateKeyPEM), jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	if err := privKey.Set(jwk.KeyIDKey, jwksKey.ID); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}

	keyAlgorithm := s.detectAlgorithmFromKey(privKey)
	now := time.Now()
	jti := uuid.New().String()

	accessClaims := jwt.New()
	if err := accessClaims.Set(jwt.SubjectKey, clientID); err != nil {
		return nil, fmt.Errorf("failed to set subject: %w", err)
	}
	if err := accessClaims.Set(jwt.IssuedAtKey, now); err != nil {
		return nil, fmt.Errorf("failed to set issued at: %w", err)
	}
	if err := accessClaims.Set(jwt.ExpirationKey, now.Add(s.expiresIn)); err != nil {
		return nil, fmt.Errorf("failed to set expiration: %w", err)
	}
	if err := accessClaims.Set(jwt.JwtIDKey, jti); err != nil {
		return nil, fmt.Errorf("failed to set JWT ID: %w", err)
	}
	if err := accessClaims.Set("token_type", types.JWTTokenTypeAccess.String()); err != nil {
		return nil, fmt.Errorf("failed to set token_type: %w", err)
	}
	if err := accessClaims.Set("actor_type", "machine"); err != nil {
		return nil, fmt.Errorf("failed to set actor_type: %w", err)
	}

	if organizationID != "" {
		if err := accessClaims.Set("org_id", organizationID); err != nil {
			return nil, fmt.Errorf("failed to set org_id: %w", err)
		}
	}

	if len(scopes) > 0 {
		if err := accessClaims.Set("scopes", scopes); err != nil {
			return nil, fmt.Errorf("failed to set scopes: %w", err)
		}
	}

	accessTokenBytes, err := jwt.Sign(accessClaims, jwt.WithKey(keyAlgorithm, privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return &types.TokenPair{
		AccessToken:  string(accessTokenBytes),
		RefreshToken: "",
		ExpiresIn:    s.expiresIn,
		TokenType:    "Bearer",
	}, nil
}

func (s *tokenService) ValidateToken(ctx context.Context, token string) (*models.Actor, error) {
	jwkSet, err := s.cacheService.GetJWKSWithFallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	parsedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(jwkSet), jwt.WithValidate(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	jti, ok := parsedToken.JwtID()
	if ok && jti != "" && s.blacklistService != nil {
		isBlacklisted, err := s.blacklistService.IsBlacklisted(ctx, jti)
		if err == nil && isBlacklisted {
			return nil, errors.New("token has been revoked")
		}
	}

	var tokenType string
	if err := parsedToken.Get("token_type", &tokenType); err != nil {
		return nil, errors.New("missing token_type claim")
	}

	if tokenType != types.JWTTokenTypeAccess.String() {
		return nil, errors.New("invalid token_type")
	}

	var actorType string
	if err := parsedToken.Get("actor_type", &actorType); err != nil || actorType == "" {
		actorType = "user"
	}

	actor := &models.Actor{
		Claims: map[string]any{"auth_mechanism": "jwt_bearer"},
	}

	if actorType == "machine" {
		var sub string
		if err := parsedToken.Get(jwt.SubjectKey, &sub); err != nil || sub == "" {
			return nil, errors.New("missing subject claim")
		}
		actor.ID = sub
		actor.Type = models.ActorMachine

		var orgID string
		if err := parsedToken.Get("org_id", &orgID); err == nil && orgID != "" {
			actor.Claims["organization_id"] = orgID
		}

		var raw any
		if err := parsedToken.Get("scopes", &raw); err == nil {
			switch v := raw.(type) {
			case []string:
				if len(v) > 0 {
					actor.Scopes = v
				}
			case []any:
				scopes := make([]string, 0, len(v))
				for _, s := range v {
					if str, ok := s.(string); ok {
						scopes = append(scopes, str)
					}
				}
				if len(scopes) > 0 {
					actor.Scopes = scopes
				}
			}
		}

		return actor, nil
	}

	var userID string
	if err := parsedToken.Get("user_id", &userID); err != nil || userID == "" {
		return nil, errors.New("missing user_id claim")
	}

	var sessionID string
	if err := parsedToken.Get("session_id", &sessionID); err != nil || sessionID == "" {
		return nil, errors.New("missing session_id claim")
	}

	if s.blacklistService != nil {
		isBlacklisted, err := s.blacklistService.IsBlacklisted(ctx, "session:"+sessionID)
		if err == nil && isBlacklisted {
			return nil, errors.New("session has been revoked")
		}
	}

	session, err := s.sessionService.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return nil, errors.New("session not found or invalid")
	}

	actor.ID = userID
	actor.Type = models.ActorUser

	return actor, nil
}
