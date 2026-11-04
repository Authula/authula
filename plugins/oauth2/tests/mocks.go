package tests

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
)

// MockProvider is a configurable OAuth2 provider used by OAuth2 plugin tests.
type MockProvider struct {
	ProviderName string
	PKCE         bool
	UserInfo     *types.UserInfo
	UserInfoErr  error
	Token        *oauth2.Token
	ExchangeErr  error

	ExchangeCode string
	ExchangeOpts []oauth2.AuthCodeOption
	AuthOptions  []oauth2.AuthCodeOption
}

func (p *MockProvider) Name() string { return p.ProviderName }

func (p *MockProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: "https://app.example.com/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://provider.example.com/authorize",
			TokenURL: "https://provider.example.com/token",
		},
		Scopes: []string{"email"},
	}
}

func (p *MockProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	p.AuthOptions = opts
	return p.GetConfig().AuthCodeURL(state, opts...)
}

func (p *MockProvider) Exchange(_ context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	p.ExchangeCode = code
	p.ExchangeOpts = opts
	if p.ExchangeErr != nil {
		return nil, p.ExchangeErr
	}
	if p.Token != nil {
		return p.Token, nil
	}
	return &oauth2.Token{AccessToken: "access-token"}, nil
}

func (p *MockProvider) GetUserInfo(_ context.Context, _ *oauth2.Token) (*types.UserInfo, error) {
	if p.UserInfoErr != nil {
		return nil, p.UserInfoErr
	}
	return p.UserInfo, nil
}

func (p *MockProvider) RequiresPKCE() bool { return p.PKCE }

// NewRegistry returns a registry containing the provided providers.
func NewRegistry(providers ...types.OAuth2Provider) *services.ProviderRegistry {
	registry := services.NewProviderRegistry()
	for _, provider := range providers {
		_ = registry.Register(provider.Name(), provider)
	}
	return registry
}
