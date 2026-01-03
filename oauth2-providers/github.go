package oauth2providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type GitHubUser struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Login     string `json:"login"`
}

type GitHubEmailInfo struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type GitHubProvider struct {
	config *models.OAuth2ProviderConfig
}

func NewGitHubProvider(config *models.OAuth2ProviderConfig) *GitHubProvider {
	if envClientID := os.Getenv(env.GithubClientID); envClientID != "" {
		config.ClientID = envClientID
	}
	if envClientSecret := os.Getenv(env.GithubClientSecret); envClientSecret != "" {
		config.ClientSecret = envClientSecret
	}
	return &GitHubProvider{config: config}
}

func (p *GitHubProvider) GetName() string {
	return "github"
}

func (p *GitHubProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		Scopes:       p.config.Scopes,
		RedirectURL:  p.config.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:       "https://github.com/login/oauth/authorize",
			TokenURL:      "https://github.com/login/oauth/access_token",
			DeviceAuthURL: "https://github.com/login/device/code",
		},
	}
}

func (p *GitHubProvider) RequiresPKCE() bool {
	return true
}

func (p *GitHubProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.GetConfig().AuthCodeURL(state, opts...)
}

func (p *GitHubProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return p.GetConfig().Exchange(ctx, code, opts...)
}

func (p *GitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*models.OAuth2UserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user info returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var githubUser GitHubUser
	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, err
	}

	var raw map[string]any
	_ = json.Unmarshal(body, &raw)

	// GitHub email might be private, need to fetch it separately if empty
	email := githubUser.Email
	verified := false

	if email == "" {
		resp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				var emails []GitHubEmailInfo
				if err := json.Unmarshal(body, &emails); err == nil {
					for _, e := range emails {
						if e.Primary {
							email = e.Email
							verified = e.Verified
							break
						}
					}
				}
			}
		}
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	return &models.OAuth2UserInfo{
		ID:       fmt.Sprintf("%d", githubUser.ID),
		Email:    email,
		Name:     name,
		Picture:  githubUser.AvatarURL,
		Verified: verified,
		Raw:      raw,
	}, nil
}
