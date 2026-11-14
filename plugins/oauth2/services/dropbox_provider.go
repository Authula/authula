package services

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

const defaultDropboxUserInfoURL = "https://api.dropboxapi.com/2/openid/userinfo"

// DropboxOAuth2Provider implements OAuth2Provider for Dropbox OpenID Connect.
type DropboxOAuth2Provider struct {
	*BaseProvider
	userInfoURL string
}

var dropboxDefaults = ProviderDefaults{
	Endpoint:    endpoints.Dropbox,
	Scopes:      []string{"openid", "profile", "email"},
	UserInfoURL: defaultDropboxUserInfoURL,
}

// NewDropboxOAuth2Provider creates a new Dropbox OAuth2 provider.
func NewDropboxOAuth2Provider(cfg types.ProviderConfig, opts *ProviderOptions) *DropboxOAuth2Provider {
	return &DropboxOAuth2Provider{
		BaseProvider: NewBaseProvider("dropbox", ResolveOAuth2Config(cfg, dropboxDefaults), opts, cfg.PKCEEnabled()),
		userInfoURL:  ResolveUserInfoURL(cfg, defaultDropboxUserInfoURL),
	}
}

// GetUserInfo fetches and normalizes Dropbox OpenID Connect user information.
func (p *DropboxOAuth2Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfoPost(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Dropbox user info: %w", err)
	}

	id, err := GetIDField(info, "sub")
	if err != nil {
		return nil, fmt.Errorf("invalid Dropbox account id: %w", err)
	}

	// Dropbox makes given_name and family_name optional and either may be null.
	name := strings.TrimSpace(GetStringField(info, "given_name") + " " + GetStringField(info, "family_name"))

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             util.NormalizeEmail(GetStringField(info, "email")),
		EmailVerified:     GetBoolField(info, "email_verified"),
		Name:              name,
		Raw:               info,
	}, nil
}
