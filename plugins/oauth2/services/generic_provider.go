package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
	"golang.org/x/oauth2"
)

// GenericProvider implements OAuth2Provider for custom OIDC/OAuth2 endpoints
type GenericProvider struct {
	*BaseProvider
	userInfoURL  string
	userIDField  string
	emailField   string
	nameField    string
	pictureField string
}

// NewGenericProvider creates a new generic OAuth2 provider
func NewGenericProvider(
	name, clientID, clientSecret, redirectURL,
	authURL, tokenURL, userInfoURL string,
	scopes []string,
	userIDField, emailField, nameField, pictureField string,
) *GenericProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	// Set defaults for field mapping
	if userIDField == "" {
		userIDField = "sub"
	}
	if emailField == "" {
		emailField = "email"
	}
	if nameField == "" {
		nameField = "name"
	}
	if pictureField == "" {
		pictureField = "picture"
	}

	return &GenericProvider{
		BaseProvider: NewBaseProvider(name, config),
		userInfoURL:  userInfoURL,
		userIDField:  userIDField,
		emailField:   emailField,
		nameField:    nameField,
		pictureField: pictureField,
	}
}

// GetUserInfo fetches and normalizes user information using custom field mapping
func (p *GenericProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info from %s: %w", p.userInfoURL, err)
	}

	// Normalize fields using custom mapping
	userInfo := &types.UserInfo{
		ProviderAccountID: fmt.Sprintf("%v", info[p.userIDField]),
		Email:             GetStringField(info, p.emailField),
		Name:              GetStringField(info, p.nameField),
		Picture:           GetStringField(info, p.pictureField),
	}

	// Store raw profile
	raw, _ := json.Marshal(info)
	userInfo.Raw = raw

	return userInfo, nil
}
