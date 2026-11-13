package services

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

const defaultGoogleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type GoogleOAuth2Provider struct {
	*BaseProvider
	userInfoURL string
	fields      providerFields
}

var googleDefaults = ProviderDefaults{
	Endpoint: google.Endpoint,
	Scopes: []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	UserInfoURL: defaultGoogleUserInfoURL,
}

func NewGoogleOAuth2Provider(cfg types.ProviderConfig, opts *ProviderOptions) *GoogleOAuth2Provider {
	return &GoogleOAuth2Provider{
		BaseProvider: NewBaseProvider("google", ResolveOAuth2Config(cfg, googleDefaults), opts, cfg.PKCEEnabled()),
		userInfoURL:  ResolveUserInfoURL(cfg, defaultGoogleUserInfoURL),
		fields: providerFields{
			userID:        defaultField(cfg.UserIDField, "id"),
			email:         defaultField(cfg.EmailField, "email"),
			name:          defaultField(cfg.NameField, "name"),
			picture:       defaultField(cfg.PictureField, "picture"),
			emailVerified: defaultField(cfg.EmailVerifiedField, "verified_email"),
		},
	}
}

func (p *GoogleOAuth2Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google user info: %w", err)
	}

	id, err := GetIDField(info, p.fields.userID)
	if err != nil {
		return nil, fmt.Errorf("invalid Google account id: %w", err)
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             util.NormalizeEmail(GetStringField(info, p.fields.email)),
		EmailVerified:     GetBoolField(info, p.fields.emailVerified),
		Name:              GetStringField(info, p.fields.name),
		Picture:           GetStringField(info, p.fields.picture),
		Raw:               info,
	}, nil
}
