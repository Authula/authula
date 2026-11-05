package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
	"github.com/Authula/authula/services"
)

var reservedTokenClaims = map[string]struct{}{
	jwt.IssuerKey:     {},
	jwt.AudienceKey:   {},
	jwt.SubjectKey:    {},
	jwt.ExpirationKey: {},
	jwt.IssuedAtKey:   {},
	jwt.NotBeforeKey:  {},
	jwt.JwtIDKey:      {},
	"token_type":      {},
	"actor_type":      {},
	"user_id":         {},
	"session_id":      {},
	"client_id":       {},
}

type tokenService struct {
	logger           models.Logger
	coreTokenService services.TokenService
	keyService       KeyService
	cacheService     CacheService
	blacklistService BlacklistService
	sessionService   services.SessionService
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
	issuer           string
	audience         string
}

func NewJWTService(
	logger models.Logger,
	sessionService services.SessionService,
	coreTokenService services.TokenService,
	keyService KeyService,
	cacheService CacheService,
	blacklistService BlacklistService,
	config TokenServiceConfig,
) services.JWTService {
	return &tokenService{
		logger:           logger,
		sessionService:   sessionService,
		coreTokenService: coreTokenService,
		keyService:       keyService,
		cacheService:     cacheService,
		blacklistService: blacklistService,
		expiresIn:        config.ExpiresIn,
		refreshExpiresIn: config.RefreshExpiresIn,
		issuer:           config.Issuer,
		audience:         config.Audience,
	}
}

func (s *tokenService) detectAlgorithmFromKey() jwa.SignatureAlgorithm {
	return jwa.EdDSA()
}

func (s *tokenService) GenerateUserToken(ctx context.Context, userID string, sessionID string, extraClaims map[string]any) (*types.TokenPair, error) {
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

	keyAlgorithm := s.detectAlgorithmFromKey()
	now := time.Now()
	accessJTI := uuid.New().String()
	refreshJTI := uuid.New().String()

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
	if err := accessClaims.Set(jwt.JwtIDKey, accessJTI); err != nil {
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
	if err := accessClaims.Set("actor_type", string(models.ActorUser)); err != nil {
		return nil, fmt.Errorf("failed to set actor_type: %w", err)
	}

	if err := s.applyExtraClaims(accessClaims, extraClaims); err != nil {
		return nil, err
	}
	if err := s.setIssuerAndAudience(accessClaims); err != nil {
		return nil, err
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
	if err := refreshClaims.Set(jwt.JwtIDKey, refreshJTI); err != nil {
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
	if err := refreshClaims.Set("actor_type", string(models.ActorUser)); err != nil {
		return nil, fmt.Errorf("failed to set actor_type in refresh token: %w", err)
	}

	if err := s.applyExtraClaims(refreshClaims, extraClaims); err != nil {
		return nil, err
	}
	if err := s.setIssuerAndAudience(refreshClaims); err != nil {
		return nil, err
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

	keyAlgorithm := s.detectAlgorithmFromKey()
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
	if err := accessClaims.Set("actor_type", string(models.ActorMachine)); err != nil {
		return nil, fmt.Errorf("failed to set actor_type: %w", err)
	}
	if err := accessClaims.Set("client_id", clientID); err != nil {
		return nil, fmt.Errorf("failed to set client_id: %w", err)
	}

	if organizationID != "" {
		if err := accessClaims.Set("organization_id", organizationID); err != nil {
			return nil, fmt.Errorf("failed to set organization_id: %w", err)
		}
	}

	if len(scopes) > 0 {
		if err := accessClaims.Set("scopes", scopes); err != nil {
			return nil, fmt.Errorf("failed to set scopes: %w", err)
		}
	}

	if err := s.setIssuerAndAudience(accessClaims); err != nil {
		return nil, err
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

	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(jwkSet),
		jwt.WithValidate(true),
		jwt.WithValidator(s.accessTokenValidator()),
	)
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

	var actorType string
	if err := parsedToken.Get("actor_type", &actorType); err != nil || actorType == "" {
		return nil, errors.New("missing actor_type claim")
	}

	switch models.ActorType(actorType) {
	case models.ActorUser, models.ActorMachine:
	default:
		return nil, errors.New("invalid actor_type claim")
	}

	actor := &models.Actor{
		Type:   models.ActorType(actorType),
		Claims: map[string]any{"auth_mechanism": "jwt_bearer"},
	}

	if actorType == models.ActorMachine.ToString() {
		var sub string
		if err := parsedToken.Get(jwt.SubjectKey, &sub); err != nil || sub == "" {
			return nil, errors.New("missing subject claim")
		}
		actor.ID = sub
		actor.Type = models.ActorMachine

		var orgID string
		if err := parsedToken.Get("organization_id", &orgID); err == nil && orgID != "" {
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
	if !session.ExpiresAt.After(time.Now()) {
		return nil, errors.New("session expired or invalid")
	}

	actor.Claims["session_id"] = session.ID
	actor.ID = userID
	actor.Type = models.ActorUser

	// Passthrough any extra claims from the token into Actor.Claims.
	for _, key := range parsedToken.Keys() {
		switch key {
		case jwt.SubjectKey, jwt.IssuedAtKey, jwt.ExpirationKey, jwt.JwtIDKey,
			jwt.IssuerKey, jwt.AudienceKey, "user_id", "session_id", "token_type",
			"actor_type", "organization_id", "scopes", "client_id":
			continue
		default:
			var val any
			if err := parsedToken.Get(key, &val); err == nil {
				actor.Claims[key] = val
			}
		}
	}

	return actor, nil
}

func (s *tokenService) ValidateRefreshToken(ctx context.Context, token string) (map[string]any, error) {
	jwkSet, err := s.cacheService.GetJWKSWithFallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(jwkSet),
		jwt.WithValidate(true),
		// Refresh expiry and rotation state are enforced against the stored
		// record so revoked-token reuse is still detected after expiry.
		jwt.WithResetValidators(true),
		jwt.WithValidator(s.refreshTokenValidator()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims := make(map[string]any, len(parsedToken.Keys()))
	for _, key := range parsedToken.Keys() {
		var val any
		if err := parsedToken.Get(key, &val); err == nil {
			claims[key] = val
		}
	}
	return claims, nil
}

func (s *tokenService) accessTokenValidator() jwt.ValidatorFunc {
	return func(_ context.Context, token jwt.Token) error {
		if err := s.validateTokenPolicy(token, types.JWTTokenTypeAccess); err != nil {
			return err
		}

		var actorType string
		if err := token.Get("actor_type", &actorType); err != nil || actorType == "" {
			return errors.New("missing actor_type claim")
		}
		switch models.ActorType(actorType) {
		case models.ActorUser, models.ActorMachine:
			return nil
		default:
			return errors.New("invalid actor_type claim")
		}
	}
}

func (s *tokenService) refreshTokenValidator() jwt.ValidatorFunc {
	return func(_ context.Context, token jwt.Token) error {
		return s.validateTokenPolicy(token, types.JWTTokenTypeRefresh)
	}
}

func (s *tokenService) validateTokenPolicy(token jwt.Token, expectedTokenType types.JWTTokenType) error {
	issuer, ok := token.Issuer()
	if !ok || issuer == "" {
		return errors.New("missing issuer claim")
	}
	if issuer != s.issuer {
		return errors.New("invalid issuer claim")
	}

	audiences, ok := token.Audience()
	if !ok || len(audiences) == 0 {
		return errors.New("missing audience claim")
	}
	if !slices.Contains(audiences, s.audience) {
		return errors.New("invalid audience claim")
	}

	var tokenType string
	if err := token.Get("token_type", &tokenType); err != nil || tokenType == "" {
		return errors.New("missing token_type claim")
	}
	if tokenType != expectedTokenType.String() {
		return errors.New("invalid token_type")
	}

	return nil
}

func (s *tokenService) applyExtraClaims(token jwt.Token, extraClaims map[string]any) error {
	for key, value := range extraClaims {
		if _, reserved := reservedTokenClaims[key]; reserved {
			s.logger.Debug("ignoring reserved JWT claim from extra claims", "claim", key)
			continue
		}
		if err := token.Set(key, value); err != nil {
			return fmt.Errorf("failed to set extra claim %s: %w", key, err)
		}
	}
	return nil
}

func (s *tokenService) setIssuerAndAudience(token jwt.Token) error {
	if err := token.Set(jwt.IssuerKey, s.issuer); err != nil {
		return fmt.Errorf("failed to set issuer: %w", err)
	}
	if err := token.Set(jwt.AudienceKey, []string{s.audience}); err != nil {
		return fmt.Errorf("failed to set audience: %w", err)
	}
	return nil
}
