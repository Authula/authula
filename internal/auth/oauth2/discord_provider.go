package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

type DiscordUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Verified bool   `json:"verified"`
}

type DiscordProvider struct {
	config *domain.OAuth2Config
}

func NewDiscordProvider(config *domain.OAuth2Config) *DiscordProvider {
	return &DiscordProvider{
		config: config,
	}
}

func (p *DiscordProvider) GetName() string {
	return "discord"
}

func (p *DiscordProvider) GetConfig() *oauth2.Config {
	var scopes []string
	if len(p.config.Scopes) > 0 {
		scopes = p.config.Scopes
	} else {
		scopes = []string{
			"email",
			"identify",
		}
	}

	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://discord.com/oauth2/authorize?prompt=consent",
			TokenURL: "https://discord.com/api/oauth2/token",
		},
	}
}

func (p *DiscordProvider) RequiresPKCE() bool {
	return true
}

func (p *DiscordProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.GetConfig().AuthCodeURL(state, opts...)
}

func (p *DiscordProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return p.GetConfig().Exchange(ctx, code, opts...)
}

func (p *DiscordProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuth2UserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user info returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var discordUser DiscordUser
	if err := json.Unmarshal(body, &discordUser); err != nil {
		return nil, err
	}

	// Construct avatar URL
	avatarURL := ""
	if discordUser.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordUser.ID, discordUser.Avatar)
	}

	return &OAuth2UserInfo{
		ID:       discordUser.ID,
		Email:    discordUser.Email,
		Name:     discordUser.Username,
		Picture:  avatarURL,
		Verified: discordUser.Verified,
	}, nil
}
