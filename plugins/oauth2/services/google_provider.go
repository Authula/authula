package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider implements OAuth2Provider for Google
type GoogleProvider struct {
	*BaseProvider
}

// NewGoogleProvider creates a new Google OAuth2 provider
func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleProvider{
		BaseProvider: NewBaseProvider("google", config),
	}
}

// GetUserInfo fetches and normalizes Google user information
func (p *GoogleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, token, "https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google user info: %w", err)
	}

	// Normalize fields
	userInfo := &types.UserInfo{
		ProviderAccountID: GetStringField(info, "id"),
		Email:             GetStringField(info, "email"),
		Name:              GetStringField(info, "name"),
		Picture:           GetStringField(info, "picture"),
	}

	// Store raw profile
	raw, _ := json.Marshal(info)
	userInfo.Raw = raw

	return userInfo, nil
}
