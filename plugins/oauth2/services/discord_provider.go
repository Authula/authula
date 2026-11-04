package services

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/Authula/authula/plugins/oauth2/types"
)

const defaultDiscordUserInfoURL = "https://discord.com/api/users/@me"

type DiscordOAuth2Provider struct {
	*BaseProvider
	userInfoURL string
	fields      providerFields
}

var discordDefaults = ProviderDefaults{
	Endpoint:    endpoints.Discord,
	Scopes:      []string{"identify", "email"},
	UserInfoURL: defaultDiscordUserInfoURL,
}

func NewDiscordOAuth2Provider(cfg types.ProviderConfig, opts *ProviderOptions) *DiscordOAuth2Provider {
	return &DiscordOAuth2Provider{
		BaseProvider: NewBaseProvider("discord", ResolveOAuth2Config(cfg, discordDefaults), opts),
		userInfoURL:  ResolveUserInfoURL(cfg, defaultDiscordUserInfoURL),
		fields: providerFields{
			userID:        defaultField(cfg.UserIDField, "id"),
			email:         defaultField(cfg.EmailField, "email"),
			name:          defaultField(cfg.NameField, "username"),
			picture:       defaultField(cfg.PictureField, "avatar"),
			emailVerified: defaultField(cfg.EmailVerifiedField, "verified"),
		},
	}
}

func (p *DiscordOAuth2Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Discord user info: %w", err)
	}

	id, err := GetIDField(info, p.fields.userID)
	if err != nil {
		return nil, fmt.Errorf("invalid Discord account id: %w", err)
	}

	picture := ""
	if avatar := GetStringField(info, p.fields.picture); avatar != "" {
		picture = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", id, avatar)
	}

	name := GetStringField(info, p.fields.name)
	if name == "" {
		name = GetStringField(info, "global_name")
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             NormalizeEmail(GetStringField(info, p.fields.email)),
		EmailVerified:     GetBoolField(info, p.fields.emailVerified),
		Name:              name,
		Picture:           picture,
		Raw:               info,
	}, nil
}
