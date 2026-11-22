package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
	rootservices "github.com/Authula/authula/services"
)

type fakeClientService struct {
	client    *types.OAuthClient
	err       error
	gotID     string
	gotSecret string
}

func (f *fakeClientService) Authenticate(_ context.Context, clientID, clientSecret string) (*types.OAuthClient, error) {
	f.gotID = clientID
	f.gotSecret = clientSecret
	return f.client, f.err
}

type fakeTokenIssuer struct {
	token   *rootservices.ClientCredentialsToken
	err     error
	calls   int
	request rootservices.ClientCredentialsTokenRequest
}

func (f *fakeTokenIssuer) IssueClientCredentialsToken(_ context.Context, req rootservices.ClientCredentialsTokenRequest) (*rootservices.ClientCredentialsToken, error) {
	f.calls++
	f.request = req
	return f.token, f.err
}

func newClientCredentialsFixture(client *types.OAuthClient) (usecases.ClientCredentialsUseCase, *fakeClientService, *fakeTokenIssuer) {
	clients := &fakeClientService{client: client}
	issuer := &fakeTokenIssuer{token: &rootservices.ClientCredentialsToken{
		AccessToken: "access-token",
		ExpiresIn:   15 * time.Minute,
		TokenType:   "Bearer",
	}}
	useCase := usecases.NewClientCredentialsUseCase(
		clients,
		issuer,
		[]string{"read", "write", "admin"},
		15*time.Minute,
		&internaltests.MockLogger{},
	)
	return useCase, clients, issuer
}

func activeClient() *types.OAuthClient {
	return &types.OAuthClient{
		ClientID:      "client-1",
		AllowedGrants: []string{types.OAuthGrantTypeClientCredentials},
		AllowedScopes: []string{"read", "write", "admin"},
		IsActive:      true,
	}
}

func scopePtr(value string) *string {
	return &value
}

func TestClientCredentialsUseCase_IssueToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		client     func() *types.OAuthClient
		scope      *string
		clientErr  error
		issuerErr  error
		wantErr    error
		wantScopes []string
		wantOrg    string
		wantIssued bool
	}{
		{
			name:       "omitted scope issues empty scopes",
			client:     activeClient,
			wantScopes: []string{},
			wantIssued: true,
		},
		{
			name:       "granted scope",
			client:     activeClient,
			scope:      scopePtr("read write"),
			wantScopes: []string{"read", "write"},
			wantIssued: true,
		},
		{
			name: "missing client credentials grant",
			client: func() *types.OAuthClient {
				client := activeClient()
				client.AllowedGrants = []string{types.OAuthGrantTypeAuthorizationCode}
				return client
			},
			wantErr: constants.ErrUnauthorizedClient,
		},
		{
			name:    "empty scope",
			client:  activeClient,
			scope:   scopePtr(""),
			wantErr: constants.ErrInvalidScope,
		},
		{
			name:    "wildcard scope",
			client:  activeClient,
			scope:   scopePtr("*"),
			wantErr: constants.ErrInvalidScope,
		},
		{
			name:    "prefixed wildcard scope",
			client:  activeClient,
			scope:   scopePtr("read*"),
			wantErr: constants.ErrInvalidScope,
		},
		{
			name:    "scope escalation beyond client ceiling",
			client:  activeClient,
			scope:   scopePtr("read delete"),
			wantErr: constants.ErrInvalidScope,
		},
		{
			name: "scope escalation beyond deployment ceiling",
			client: func() *types.OAuthClient {
				client := activeClient()
				client.AllowedScopes = []string{"read", "write", "admin", "billing"}
				return client
			},
			scope:   scopePtr("billing"),
			wantErr: constants.ErrInvalidScope,
		},
		{
			name:    "case sensitive scope",
			client:  activeClient,
			scope:   scopePtr("Read"),
			wantErr: constants.ErrInvalidScope,
		},
		{
			name:      "client authentication failure",
			client:    activeClient,
			clientErr: constants.ErrInvalidClient,
			wantErr:   constants.ErrInvalidClient,
		},
		{
			name:      "issuer failure",
			client:    activeClient,
			issuerErr: errors.New("signing failed"),
			wantErr:   constants.ErrTokenIssuanceFailed,
		},
		{
			name: "organization client id claim",
			client: func() *types.OAuthClient {
				client := activeClient()
				client.OrganizationID = new("org-1")
				return client
			},
			wantScopes: []string{},
			wantOrg:    "org-1",
			wantIssued: true,
		},
		{
			name: "organization owner fallback",
			client: func() *types.OAuthClient {
				client := activeClient()
				client.OwnerType = types.OAuthClientOwnerTypeOrganization
				client.OwnerID = "org-2"
				return client
			},
			wantScopes: []string{},
			wantOrg:    "org-2",
			wantIssued: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase, clients, issuer := newClientCredentialsFixture(tt.client())
			clients.err = tt.clientErr
			issuer.err = tt.issuerErr

			result, err := useCase.IssueToken(context.Background(), &types.TokenRequest{
				ClientID:     "client-1",
				ClientSecret: "secret",
				Scope:        tt.scope,
			})

			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "access-token", result.AccessToken)
			assert.Equal(t, "Bearer", result.TokenType)
			assert.Equal(t, 15*time.Minute, result.ExpiresIn)
			assert.Equal(t, tt.wantScopes, result.Scopes)
			assert.Equal(t, "client-1", clients.gotID)
			assert.Equal(t, "secret", clients.gotSecret)
			assert.True(t, tt.wantIssued)
			assert.Equal(t, 1, issuer.calls)
			assert.Equal(t, tt.wantScopes, issuer.request.Scopes)
			assert.Equal(t, tt.wantOrg, issuer.request.OrganizationID)
			assert.Equal(t, 15*time.Minute, issuer.request.ExpiresIn)
		})
	}
}

func TestClientCredentialsUseCase_RejectsAllOIDCScopes(t *testing.T) {
	t.Parallel()

	for _, scope := range []string{"openid", "profile", "email", "address", "phone", "offline_access"} {
		t.Run(scope, func(t *testing.T) {
			t.Parallel()

			client := activeClient()
			client.AllowedScopes = append(client.AllowedScopes, scope)
			useCase, _, _ := newClientCredentialsFixture(client)

			result, err := useCase.IssueToken(context.Background(), &types.TokenRequest{
				ClientID:     "client-1",
				ClientSecret: "secret",
				Scope:        scopePtr(scope),
			})

			require.ErrorIs(t, err, constants.ErrInvalidScope)
			require.Nil(t, result)
		})
	}
}
