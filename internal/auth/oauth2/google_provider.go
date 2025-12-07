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

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type GoogleProvider struct {
	config *domain.OAuth2Config
}

func NewGoogleProvider(config *domain.OAuth2Config) *GoogleProvider {
	return &GoogleProvider{config: config}
}

func (p *GoogleProvider) GetName() string {
	return "google"
}

func (p *GoogleProvider) GetConfig() *oauth2.Config {
	var scopes []string
	if len(p.config.Scopes) > 0 {
		scopes = p.config.Scopes
	} else {
		scopes = []string{
			"openid",
			"email",
			"profile",
		}
	}

	return &oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth?access_type=offline&prompt=consent",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}
}

func (p *GoogleProvider) RequiresPKCE() bool {
	return true
}

func (p *GoogleProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.GetConfig().AuthCodeURL(state, opts...)
}

func (p *GoogleProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return p.GetConfig().Exchange(ctx, code, opts...)
}

func (p *GoogleProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuth2UserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google user info returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var googleUser GoogleUser
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, err
	}

	return &OAuth2UserInfo{
		ID:       googleUser.ID,
		Email:    googleUser.Email,
		Name:     googleUser.Name,
		Picture:  googleUser.Picture,
		Verified: googleUser.VerifiedEmail,
	}, nil
}
