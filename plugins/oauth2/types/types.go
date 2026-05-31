package types

import (
	"context"

	"golang.org/x/oauth2"
)

// OAuth2Provider interface defines the contract for OAuth2 providers
type OAuth2Provider interface {
	Name() string
	GetConfig() *oauth2.Config
	GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	RequiresPKCE() bool
}

// UserInfo represents normalized user information from OAuth2 providers
type UserInfo struct {
	ProviderAccountID string         `json:"provider_account_id"`
	Email             string         `json:"email"`
	Name              string         `json:"name"`
	Picture           string         `json:"picture"`
	Raw               map[string]any `json:"raw"`
}
