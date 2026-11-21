package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"maps"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
	oauth2types "github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/services"
)

var _ services.JWTService = (*tokenService)(nil)

func TestNewJWTService(t *testing.T) {
	t.Parallel()

	f := newServiceTestFixture(t)
	config := TokenServiceConfig{
		ExpiresIn:           30 * time.Minute,
		RefreshExpiresIn:    48 * time.Hour,
		Issuer:              "https://issuer.example.com/api/auth",
		Audience:            "test-audience",
		AccessTokenClientID: "test-client",
	}

	svc := NewJWTService(f.logger, f.sessionSvc, f.coreTokenSvc, f.keySvc, f.cacheSvc, config)

	concrete, ok := svc.(*tokenService)
	require.True(t, ok, "NewJWTService must return the concrete token service")
	require.Equal(t, f.logger, concrete.logger)
	require.Equal(t, f.sessionSvc, concrete.sessionService)
	require.Equal(t, f.coreTokenSvc, concrete.coreTokenService)
	require.Equal(t, f.keySvc, concrete.keyService)
	require.Equal(t, f.cacheSvc, concrete.cacheService)
	require.Equal(t, config.ExpiresIn, concrete.expiresIn)
	require.Equal(t, config.RefreshExpiresIn, concrete.refreshExpiresIn)
	require.Equal(t, config.Issuer, concrete.issuer)
	require.Equal(t, config.Audience, concrete.audience)
	require.Equal(t, config.AccessTokenClientID, concrete.accessTokenClientID)
}

func algNoneToken(t *testing.T) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "none", "typ": types.AccessTokenJWTType})
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]any{jwt.SubjectKey: "user-1"})
	require.NoError(t, err)

	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "."
}

func tokenTypeHeader(t *testing.T, tokenStr string) string {
	t.Helper()

	message, err := jws.Parse([]byte(tokenStr))
	require.NoError(t, err)
	require.NotEmpty(t, message.Signatures())

	typ, ok := message.Signatures()[0].ProtectedHeaders().Type()
	require.True(t, ok)
	return typ
}

func parseTestToken(t *testing.T, tokenStr string, f *serviceTestFixture) map[string]any {
	t.Helper()

	pubKey, err := jwk.ParseKey([]byte(f.activeKey.PublicKey), jwk.WithX509(true))
	require.NoError(t, err)

	parsed, err := jwt.Parse([]byte(tokenStr), jwt.WithKey(jwa.EdDSA(), pubKey), jwt.WithValidate(false))
	require.NoError(t, err)

	return maps.Collect(parsed.Claims())
}

func TestTokenService_ValidateToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name        string
		claims      map[string]any
		raw         bool
		withoutType bool
		headers     map[string]any
		rawToken    string
		setupMocks  func(t *testing.T, f *serviceTestFixture)
		wantActor   func(t *testing.T, actor *models.Actor)
		wantErr     string
	}{
		{
			name: "valid_user_token",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-1",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "user-1", actor.ID)
				require.Equal(t, models.ActorUser, actor.Type)
				require.Equal(t, "jwt_bearer", actor.Claims["auth_mechanism"])
			},
		},
		{
			name: "valid_user_token_does_not_leak_registered_claims",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "client-1",
				"organization_id": "org-1",
				"user_id":         "user-1",
				jwt.JwtIDKey:      "jti-leak",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "user-1", actor.ID)
				for _, claim := range []string{"iss", "aud", "client_id", "organization_id", "user_id"} {
					_, ok := actor.Claims[claim]
					require.False(t, ok, "claim %s must not leak into Actor.Claims", claim)
				}
			},
		},
		{
			name: "missing_actor_type",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				jwt.JwtIDKey:      "jti-2",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "missing actor_type claim",
		},
		{
			name: "unknown_actor_type",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "admin",
				jwt.JwtIDKey:      "jti-unknown-actor",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid actor_type claim",
		},
		{
			name: "missing_issuer",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.JwtIDKey:      "jti-missing-iss",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			wantErr: "missing issuer claim",
		},
		{
			name: "wrong_issuer",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.IssuerKey:     "https://evil.example.com",
				jwt.JwtIDKey:      "jti-wrong-iss",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid issuer claim",
		},
		{
			name: "missing_audience",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.IssuerKey:     testIssuer,
				jwt.JwtIDKey:      "jti-missing-aud",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			wantErr: "missing audience claim",
		},
		{
			name: "wrong_audience",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.AudienceKey:   []string{"another-service"},
				jwt.JwtIDKey:      "jti-wrong-aud",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid audience claim",
		},
		{
			name: "external_audience_rejected",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.AudienceKey:   []string{"external-api"},
				jwt.JwtIDKey:      "jti-external-aud",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid audience claim",
		},
		{
			name: "refresh_token_rejected_as_access",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeRefresh.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-refresh-as-access",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid token_type",
		},
		{
			name: "missing_token_type",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-missing-type",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "missing token_type claim",
		},
		{
			name: "valid_machine_token",
			claims: map[string]any{
				jwt.SubjectKey:    "client-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "machine",
				"organization_id": "org-1",
				"scope":           "read:users write:users",
				jwt.JwtIDKey:      "jti-3",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "client-1", actor.ID)
				require.Equal(t, models.ActorMachine, actor.Type)
				orgID, ok := actor.GetClaimString("organization_id")
				require.True(t, ok)
				require.Equal(t, "org-1", orgID)
				require.Equal(t, []string{"read:users", "write:users"}, actor.Scopes)
				require.Equal(t, "jwt_bearer", actor.Claims["auth_mechanism"])
			},
		},
		{
			name: "machine_token_no_optional_fields",
			claims: map[string]any{
				jwt.SubjectKey:    "client-2",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "machine",
				jwt.JwtIDKey:      "jti-4",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "client-2", actor.ID)
				require.Equal(t, models.ActorMachine, actor.Type)
				_, ok := actor.GetClaimString("organization_id")
				require.False(t, ok)
				require.Nil(t, actor.Scopes)
				require.Equal(t, "jwt_bearer", actor.Claims["auth_mechanism"])
			},
		},
		{
			name: "machine_token_scope_not_string",
			claims: map[string]any{
				jwt.SubjectKey:    "client-2",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "machine",
				"scope":           []string{"read"},
				jwt.JwtIDKey:      "jti-machine-scope-type",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid scope claim",
		},
		{
			name: "machine_token_scope_malformed",
			claims: map[string]any{
				jwt.SubjectKey:    "client-2",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "machine",
				"scope":           "read  write",
				jwt.JwtIDKey:      "jti-machine-scope-malformed",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			wantErr: "invalid scope claim",
		},
		{
			name: "expired_token",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-5",
				jwt.IssuedAtKey:   time.Now().Add(-2 * time.Hour),
				jwt.ExpirationKey: time.Now().Add(-1 * time.Hour),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {},
			wantErr:    "failed to parse token",
		},
		{
			name: "missing_subject",
			claims: map[string]any{
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-7",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
			},
			wantErr: "missing subject claim",
		},
		{
			name: "missing_session",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-8",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(nil, nil)
			},
			wantErr: "session not found or invalid",
		},
		{
			name: "expired_session",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-9",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{
					ID: "sess-1", ExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			wantErr: "session expired or invalid",
		},
		{
			name: "missing_session_id",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-missing-session",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
			},
			wantErr: "missing session_id claim",
		},
		{
			name: "machine_token_missing_subject",
			claims: map[string]any{
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "machine",
				jwt.JwtIDKey:      "jti-machine-no-sub",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
			},
			wantErr: "missing subject claim",
		},
		{
			name: "allowlisted_user_claims_are_passed_through",
			claims: map[string]any{
				jwt.SubjectKey:       "user-1",
				"session_id":         "sess-1",
				"token_type":         types.JWTTokenTypeAccess.String(),
				"actor_type":         "user",
				"impersonator_id":    "admin-1",
				"impersonator_scope": "read write",
				jwt.JwtIDKey:         "jti-extra-claims",
				jwt.IssuedAtKey:      time.Now(),
				jwt.ExpirationKey:    time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{
					ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "admin-1", actor.Claims["impersonator_id"])
				require.Equal(t, "read write", actor.Claims["impersonator_scope"])
				require.Equal(t, "sess-1", actor.Claims["session_id"])
			},
		},
		{
			name: "disallowed_user_claims_are_dropped",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"department":      "engineering",
				"role_key":        "admin",
				"tenant":          "acme",
				jwt.JwtIDKey:      "jti-disallowed-claims",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{
					ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour),
				}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				for _, claim := range []string{"department", "role_key", "tenant"} {
					_, ok := actor.Claims[claim]
					require.False(t, ok, "claim %s must not leak into Actor.Claims", claim)
				}
			},
		},
		{
			name: "missing_type_header",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.IssuerKey:     testIssuer,
				jwt.AudienceKey:   []string{testAudience},
				jwt.JwtIDKey:      "jti-no-type",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:         true,
			withoutType: true,
			wantErr:     "invalid token type header",
		},
		{
			name: "typ_jwt_instead_of_at_jwt",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.IssuerKey:     testIssuer,
				jwt.AudienceKey:   []string{testAudience},
				jwt.JwtIDKey:      "jti-wrong-type",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			headers: map[string]any{jws.TypeKey: "JWT"},
			wantErr: "invalid token type header",
		},
		{
			name: "media_type_header_is_accepted",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.IssuerKey:     testIssuer,
				jwt.AudienceKey:   []string{testAudience},
				jwt.JwtIDKey:      "jti-media-type",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			headers: map[string]any{jws.TypeKey: types.AccessTokenJWTTypeMedia},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "user-1", actor.ID)
			},
		},
		{
			name:     "alg_none_is_rejected",
			rawToken: algNoneToken(t),
			wantErr:  "invalid token algorithm",
		},
		{
			name: "missing_client_id",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.IssuerKey:     testIssuer,
				jwt.AudienceKey:   []string{testAudience},
				jwt.JwtIDKey:      "jti-no-client",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			wantErr: "missing client_id claim",
		},
		{
			name: "missing_jti",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.IssuerKey:     testIssuer,
				jwt.AudienceKey:   []string{testAudience},
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			wantErr: "missing JWT ID claim",
		},
		{
			name: "missing_issued_at",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "authula",
				jwt.IssuerKey:     testIssuer,
				jwt.AudienceKey:   []string{testAudience},
				jwt.JwtIDKey:      "jti-no-iat",
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			raw:     true,
			wantErr: "missing issued at claim",
		},
		{
			name: "missing_expiration",
			claims: map[string]any{
				jwt.SubjectKey:  "user-1",
				"session_id":    "sess-1",
				"token_type":    types.JWTTokenTypeAccess.String(),
				"actor_type":    "user",
				"client_id":     "authula",
				jwt.IssuerKey:   testIssuer,
				jwt.AudienceKey: []string{testAudience},
				jwt.JwtIDKey:    "jti-no-exp",
				jwt.IssuedAtKey: time.Now(),
			},
			raw:     true,
			wantErr: "missing expiration claim",
		},
		{
			name:     "malformed_token",
			rawToken: "not-a-jwt",
			wantErr:  "failed to parse token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			f.setupKeyServiceMock()
			if tt.setupMocks != nil {
				tt.setupMocks(t, f)
			}

			tokenStr := tt.rawToken
			switch {
			case tokenStr != "":
			case tt.headers != nil:
				headers := jws.NewHeaders()
				for key, value := range tt.headers {
					require.NoError(t, headers.Set(key, value))
				}
				tokenStr = f.signTestTokenWithHeaders(t, tt.claims, headers)
			case tt.withoutType:
				tokenStr = f.signTestTokenWithoutType(t, tt.claims)
			case tt.raw:
				tokenStr = f.signTestTokenRaw(t, tt.claims)
			default:
				tokenStr = f.signTestToken(t, tt.claims)
			}
			actor, err := svc.ValidateToken(ctx, tokenStr)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, actor)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actor)
				tt.wantActor(t, actor)
			}

			f.sessionSvc.AssertExpectations(t)
		})
	}
}

func TestTokenService_ValidateMachineTokenSkipsSessionLookup(t *testing.T) {
	t.Parallel()

	f := newServiceTestFixture(t)
	f.setupKeyServiceMock()
	svc := f.newJWTService()

	token := f.signTestToken(t, map[string]any{
		jwt.SubjectKey:    "client-1",
		"token_type":      types.JWTTokenTypeAccess.String(),
		"actor_type":      "machine",
		"client_id":       "client-1",
		jwt.JwtIDKey:      "jti-machine",
		jwt.IssuedAtKey:   time.Now(),
		jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
	})

	actor, err := svc.ValidateToken(context.Background(), token)

	require.NoError(t, err)
	require.NotNil(t, actor)
	require.Equal(t, models.ActorMachine, actor.Type)
	f.sessionSvc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestTokenService_GenerateUserToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		userID      string
		sessionID   string
		extraClaims map[string]any
		setupMocks  func(*serviceTestFixture)
		wantErr     string
		wantClaims  map[string]any
		wantAbsent  []string
	}{
		{
			name:      "success",
			userID:    "user-1",
			sessionID: "sess-1",
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
		},
		{
			name:      "allowlisted_extra_claims_are_added",
			userID:    "user-1",
			sessionID: "sess-1",
			extraClaims: map[string]any{
				"impersonator_id":    "admin-1",
				"impersonator_scope": "read write",
			},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantClaims: map[string]any{
				"impersonator_id":    "admin-1",
				"impersonator_scope": "read write",
			},
		},
		{
			name:      "non_allowlisted_extra_claims_are_ignored",
			userID:    "user-1",
			sessionID: "sess-1",
			extraClaims: map[string]any{
				jwt.IssuerKey:     "https://evil.example.com",
				jwt.AudienceKey:   []string{"evil"},
				jwt.SubjectKey:    "evil-subject",
				jwt.ExpirationKey: time.Now().Add(365 * 24 * time.Hour),
				jwt.IssuedAtKey:   time.Now().Add(-time.Hour),
				jwt.NotBeforeKey:  time.Now().Add(-time.Hour),
				jwt.JwtIDKey:      "evil-jti",
				"token_type":      types.JWTTokenTypeRefresh.String(),
				"actor_type":      "machine",
				"user_id":         "evil-user",
				"session_id":      "evil-session",
				"client_id":       "evil-client",
				"scope":           "read",
				"organization_id": "evil-org",
				"tenant":          "evil-tenant",
			},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantAbsent: []string{"user_id", "scope", "organization_id", "tenant"},
		},
		{
			name:       "empty session id",
			userID:     "user-1",
			sessionID:  "",
			setupMocks: func(f *serviceTestFixture) {},
			wantErr:    "session id is required",
		},
		{
			name:      "key service error",
			userID:    "user-1",
			sessionID: "sess-1",
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: "failed to get active key",
		},
		{
			name:      "decrypt error",
			userID:    "user-1",
			sessionID: "sess-1",
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return("", assert.AnError)
			},
			wantErr: "failed to decrypt private key",
		},
		{
			name:      "invalid private key pem",
			userID:    "user-1",
			sessionID: "sess-1",
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return("not a pem block", nil)
			},
			wantErr: "failed to parse private key",
		},
		{
			name:        "signing failure from invalid allowlisted claim value",
			userID:      "user-1",
			sessionID:   "sess-1",
			extraClaims: map[string]any{"impersonator_id": math.NaN()},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantErr: "failed to sign access token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			tt.setupMocks(f)

			ctx := context.Background()
			pair, err := svc.GenerateUserToken(ctx, tt.userID, tt.sessionID, tt.extraClaims)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, pair)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pair)
				require.NotEmpty(t, pair.AccessToken)
				require.NotEmpty(t, pair.RefreshToken)
				require.Equal(t, 15*time.Minute, pair.ExpiresIn)
				require.Equal(t, "Bearer", pair.TokenType)

				claims := parseTestToken(t, pair.AccessToken, f)
				require.Equal(t, f.issuer, claims["iss"])
				require.EqualValues(t, []string{f.audience}, claims["aud"])
				require.Equal(t, tt.userID, claims["sub"])
				require.NotContains(t, claims, "user_id")
				require.Equal(t, tt.sessionID, claims["session_id"])
				require.Equal(t, f.accessTokenClientID, claims["client_id"])
				require.Equal(t, "user", claims["actor_type"])
				require.Equal(t, "access_token", claims["token_type"])
				require.NotEmpty(t, claims["jti"])
				require.Equal(t, types.AccessTokenJWTType, tokenTypeHeader(t, pair.AccessToken))

				require.NotEqual(t, types.AccessTokenJWTType, tokenTypeHeader(t, pair.RefreshToken))

				refreshClaims := parseTestToken(t, pair.RefreshToken, f)
				require.Equal(t, f.issuer, refreshClaims["iss"])
				require.EqualValues(t, []string{f.audience}, refreshClaims["aud"])
				require.Equal(t, tt.userID, refreshClaims["sub"])
				require.NotContains(t, refreshClaims, "user_id")
				require.Equal(t, tt.sessionID, refreshClaims["session_id"])
				require.Equal(t, "refresh_token", refreshClaims["token_type"])
				require.NotContains(t, refreshClaims, "client_id")
				require.NotEmpty(t, refreshClaims["jti"])
				require.NotEqual(t, claims["jti"], refreshClaims["jti"], "access and refresh tokens must not share a jti")

				for claim, want := range tt.wantClaims {
					require.Equal(t, want, claims[claim], "access token claim %q", claim)
					require.Equal(t, want, refreshClaims[claim], "refresh token claim %q", claim)
				}
				for _, claim := range tt.wantAbsent {
					require.NotContains(t, claims, claim, "access token must not contain %q", claim)
					require.NotContains(t, refreshClaims, claim, "refresh token must not contain %q", claim)
				}
			}

			f.keySvc.AssertExpectations(t)
			f.coreTokenSvc.AssertExpectations(t)
		})
	}
}

func TestTokenService_GenerateMachineToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		clientID       string
		organizationID string
		scopes         []string
		setupMocks     func(*serviceTestFixture)
		wantErr        string
		wantOrgID      string
		wantScope      string
	}{
		{
			name:           "success",
			clientID:       "client-1",
			organizationID: "org-1",
			scopes:         []string{"read", "write"},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantOrgID: "org-1",
			wantScope: "read write",
		},
		{
			name:           "no optional fields",
			clientID:       "client-2",
			organizationID: "",
			scopes:         nil,
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
		},
		{
			name:           "key service error",
			clientID:       "client-1",
			organizationID: "org-1",
			scopes:         []string{"read"},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: "failed to get active key",
		},
		{
			name:           "decrypt error",
			clientID:       "client-1",
			organizationID: "org-1",
			scopes:         []string{"read"},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return("", assert.AnError)
			},
			wantErr: "failed to decrypt private key",
		},
		{
			name:           "invalid private key pem",
			clientID:       "client-1",
			organizationID: "",
			scopes:         nil,
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return("not a pem block", nil)
			},
			wantErr: "failed to parse private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			tt.setupMocks(f)

			ctx := context.Background()
			pair, err := svc.GenerateMachineToken(ctx, tt.clientID, tt.organizationID, tt.scopes)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, pair)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pair)
				require.NotEmpty(t, pair.AccessToken)
				require.Empty(t, pair.RefreshToken)
				require.Equal(t, 15*time.Minute, pair.ExpiresIn)
				require.Equal(t, "Bearer", pair.TokenType)

				claims := parseTestToken(t, pair.AccessToken, f)
				require.Equal(t, f.issuer, claims["iss"])
				require.EqualValues(t, []string{f.audience}, claims["aud"])
				require.Equal(t, tt.clientID, claims["sub"])
				require.Equal(t, tt.clientID, claims["client_id"])
				require.NotContains(t, claims, "user_id")
				require.Equal(t, "machine", claims["actor_type"])
				require.Equal(t, "access_token", claims["token_type"])
				require.NotEmpty(t, claims["jti"])
				require.Equal(t, types.AccessTokenJWTType, tokenTypeHeader(t, pair.AccessToken))

				if tt.wantOrgID != "" {
					require.Equal(t, tt.wantOrgID, claims["organization_id"])
				} else {
					require.Empty(t, claims["organization_id"])
				}

				if tt.wantScope != "" {
					require.Equal(t, tt.wantScope, claims["scope"])
				} else {
					require.NotContains(t, claims, "scope")
				}
			}

			f.keySvc.AssertExpectations(t)
			f.coreTokenSvc.AssertExpectations(t)
		})
	}
}

func TestTokenService_IssueClientCredentialsToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     services.ClientCredentialsTokenRequest
		setupMocks  func(*serviceTestFixture)
		wantErr     string
		wantExpires time.Duration
	}{
		{
			name: "uses caller ttl",
			request: services.ClientCredentialsTokenRequest{
				ClientID:       "client-1",
				OrganizationID: "org-1",
				Scopes:         []string{"read", "write"},
				ExpiresIn:      5 * time.Minute,
			},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantExpires: 5 * time.Minute,
		},
		{
			name: "falls back to configured ttl",
			request: services.ClientCredentialsTokenRequest{
				ClientID: "client-1",
			},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantExpires: 15 * time.Minute,
		},
		{
			name: "propagates signing errors",
			request: services.ClientCredentialsTokenRequest{
				ClientID:  "client-1",
				ExpiresIn: time.Minute,
			},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: "failed to get active key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			tt.setupMocks(f)

			token, err := svc.IssueClientCredentialsToken(context.Background(), tt.request)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, token)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, token)
			require.NotEmpty(t, token.AccessToken)
			require.Equal(t, tt.wantExpires, token.ExpiresIn)
			require.Equal(t, "Bearer", token.TokenType)

			claims := parseTestToken(t, token.AccessToken, f)
			pubKey, err := jwk.ParseKey([]byte(f.activeKey.PublicKey), jwk.WithX509(true))
			require.NoError(t, err)
			parsed, err := jwt.Parse([]byte(token.AccessToken), jwt.WithKey(jwa.EdDSA(), pubKey), jwt.WithValidate(false))
			require.NoError(t, err)
			expiresAt, ok := parsed.Expiration()
			require.True(t, ok)
			issuedAt, ok := parsed.IssuedAt()
			require.True(t, ok)
			require.InDelta(t, tt.wantExpires.Seconds(), expiresAt.Sub(issuedAt).Seconds(), 1)

			require.NotContains(t, claims, "user_id")
			if len(tt.request.Scopes) > 0 {
				require.Equal(t, oauth2types.JoinScope(tt.request.Scopes), claims["scope"])
			} else {
				require.NotContains(t, claims, "scope")
			}
		})
	}
}

func TestTokenService_ClientCredentialsTokenRoundTrip(t *testing.T) {

	tests := []struct {
		name    string
		scopes  []string
		wantOrg string
	}{
		{name: "empty scopes", scopes: nil, wantOrg: "org-1"},
		{name: "granted scopes", scopes: []string{"read", "write"}, wantOrg: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			f.setupKeyServiceMock()
			f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			svc := f.newJWTService()

			token, err := svc.IssueClientCredentialsToken(context.Background(), services.ClientCredentialsTokenRequest{
				ClientID:       "client-1",
				OrganizationID: tt.wantOrg,
				Scopes:         tt.scopes,
				ExpiresIn:      10 * time.Minute,
			})
			require.NoError(t, err)

			actor, err := svc.ValidateToken(context.Background(), token.AccessToken)
			require.NoError(t, err)
			require.NotNil(t, actor)
			require.Equal(t, models.ActorMachine, actor.Type)
			require.Equal(t, "client-1", actor.ID)
			require.Equal(t, tt.scopes, actor.Scopes)

			if tt.wantOrg != "" {
				require.Equal(t, tt.wantOrg, actor.Claims["organization_id"])
			}
		})
	}
}

func TestTokenService_ValidateRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	validRefreshClaims := func() map[string]any {
		return map[string]any{
			jwt.SubjectKey:    "user-1",
			"session_id":      "sess-1",
			"token_type":      types.JWTTokenTypeRefresh.String(),
			"actor_type":      "user",
			jwt.JwtIDKey:      "refresh-jti",
			jwt.IssuedAtKey:   time.Now(),
			jwt.ExpirationKey: time.Now().Add(time.Hour),
		}
	}

	tests := []struct {
		name        string
		claims      map[string]any
		raw         bool
		tamper      bool
		wantErr     string
		wantSession string
		wantUser    string
	}{
		{
			name:        "valid_refresh_token",
			claims:      validRefreshClaims(),
			wantSession: "sess-1",
			wantUser:    "user-1",
		},
		{
			name: "access_token_rejected_as_refresh",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				claims["token_type"] = types.JWTTokenTypeAccess.String()
				return claims
			}(),
			wantErr: "invalid token_type",
		},
		{
			name: "missing_issuer",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				return claims
			}(),
			raw:     true,
			wantErr: "missing issuer claim",
		},
		{
			name: "wrong_issuer",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				claims[jwt.IssuerKey] = "https://evil.example.com"
				return claims
			}(),
			wantErr: "invalid issuer claim",
		},
		{
			name: "missing_audience",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				claims[jwt.IssuerKey] = testIssuer
				return claims
			}(),
			raw:     true,
			wantErr: "missing audience claim",
		},
		{
			name: "wrong_audience",
			claims: func() map[string]any {
				claims := validRefreshClaims()
				claims[jwt.AudienceKey] = []string{"another-service"}
				return claims
			}(),
			wantErr: "invalid audience claim",
		},
		{
			name:    "tampered_signature",
			claims:  validRefreshClaims(),
			tamper:  true,
			wantErr: "failed to parse token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			f.setupKeyServiceMock()

			tokenStr := ""
			if tt.raw {
				tokenStr = f.signTestTokenRaw(t, tt.claims)
			} else {
				tokenStr = f.signTestToken(t, tt.claims)
			}
			if tt.tamper {
				// Mutate a byte inside the signature. Changing the final base64url
				// character can leave the decoded signature unchanged.
				signatureStart := strings.LastIndex(tokenStr, ".") + 1
				index := signatureStart + 10
				replacement := "A"
				if tokenStr[index:index+1] == replacement {
					replacement = "B"
				}
				tokenStr = tokenStr[:index] + replacement + tokenStr[index+1:]
			}

			claims, err := svc.ValidateRefreshToken(ctx, tokenStr)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, claims)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantSession, claims["session_id"])
			require.Equal(t, tt.wantUser, claims["sub"])
			require.NotContains(t, claims, "user_id")
		})
	}
}

func TestValidateAccessTokenHeaders(t *testing.T) {
	t.Parallel()

	accessHeaders := func() jws.Headers {
		headers := jws.NewHeaders()
		require.NoError(t, headers.Set(jws.TypeKey, types.AccessTokenJWTType))
		require.NoError(t, headers.Set(jws.AlgorithmKey, jwa.EdDSA()))
		return headers
	}

	tests := []struct {
		name    string
		message *jws.Message
		wantErr string
	}{
		{
			name:    "missing signature",
			message: jws.NewMessage(),
			wantErr: "missing token signature",
		},
		{
			name: "missing type",
			message: jws.NewMessage().AppendSignature(
				jws.NewSignature().SetProtectedHeaders(func() jws.Headers {
					headers := jws.NewHeaders()
					require.NoError(t, headers.Set(jws.AlgorithmKey, jwa.EdDSA()))
					return headers
				}()),
			),
			wantErr: "invalid token type header",
		},
		{
			name: "missing algorithm",
			message: jws.NewMessage().AppendSignature(
				jws.NewSignature().SetProtectedHeaders(func() jws.Headers {
					headers := jws.NewHeaders()
					require.NoError(t, headers.Set(jws.TypeKey, types.AccessTokenJWTType))
					return headers
				}()),
			),
			wantErr: "invalid token algorithm",
		},
		{
			name:    "valid headers",
			message: jws.NewMessage().AppendSignature(jws.NewSignature().SetProtectedHeaders(accessHeaders())),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAccessTokenHeaders(tt.message)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTokenService_SignTokenPairErrors(t *testing.T) {
	t.Parallel()

	f := newServiceTestFixture(t)
	f.setupKeyServiceMock()
	f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
	svc := f.newJWTService()

	validClaims := jwt.New()
	require.NoError(t, validClaims.Set(jwt.SubjectKey, "subject"))
	invalidClaims := jwt.New()
	require.NoError(t, invalidClaims.Set("invalid", math.NaN()))

	_, _, err := svc.signTokenPair(invalidClaims, validClaims)
	require.ErrorContains(t, err, "failed to sign access token")

	_, _, err = svc.signTokenPair(validClaims, invalidClaims)
	require.ErrorContains(t, err, "failed to sign refresh token")

	_, err = svc.signMachineToken(invalidClaims, time.Minute)
	require.ErrorContains(t, err, "failed to sign access token")

	f.keySvc.AssertExpectations(t)
	f.coreTokenSvc.AssertExpectations(t)
}

func TestTokenService_JWKSFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		validate func(*tokenService, context.Context, string) error
	}{
		{
			name: "validate token",
			validate: func(svc *tokenService, ctx context.Context, token string) error {
				_, err := svc.ValidateToken(ctx, token)
				return err
			},
		},
		{
			name: "validate refresh token",
			validate: func(svc *tokenService, ctx context.Context, token string) error {
				_, err := svc.ValidateRefreshToken(ctx, token)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newServiceTestFixture(t)
			f.cacheSvc.On("GetJWKSWithFallback", mock.Anything).Return(nil, assert.AnError)
			svc := f.newJWTService()

			err := tt.validate(svc, context.Background(), "irrelevant-token")

			require.Error(t, err)
			require.Contains(t, err.Error(), "failed to get JWKS")
			f.cacheSvc.AssertExpectations(t)
		})
	}
}
