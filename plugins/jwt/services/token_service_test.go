package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/types"
)

func parseTestToken(t *testing.T, tokenStr string, f *serviceTestFixture) map[string]any {
	t.Helper()

	pubKey, err := jwk.ParseKey([]byte(f.activeKey.PublicKey), jwk.WithPEM(true))
	require.NoError(t, err)

	parsed, err := jwt.Parse([]byte(tokenStr), jwt.WithKey(jwa.EdDSA(), pubKey), jwt.WithValidate(false))
	require.NoError(t, err)

	result := make(map[string]any)
	for _, key := range parsed.Keys() {
		var value any
		if err := parsed.Get(key, &value); err == nil {
			result[key] = value
		}
	}
	return result
}

func TestTokenService_ValidateToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name       string
		claims     map[string]any
		raw        bool
		setupMocks func(t *testing.T, f *serviceTestFixture)
		wantActor  func(t *testing.T, actor *models.Actor)
		wantErr    string
	}{
		{
			name: "valid_user_token",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-1",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-1").Return(false, nil)
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "session:sess-1").Return(false, nil)
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
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				"client_id":       "client-1",
				"organization_id": "org-1",
				jwt.JwtIDKey:      "jti-leak",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-leak").Return(false, nil)
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "session:sess-1").Return(false, nil)
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{ID: "sess-1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "user-1", actor.ID)
				for _, claim := range []string{"iss", "aud", "client_id", "organization_id"} {
					_, ok := actor.Claims[claim]
					require.False(t, ok, "claim %s must not leak into Actor.Claims", claim)
				}
			},
		},
		{
			name: "missing_actor_type",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
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
				"user_id":         "user-1",
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
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
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
				"user_id":         "user-1",
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
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
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
				"user_id":         "user-1",
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
				"user_id":         "user-1",
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
				"user_id":         "user-1",
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
			name: "valid_machine_token",
			claims: map[string]any{
				jwt.SubjectKey:    "client-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "machine",
				"organization_id": "org-1",
				"scopes":          []string{"read:users", "write:users"},
				jwt.JwtIDKey:      "jti-3",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-3").Return(false, nil)
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
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-4").Return(false, nil)
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
			name: "expired_token",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
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
			name: "blacklisted_token",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-6",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-6").Return(true, nil)
			},
			wantErr: "token has been revoked",
		},
		{
			name: "missing_user_id",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-7",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-7").Return(false, nil)
			},
			wantErr: "missing user_id claim",
		},
		{
			name: "missing_session",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-8",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-8").Return(false, nil)
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "session:sess-1").Return(false, nil)
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(nil, nil)
			},
			wantErr: "session not found or invalid",
		},
		{
			name: "expired_session",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"token_type":      types.JWTTokenTypeAccess.String(),
				"actor_type":      "user",
				jwt.JwtIDKey:      "jti-9",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-9").Return(false, nil)
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "session:sess-1").Return(false, nil)
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{
					ID: "sess-1", ExpiresAt: time.Now().Add(-time.Minute),
				}, nil)
			},
			wantErr: "session expired or invalid",
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

			tokenStr := ""
			if tt.raw {
				tokenStr = f.signTestTokenRaw(t, tt.claims)
			} else {
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

			f.blacklistSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
		})
	}
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
			name:      "reserved_extra_claims_are_ignored",
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
				"custom":          "kept",
			},
			setupMocks: func(f *serviceTestFixture) {
				f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
				f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)
			},
			wantClaims: map[string]any{"custom": "kept"},
			wantAbsent: []string{"client_id"},
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
				require.Equal(t, []string{f.audience}, claims["aud"])
				require.Equal(t, tt.userID, claims["sub"])
				require.Equal(t, tt.userID, claims["user_id"])
				require.Equal(t, tt.sessionID, claims["session_id"])
				require.Equal(t, "user", claims["actor_type"])
				require.Equal(t, "access_token", claims["token_type"])
				require.NotEmpty(t, claims["jti"])

				refreshClaims := parseTestToken(t, pair.RefreshToken, f)
				require.Equal(t, f.issuer, refreshClaims["iss"])
				require.Equal(t, []string{f.audience}, refreshClaims["aud"])
				require.Equal(t, tt.userID, refreshClaims["user_id"])
				require.Equal(t, tt.sessionID, refreshClaims["session_id"])
				require.Equal(t, "refresh_token", refreshClaims["token_type"])
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
		wantScopes     []any
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
			wantOrgID:  "org-1",
			wantScopes: []any{"read", "write"},
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
				require.Equal(t, []string{f.audience}, claims["aud"])
				require.Equal(t, tt.clientID, claims["sub"])
				require.Equal(t, tt.clientID, claims["client_id"])
				require.Equal(t, "machine", claims["actor_type"])
				require.Equal(t, "access_token", claims["token_type"])
				require.NotEmpty(t, claims["jti"])

				if tt.wantOrgID != "" {
					require.Equal(t, tt.wantOrgID, claims["organization_id"])
				} else {
					require.Empty(t, claims["organization_id"])
				}

				if tt.wantScopes != nil {
					scopes, ok := claims["scopes"].([]any)
					require.True(t, ok)
					require.ElementsMatch(t, tt.wantScopes, scopes)
				} else {
					require.Nil(t, claims["scopes"])
				}
			}

			f.keySvc.AssertExpectations(t)
			f.coreTokenSvc.AssertExpectations(t)
		})
	}
}

func TestTokenService_ValidateRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	validRefreshClaims := func() map[string]any {
		return map[string]any{
			jwt.SubjectKey:    "user-1",
			"user_id":         "user-1",
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
			require.Equal(t, tt.wantUser, claims["user_id"])
		})
	}
}
