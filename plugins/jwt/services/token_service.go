package services

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
	oauth2types "github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/services"
)

var _ services.ClientCredentialsTokenIssuer = (*tokenService)(nil)

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
	logger              models.Logger
	coreTokenService    services.TokenService
	keyService          KeyService
	cacheService        CacheService
	sessionService      services.SessionService
	expiresIn           time.Duration
	refreshExpiresIn    time.Duration
	issuer              string
	audience            string
	accessTokenClientID string
}

func NewJWTService(
	logger models.Logger,
	sessionService services.SessionService,
	coreTokenService services.TokenService,
	keyService KeyService,
	cacheService CacheService,
	config TokenServiceConfig,
) services.JWTService {
	return &tokenService{
		logger:              logger,
		sessionService:      sessionService,
		coreTokenService:    coreTokenService,
		keyService:          keyService,
		cacheService:        cacheService,
		expiresIn:           config.ExpiresIn,
		refreshExpiresIn:    config.RefreshExpiresIn,
		issuer:              config.Issuer,
		audience:            config.Audience,
		accessTokenClientID: config.AccessTokenClientID,
	}
}

func (s *tokenService) detectAlgorithmFromKey() jwa.SignatureAlgorithm {
	return jwa.EdDSA()
}

func (s *tokenService) GenerateUserToken(ctx context.Context, userID string, sessionID string, extraClaims map[string]any) (*types.TokenPair, error) {
	if sessionID == "" {
		return nil, errors.New("session id is required to generate tokens")
	}

	return s.generateUserTokenPair(userID, sessionID, extraClaims)
}

func (s *tokenService) generateUserTokenPair(userID string, sessionID string, extraClaims map[string]any) (*types.TokenPair, error) {
	now := time.Now()
	accessJTI := uuid.New().String()
	refreshJTI := uuid.New().String()

	accessClaims := jwt.New()
	setClaims(accessClaims, map[string]any{
		jwt.SubjectKey:    userID,
		jwt.IssuedAtKey:   now,
		jwt.ExpirationKey: now.Add(s.expiresIn),
		jwt.JwtIDKey:      accessJTI,
		"user_id":         userID,
		"session_id":      sessionID,
		"token_type":      types.JWTTokenTypeAccess.String(),
		"actor_type":      string(models.ActorUser),
		"client_id":       s.accessTokenClientID,
		jwt.IssuerKey:     s.issuer,
		jwt.AudienceKey:   []string{s.audience},
	})
	refreshClaims := jwt.New()
	setClaims(refreshClaims, map[string]any{
		jwt.SubjectKey:    userID,
		jwt.IssuedAtKey:   now,
		jwt.ExpirationKey: now.Add(s.refreshExpiresIn),
		jwt.JwtIDKey:      refreshJTI,
		"user_id":         userID,
		"session_id":      sessionID,
		"token_type":      types.JWTTokenTypeRefresh.String(),
		"actor_type":      string(models.ActorUser),
		jwt.IssuerKey:     s.issuer,
		jwt.AudienceKey:   []string{s.audience},
	})
	s.applyExtraClaimsToTokens(extraClaims, accessClaims, refreshClaims)

	accessTokenBytes, refreshTokenBytes, err := s.signTokenPair(accessClaims, refreshClaims)
	if err != nil {
		return nil, err
	}

	return &types.TokenPair{
		AccessToken:  string(accessTokenBytes),
		RefreshToken: string(refreshTokenBytes),
		ExpiresIn:    s.expiresIn,
		TokenType:    "Bearer",
	}, nil
}

func (s *tokenService) GenerateMachineToken(ctx context.Context, clientID string, organizationID string, scopes []string) (*types.TokenPair, error) {
	return s.generateMachineToken(ctx, clientID, organizationID, scopes, s.expiresIn)
}

func (s *tokenService) IssueClientCredentialsToken(ctx context.Context, req services.ClientCredentialsTokenRequest) (*services.ClientCredentialsToken, error) {
	ttl := req.ExpiresIn
	if ttl <= 0 {
		ttl = s.expiresIn
	}

	pair, err := s.generateMachineToken(ctx, req.ClientID, req.OrganizationID, req.Scopes, ttl)
	if err != nil {
		return nil, err
	}

	return &services.ClientCredentialsToken{
		AccessToken: pair.AccessToken,
		ExpiresIn:   ttl,
		TokenType:   pair.TokenType,
	}, nil
}

func (s *tokenService) generateMachineToken(ctx context.Context, clientID string, organizationID string, scopes []string, ttl time.Duration) (*types.TokenPair, error) {
	if scopes == nil {
		scopes = []string{}
	}

	accessClaims := jwt.New()
	setClaims(accessClaims, map[string]any{
		jwt.SubjectKey:    clientID,
		jwt.IssuedAtKey:   time.Now(),
		jwt.ExpirationKey: time.Now().Add(ttl),
		jwt.JwtIDKey:      uuid.New().String(),
		"token_type":      types.JWTTokenTypeAccess.String(),
		"actor_type":      string(models.ActorMachine),
		"client_id":       clientID,
		"scopes":          scopes,
		jwt.IssuerKey:     s.issuer,
		jwt.AudienceKey:   []string{s.audience},
	})

	if organizationID != "" {
		setClaims(accessClaims, map[string]any{"organization_id": organizationID})
	}
	if len(scopes) > 0 {
		setClaims(accessClaims, map[string]any{"scope": oauth2types.JoinScope(scopes)})
	}

	return s.signMachineToken(accessClaims, ttl)
}

func (s *tokenService) signTokenPair(accessClaims, refreshClaims jwt.Token) ([]byte, []byte, error) {
	privKey, err := s.loadSigningKey()
	if err != nil {
		return nil, nil, err
	}

	keyAlgorithm := s.detectAlgorithmFromKey()
	accessTokenBytes, err := signToken(accessClaims, keyAlgorithm, privKey, types.AccessTokenJWTType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshTokenBytes, err := signToken(refreshClaims, keyAlgorithm, privKey, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessTokenBytes, refreshTokenBytes, nil
}

func (s *tokenService) signMachineToken(accessClaims jwt.Token, ttl time.Duration) (*types.TokenPair, error) {
	privKey, err := s.loadSigningKey()
	if err != nil {
		return nil, err
	}

	keyAlgorithm := s.detectAlgorithmFromKey()
	accessTokenBytes, err := signToken(accessClaims, keyAlgorithm, privKey, types.AccessTokenJWTType)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return &types.TokenPair{
		AccessToken:  string(accessTokenBytes),
		RefreshToken: "",
		ExpiresIn:    ttl,
		TokenType:    "Bearer",
	}, nil
}

func (s *tokenService) loadSigningKey() (jwk.Key, error) {
	jwksKey, err := s.keyService.GetActiveKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	privateKeyPEM, err := s.coreTokenService.Decrypt(jwksKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	privKey, err := jwk.ParseKey([]byte(privateKeyPEM), jwk.WithX509(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	_ = privKey.Set(jwk.KeyIDKey, jwksKey.ID)

	return privKey, nil
}

func signToken(claims jwt.Token, keyAlgorithm jwa.SignatureAlgorithm, privKey jwk.Key, tokenType string) ([]byte, error) {
	if tokenType == "" {
		return jwt.Sign(claims, jwt.WithKey(keyAlgorithm, privKey))
	}

	headers := jws.NewHeaders()
	_ = headers.Set(jws.TypeKey, tokenType)

	return jwt.Sign(claims, jwt.WithKey(keyAlgorithm, privKey, jws.WithProtectedHeaders(headers)))
}

func setClaims(token jwt.Token, claims map[string]any) {
	for key, value := range claims {
		_ = token.Set(key, value)
	}
}

func (s *tokenService) ValidateToken(ctx context.Context, token string) (*models.Actor, error) {
	jwkSet, err := s.cacheService.GetJWKSWithFallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	message, err := jws.Parse([]byte(token))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if err := validateAccessTokenHeaders(message); err != nil {
		return nil, err
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

	actorType, _ := jwt.Get[string](parsedToken, "actor_type")
	actor := &models.Actor{
		Type:   models.ActorType(actorType),
		Claims: map[string]any{"auth_mechanism": "jwt_bearer"},
	}

	if actorType == models.ActorMachine.ToString() {
		actor.ID, _ = jwt.Get[string](parsedToken, jwt.SubjectKey)
		actor.Type = models.ActorMachine

		orgID, err := jwt.Get[string](parsedToken, "organization_id")
		if err == nil && orgID != "" {
			actor.Claims["organization_id"] = orgID
		}

		if raw, ok := parsedToken.Field("scopes"); ok {
			if values, ok := raw.([]any); ok {
				scopes := make([]string, 0, len(values))
				for _, value := range values {
					if scope, ok := value.(string); ok {
						scopes = append(scopes, scope)
					}
				}
				if len(scopes) > 0 {
					actor.Scopes = scopes
				}
			}
		}

		return actor, nil
	}

	userID, err := jwt.Get[string](parsedToken, "user_id")
	if err != nil || userID == "" {
		return nil, errors.New("missing user_id claim")
	}

	sessionID, err := jwt.Get[string](parsedToken, "session_id")
	if err != nil || sessionID == "" {
		return nil, errors.New("missing session_id claim")
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
	for key, val := range parsedToken.Claims() {
		switch key {
		case jwt.SubjectKey, jwt.IssuedAtKey, jwt.ExpirationKey, jwt.JwtIDKey,
			jwt.IssuerKey, jwt.AudienceKey, "user_id", "session_id", "token_type",
			"actor_type", "organization_id", "scopes", "client_id":
			continue
		default:
			actor.Claims[key] = val
		}
	}

	return actor, nil
}

func validateAccessTokenHeaders(message *jws.Message) error {
	signatures := message.Signatures()
	if len(signatures) == 0 {
		return errors.New("missing token signature")
	}

	headers := signatures[0].ProtectedHeaders()
	tokenType, ok := headers.Type()
	if !ok || (tokenType != types.AccessTokenJWTType && tokenType != types.AccessTokenJWTTypeMedia) {
		return errors.New("invalid token type header")
	}

	algorithm, ok := headers.Algorithm()
	if !ok || algorithm.String() != jwa.EdDSA().String() {
		return errors.New("invalid token algorithm")
	}

	return nil
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

	return maps.Collect(parsedToken.Claims()), nil
}

func (s *tokenService) accessTokenValidator() jwt.ValidatorFunc {
	return func(_ context.Context, token jwt.Token) error {
		if err := s.validateTokenPolicy(token, types.JWTTokenTypeAccess); err != nil {
			return err
		}
		if err := validateAccessTokenClaims(token); err != nil {
			return err
		}

		actorType, err := jwt.Get[string](token, "actor_type")
		if err != nil || actorType == "" {
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

func validateAccessTokenClaims(token jwt.Token) error {
	if _, ok := token.Expiration(); !ok {
		return errors.New("missing expiration claim")
	}
	if _, ok := token.IssuedAt(); !ok {
		return errors.New("missing issued at claim")
	}
	if jti, ok := token.JwtID(); !ok || jti == "" {
		return errors.New("missing JWT ID claim")
	}
	if subject, ok := token.Subject(); !ok || subject == "" {
		return errors.New("missing subject claim")
	}
	clientID, err := jwt.Get[string](token, "client_id")
	if err != nil || clientID == "" {
		return errors.New("missing client_id claim")
	}
	return nil
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

	tokenType, err := jwt.Get[string](token, "token_type")
	if err != nil || tokenType == "" {
		return errors.New("missing token_type claim")
	}
	if tokenType != expectedTokenType.String() {
		return errors.New("invalid token_type")
	}

	return nil
}

func (s *tokenService) applyExtraClaimsToTokens(extraClaims map[string]any, tokens ...jwt.Token) {
	for _, token := range tokens {
		for key, value := range extraClaims {
			if _, reserved := reservedTokenClaims[key]; reserved {
				s.logger.Debug("ignoring reserved JWT claim from extra claims", "claim", key)
				continue
			}
			_ = token.Set(key, value)
		}
	}
}
