package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
)

// GitHubProvider implements OAuth2Provider for GitHub
type GitHubProvider struct {
	*BaseProvider
}

// NewGitHubProvider creates a new GitHub OAuth2 provider
func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"read:user",
			"user:email",
		},
		Endpoint: github.Endpoint,
	}

	return &GitHubProvider{
		BaseProvider: NewBaseProvider("github", config),
	}
}

// GetUserInfo fetches and normalizes GitHub user information
func (p *GitHubProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, token, "https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}

	// Get email if not in main response
	email := GetStringField(info, "email")
	if email == "" {
		email, _ = p.getGitHubUserEmail(ctx, token)
	}

	// Normalize fields
	userInfo := &types.UserInfo{
		ProviderAccountID: fmt.Sprintf("%v", info["id"]),
		Email:             email,
		Name:              GetStringField(info, "login"),
		Picture:           GetStringField(info, "avatar_url"),
	}

	// Store raw profile
	raw, _ := json.Marshal(info)
	userInfo.Raw = raw

	return userInfo, nil
}

// getGitHubUserEmail fetches the user's email from GitHub
func (p *GitHubProvider) getGitHubUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create email request: %w", err)
	}

	token.SetAuthHeader(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch emails: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read email response: %w", err)
	}

	var emailsArray []map[string]any
	if err := json.Unmarshal(body, &emailsArray); err != nil {
		return "", fmt.Errorf("failed to parse emails: %w", err)
	}

	for _, emailMap := range emailsArray {
		if verified, ok := emailMap["verified"].(bool); ok && verified {
			if primary, ok := emailMap["primary"].(bool); ok && primary {
				if email, ok := emailMap["email"].(string); ok {
					return email, nil
				}
			}
		}
	}

	// If no primary verified email, get any verified email
	for _, emailMap := range emailsArray {
		if verified, ok := emailMap["verified"].(bool); ok && verified {
			if email, ok := emailMap["email"].(string); ok {
				return email, nil
			}
		}
	}

	return "", nil
}

// RequiresPKCE returns false for GitHub (doesn't require PKCE)
func (p *GitHubProvider) RequiresPKCE() bool {
	return false
}
