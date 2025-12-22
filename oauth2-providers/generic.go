package oauth2providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type GenericProvider struct {
	name   string
	config *models.OAuth2ProviderConfig
}

func NewGenericProvider(name string, config *models.OAuth2ProviderConfig) *GenericProvider {
	return &GenericProvider{
		name:   name,
		config: config,
	}
}

func (p *GenericProvider) GetName() string {
	return p.name
}

func (p *GenericProvider) GetConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Scopes:       p.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.config.AuthURL,
			TokenURL: p.config.TokenURL,
		},
	}
}

func (p *GenericProvider) RequiresPKCE() bool {
	return false // Default to false for generic providers, can be made configurable if needed
}

func (p *GenericProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.GetConfig().AuthCodeURL(state, opts...)
}

func (p *GenericProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return p.GetConfig().Exchange(ctx, code, opts...)
}

func (p *GenericProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*models.OAuth2UserInfo, error) {
	if p.config.UserInfoURL == "" {
		return nil, fmt.Errorf("user info URL not configured for provider %s", p.name)
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(p.config.UserInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generic user info returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Try to extract common fields
	userInfo := &models.OAuth2UserInfo{
		Raw: data,
	}

	if id, ok := data["id"].(string); ok {
		userInfo.ID = id
	} else if id, ok := data["sub"].(string); ok {
		userInfo.ID = id
	}

	if email, ok := data["email"].(string); ok {
		userInfo.Email = email
	}

	if name, ok := data["name"].(string); ok {
		userInfo.Name = name
	}

	if picture, ok := data["picture"].(string); ok {
		userInfo.Picture = picture
	} else if avatar, ok := data["avatar_url"].(string); ok {
		userInfo.Picture = avatar
	}

	return userInfo, nil
}
