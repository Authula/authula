package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/types"
)

type stubOAuthClientRepository struct {
	repositories.OAuthClientRepository
	client *types.OAuthClient
	err    error
}

func (s *stubOAuthClientRepository) GetByClientID(context.Context, string) (*types.OAuthClient, error) {
	return s.client, s.err
}

func TestClientService_Authenticate(t *testing.T) {
	t.Parallel()

	secrets := NewSHA256ClientSecretService("authula_cs_")
	const secret = "authula_cs_correct-secret"
	hash, salt, err := secrets.Hash(secret)
	require.NoError(t, err)

	tests := []struct {
		name        string
		repo        *stubOAuthClientRepository
		clientID    string
		secret      string
		wantErr     error
		wantErrText string
	}{
		{
			name: "valid client",
			repo: &stubOAuthClientRepository{client: &types.OAuthClient{
				ClientID:         "client-1",
				ClientSecretHash: hash,
				ClientSecretSalt: salt,
				IsActive:         true,
			}},
			clientID: "client-1",
			secret:   secret,
		},
		{
			name:     "unknown client",
			repo:     &stubOAuthClientRepository{},
			clientID: "missing",
			secret:   secret,
			wantErr:  constants.ErrInvalidClient,
		},
		{
			name: "wrong secret",
			repo: &stubOAuthClientRepository{client: &types.OAuthClient{
				ClientID:         "client-1",
				ClientSecretHash: hash,
				ClientSecretSalt: salt,
				IsActive:         true,
			}},
			clientID: "client-1",
			secret:   "wrong",
			wantErr:  constants.ErrInvalidClient,
		},
		{
			name: "inactive client",
			repo: &stubOAuthClientRepository{client: &types.OAuthClient{
				ClientID:         "client-1",
				ClientSecretHash: hash,
				ClientSecretSalt: salt,
				IsActive:         false,
			}},
			clientID: "client-1",
			secret:   secret,
			wantErr:  constants.ErrInvalidClient,
		},
		{
			name:     "repository failure",
			repo:     &stubOAuthClientRepository{err: errors.New("database down")},
			clientID: "client-1",
			secret:   secret,
			wantErr:  constants.ErrClientLookupFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewClientService(tt.repo, secrets)
			client, err := service.Authenticate(context.Background(), tt.clientID, tt.secret)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, client)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, tt.clientID, client.ClientID)
		})
	}
}

func TestClientService_DoesNotCollapseRepositoryErrors(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("database down")
	service := NewClientService(
		&stubOAuthClientRepository{err: repoErr},
		NewSHA256ClientSecretService("authula_cs_"),
	)

	_, err := service.Authenticate(context.Background(), "client-1", "secret")

	require.Error(t, err)
	assert.NotErrorIs(t, err, constants.ErrInvalidClient)
	assert.ErrorIs(t, err, repoErr)
}
