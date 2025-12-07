package oauth2

import (
	"context"

	"golang.org/x/oauth2"
)

// OAuth2Provider defines the interface for OAuth2 providers.
type OAuth2Provider interface {
	GetName() string
	GetConfig() *oauth2.Config
	RequiresPKCE() bool
	GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuth2UserInfo, error)
}
