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
	fields      providerFields
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
		fields: providerFields{
			userID:        defaultField(cfg.UserIDField, "sub"),
			email:         defaultField(cfg.EmailField, "email"),
			name:          defaultField(cfg.NameField, "name"),
			picture:       cfg.PictureField,
			emailVerified: defaultField(cfg.EmailVerifiedField, "email_verified"),
		},
	}
}

// GetUserInfo fetches and normalizes Dropbox OpenID Connect user information.
func (p *DropboxOAuth2Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfoPost(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Dropbox user info: %w", err)
	}

	id, err := GetIDField(info, p.fields.userID)
	if err != nil {
		return nil, fmt.Errorf("invalid Dropbox account id: %w", err)
	}

	name := GetStringField(info, p.fields.name)
	if name == "" {
		// Dropbox OIDC may omit "name"; fall back to the structured name claims.
		name = strings.TrimSpace(GetStringField(info, "given_name") + " " + GetStringField(info, "family_name"))
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             util.NormalizeEmail(GetStringField(info, p.fields.email)),
		EmailVerified:     GetBoolField(info, p.fields.emailVerified),
		Name:              name,
		Picture:           GetStringField(info, p.fields.picture),
		Raw:               info,
	}, nil
}
