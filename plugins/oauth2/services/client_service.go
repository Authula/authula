package services

import (
	"context"
	"fmt"

	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/types"
)

type clientService struct {
	repo      repositories.OAuthClientRepository
	secrets   ClientSecretService
	dummyHash string
	dummySalt string
}

// NewClientService returns a client authentication service backed by the OAuth
// client repository and the configured secret hashing service.
func NewClientService(repo repositories.OAuthClientRepository, secrets ClientSecretService) ClientService {
	dummyHash, dummySalt, _ := secrets.Hash("authula-dummy-client-secret")
	return &clientService{
		repo:      repo,
		secrets:   secrets,
		dummyHash: dummyHash,
		dummySalt: dummySalt,
	}
}

func (s *clientService) Authenticate(ctx context.Context, clientID, clientSecret string) (*types.OAuthClient, error) {
	client, err := s.repo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", constants.ErrClientLookupFailed, err)
	}

	if client == nil {
		s.secrets.Verify(clientSecret, s.dummyHash, s.dummySalt)
		return nil, constants.ErrInvalidClient
	}

	if !s.secrets.Verify(clientSecret, client.ClientSecretHash, client.ClientSecretSalt) {
		return nil, constants.ErrInvalidClient
	}

	if !client.IsActive {
		return nil, constants.ErrInvalidClient
	}

	return client, nil
}
