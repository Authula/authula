package services

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// GenericProvider implements OAuth2Provider for custom OIDC/OAuth2 endpoints
type GenericProvider struct {
	*BaseProvider
	userInfoURL        string
	userIDField        string
	emailField         string
	nameField          string
	pictureField       string
	emailVerifiedField string
}

// NewGenericProvider creates a new generic OAuth2 provider
func NewGenericProvider(name string, cfg types.ProviderConfig, opts *ProviderOptions) *GenericProvider {
	config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		},
	}

	return &GenericProvider{
		BaseProvider:       NewBaseProvider(name, config, opts),
		userInfoURL:        cfg.UserInfoURL,
		userIDField:        defaultField(cfg.UserIDField, "sub"),
		emailField:         defaultField(cfg.EmailField, "email"),
		nameField:          defaultField(cfg.NameField, "name"),
		pictureField:       defaultField(cfg.PictureField, "picture"),
		emailVerifiedField: cfg.EmailVerifiedField,
	}
}

// GetUserInfo fetches and normalizes user information using custom field mapping
func (p *GenericProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info from %s: %w", p.userInfoURL, err)
	}

	id, err := GetIDField(info, p.userIDField)
	if err != nil {
		return nil, fmt.Errorf("invalid provider account id: %w", err)
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             NormalizeEmail(GetStringField(info, p.emailField)),
		EmailVerified:     GetBoolField(info, p.emailVerifiedField),
		Name:              GetStringField(info, p.nameField),
		Picture:           GetStringField(info, p.pictureField),
		Raw:               info,
	}, nil
}
