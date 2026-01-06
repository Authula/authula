package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// BaseProvider provides common functionality for OAuth2 providers
type BaseProvider struct {
	config *oauth2.Config
	name   string
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, config *oauth2.Config) *BaseProvider {
	return &BaseProvider{
		config: config,
		name:   name,
	}
}

// GetConfig returns the oauth2 config
func (p *BaseProvider) GetConfig() *oauth2.Config {
	return p.config
}

// GetAuthURL returns the authorization URL
func (p *BaseProvider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	return p.config.AuthCodeURL(state, opts...)
}

// Exchange exchanges the authorization code for a token
func (p *BaseProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code, opts...)
}

// Name returns the provider name
func (p *BaseProvider) Name() string {
	return p.name
}

// RequiresPKCE returns whether the provider requires PKCE
func (p *BaseProvider) RequiresPKCE() bool {
	return true
}

// FetchUserInfo is a helper to fetch and parse user info
func FetchUserInfo(ctx context.Context, token *oauth2.Token, url string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	token.SetAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return info, nil
}

// GetStringField safely gets a string field from a map
func GetStringField(data map[string]interface{}, field string) string {
	if val, ok := data[field]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// VerifyHMAC verifies an HMAC signature
func VerifyHMAC(signature, data string, secret []byte) bool {
	expectedMAC := hmac.New(sha256.New, secret)
	expectedMAC.Write([]byte(data))
	expectedSignature := fmt.Sprintf("%x", expectedMAC.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
