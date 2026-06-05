package services

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
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

	var sub, userID, sessionID, tokenType, actType, orgID, jti string
	var scopes []any

	_ = parsed.Get("sub", &sub)
	_ = parsed.Get("user_id", &userID)
	_ = parsed.Get("session_id", &sessionID)
	_ = parsed.Get("type", &tokenType)
	_ = parsed.Get("act_type", &actType)
	_ = parsed.Get("org_id", &orgID)
	_ = parsed.Get("scopes", &scopes)
	_ = parsed.Get(jwt.JwtIDKey, &jti)

	result := make(map[string]any)
	if sub != "" {
		result["sub"] = sub
	}
	if userID != "" {
		result["user_id"] = userID
	}
	if sessionID != "" {
		result["session_id"] = sessionID
	}
	if tokenType != "" {
		result["type"] = tokenType
	}
	if actType != "" {
		result["act_type"] = actType
	}
	if orgID != "" {
		result["org_id"] = orgID
	}
	if len(scopes) > 0 {
		result["scopes"] = scopes
	}
	if jti != "" {
		result["jti"] = jti
	}
	return result
}

func TestTokenService_ValidateToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name       string
		claims     map[string]any
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
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "user",
				jwt.JwtIDKey:      "jti-1",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-1").Return(false, nil)
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "session:sess-1").Return(false, nil)
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{ID: "sess-1"}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "user-1", actor.ID)
				require.Equal(t, models.ActorUser, actor.Type)
				require.Nil(t, actor.OrganizationID)
				require.Equal(t, "jwt_bearer", actor.Metadata["auth_mechanism"])
			},
		},
		{
			name: "valid_user_token_no_act_type",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"type":            types.JWTTokenTypeAccess.String(),
				jwt.JwtIDKey:      "jti-2",
				jwt.IssuedAtKey:   time.Now(),
				jwt.ExpirationKey: time.Now().Add(15 * time.Minute),
			},
			setupMocks: func(t *testing.T, f *serviceTestFixture) {
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "jti-2").Return(false, nil)
				f.blacklistSvc.On("IsBlacklisted", mock.Anything, "session:sess-1").Return(false, nil)
				f.sessionSvc.On("GetByID", mock.Anything, "sess-1").Return(&models.Session{ID: "sess-1"}, nil)
			},
			wantActor: func(t *testing.T, actor *models.Actor) {
				require.Equal(t, "user-1", actor.ID)
				require.Equal(t, models.ActorUser, actor.Type)
				require.Equal(t, "jwt_bearer", actor.Metadata["auth_mechanism"])
			},
		},
		{
			name: "valid_machine_token",
			claims: map[string]any{
				jwt.SubjectKey:    "client-1",
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "machine",
				"org_id":          "org-1",
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
				require.NotNil(t, actor.OrganizationID)
				require.Equal(t, "org-1", *actor.OrganizationID)
				require.Equal(t, []string{"read:users", "write:users"}, actor.Scopes)
				require.Equal(t, "jwt_bearer", actor.Metadata["auth_mechanism"])
			},
		},
		{
			name: "machine_token_no_optional_fields",
			claims: map[string]any{
				jwt.SubjectKey:    "client-2",
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "machine",
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
				require.Nil(t, actor.OrganizationID)
				require.Nil(t, actor.Scopes)
				require.Equal(t, "jwt_bearer", actor.Metadata["auth_mechanism"])
			},
		},
		{
			name: "expired_token",
			claims: map[string]any{
				jwt.SubjectKey:    "user-1",
				"user_id":         "user-1",
				"session_id":      "sess-1",
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "user",
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
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "user",
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
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "user",
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
				"type":            types.JWTTokenTypeAccess.String(),
				"act_type":        "user",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newServiceTestFixture(t)
			svc := f.newJWTService()
			f.setupKeyServiceMock()
			tt.setupMocks(t, f)

			tokenStr := f.signTestToken(t, tt.claims)
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

	t.Run("success", func(t *testing.T) {
		f := newServiceTestFixture(t)
		svc := f.newJWTService()

		f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
		f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)

		ctx := context.Background()
		pair, err := svc.GenerateUserToken(ctx, "user-1", "sess-1")

		require.NoError(t, err)
		require.NotNil(t, pair)
		require.NotEmpty(t, pair.AccessToken)
		require.NotEmpty(t, pair.RefreshToken)
		require.Equal(t, 15*time.Minute, pair.ExpiresIn)
		require.Equal(t, "Bearer", pair.TokenType)

		claims := parseTestToken(t, pair.AccessToken, f)
		require.Equal(t, "user-1", claims["sub"])
		require.Equal(t, "user-1", claims["user_id"])
		require.Equal(t, "sess-1", claims["session_id"])
		require.Equal(t, "user", claims["act_type"])
		require.Equal(t, "access_token", claims["type"])
		require.NotEmpty(t, claims["jti"])

		f.keySvc.AssertExpectations(t)
		f.coreTokenSvc.AssertExpectations(t)
	})

	t.Run("empty_session_id", func(t *testing.T) {
		f := newServiceTestFixture(t)
		svc := f.newJWTService()

		ctx := context.Background()
		pair, err := svc.GenerateUserToken(ctx, "user-1", "")

		require.Error(t, err)
		require.Nil(t, pair)
		require.Contains(t, err.Error(), "session id is required")
	})
}

func TestTokenService_GenerateMachineToken(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		f := newServiceTestFixture(t)
		svc := f.newJWTService()

		f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
		f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)

		ctx := context.Background()
		pair, err := svc.GenerateMachineToken(ctx, "client-1", "org-1", []string{"read", "write"})

		require.NoError(t, err)
		require.NotNil(t, pair)
		require.NotEmpty(t, pair.AccessToken)
		require.Empty(t, pair.RefreshToken)
		require.Equal(t, 15*time.Minute, pair.ExpiresIn)
		require.Equal(t, "Bearer", pair.TokenType)

		claims := parseTestToken(t, pair.AccessToken, f)
		require.Equal(t, "client-1", claims["sub"])
		require.Equal(t, "machine", claims["act_type"])
		require.Equal(t, "org-1", claims["org_id"])
		require.Equal(t, "access_token", claims["type"])
		require.NotEmpty(t, claims["jti"])

		scopes, ok := claims["scopes"].([]any)
		require.True(t, ok)
		require.ElementsMatch(t, []string{"read", "write"}, []any{scopes[0], scopes[1]})

		f.keySvc.AssertExpectations(t)
		f.coreTokenSvc.AssertExpectations(t)
	})

	t.Run("no_optional_fields", func(t *testing.T) {
		f := newServiceTestFixture(t)
		svc := f.newJWTService()

		f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil)
		f.coreTokenSvc.On("Decrypt", f.activeKey.PrivateKey).Return(f.activeKey.PrivateKey, nil)

		ctx := context.Background()
		pair, err := svc.GenerateMachineToken(ctx, "client-2", "", nil)

		require.NoError(t, err)
		require.NotNil(t, pair)
		require.NotEmpty(t, pair.AccessToken)
		require.Empty(t, pair.RefreshToken)

		claims := parseTestToken(t, pair.AccessToken, f)
		require.Equal(t, "client-2", claims["sub"])
		require.Equal(t, "machine", claims["act_type"])
		require.Empty(t, claims["org_id"])
		require.Nil(t, claims["scopes"])

		f.keySvc.AssertExpectations(t)
		f.coreTokenSvc.AssertExpectations(t)
	})
}
