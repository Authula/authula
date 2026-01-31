package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// DiscordProvider implements OAuth2Provider for Discord
type DiscordProvider struct {
	*BaseProvider
}

// NewDiscordProvider creates a new Discord OAuth2 provider
func NewDiscordProvider(clientID, clientSecret, redirectURL string) *DiscordProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"identify",
			"email",
		},
		Endpoint: endpoints.Discord,
	}

	return &DiscordProvider{
		BaseProvider: NewBaseProvider("discord", config),
	}
}

// GetUserInfo fetches and normalizes Discord user information
func (p *DiscordProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, token, "https://discord.com/api/users/@me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Discord user info: %w", err)
	}

	// Normalize fields
	userInfo := &types.UserInfo{
		ProviderAccountID: GetStringField(info, "id"),
		Email:             GetStringField(info, "email"),
		Name:              GetStringField(info, "username"),
		Picture:           fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", GetStringField(info, "id"), GetStringField(info, "avatar")),
	}

	// Store raw profile
	raw, _ := json.Marshal(info)
	userInfo.Raw = raw

	return userInfo, nil
}
