package services

import (
	"context"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// ClientSecretService generates and verifies OAuth client secrets. Secrets are
// only ever returned on creation or rotation paths.
type ClientSecretService interface {
	Generate() (secret string, err error)
	Hash(secret string) (hash string, salt string, err error)
	Verify(secret, hash, salt string) bool
	Prefix(secret string) string
}

// ClientService authenticates confidential OAuth clients.
type ClientService interface {
	Authenticate(ctx context.Context, clientID, clientSecret string) (*types.OAuthClient, error)
}
