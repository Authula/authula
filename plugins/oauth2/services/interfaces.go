package services

import (
	"context"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// GeneratedClientSecret contains a new client secret and the metadata needed
// to store it. Value is only returned to the operator once.
type GeneratedClientSecret struct {
	Value  string
	Hash   string
	Salt   string
	Prefix string
	Start  string
	Last   string
}

// ClientSecretService generates and verifies OAuth client secrets. Secrets are
// only ever returned on creation or rotation paths.
type ClientSecretService interface {
	Generate() (*GeneratedClientSecret, error)
	Hash(secret string) (hash string, salt string, err error)
	Verify(secret, hash, salt string) bool
}

// ClientService authenticates confidential OAuth clients.
type ClientService interface {
	Authenticate(ctx context.Context, clientID, clientSecret string) (*types.OAuthClient, error)
}
