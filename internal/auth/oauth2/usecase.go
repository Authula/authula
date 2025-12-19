package oauth2

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type OAuth2UseCase interface {
	// PrepareOAuth2Login generates the authorization URL and state for the OAuth2 login flow
	PrepareOAuth2Login(ctx context.Context, providerName string) (*models.OAuth2LoginResult, error)

	// SignInWithOAuth2 handles the OAuth2 callback, validates the state, exchanges the code for tokens,
	// and either creates a new user or updates an existing one's OAuth2 credentials
	SignInWithOAuth2(ctx context.Context, providerName string, code string, state string, verifier *string) (*models.SignInResult, error)
}
