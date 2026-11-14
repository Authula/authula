package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/models"
)

const (
	defaultProviderHTTPTimeout = 10 * time.Second
	maxUserInfoResponseBytes   = 1 << 20 // 1 MiB
)

// ProviderOptions carries optional dependencies shared by providers.
type ProviderOptions struct {
	HTTPClient *http.Client
	Logger     models.Logger
}

func (o *ProviderOptions) httpClient() *http.Client {
	if o != nil && o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: defaultProviderHTTPTimeout}
}

func (o *ProviderOptions) logger() models.Logger {
	if o == nil {
		return nil
	}
	return o.Logger
}

// BaseProvider provides common functionality for OAuth2 providers
type BaseProvider struct {
	config *oauth2.Config
	name   string
	opts   ProviderOptions
	pkce   bool
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, config *oauth2.Config, opts *ProviderOptions, pkce bool) *BaseProvider {
	var options ProviderOptions
	if opts != nil {
		options = *opts
	}
	if !pkce && options.Logger != nil {
		options.Logger.Warn(
			"oauth provider has PKCE disabled and is not OAuth 2.1 compliant",
			"provider", name,
		)
	}
	return &BaseProvider{
		config: config,
		name:   name,
		opts:   options,
		pkce:   pkce,
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
	return p.pkce
}

// FetchUserInfo fetches and parses user info with a bounded response body.
func FetchUserInfo(ctx context.Context, client *http.Client, logger models.Logger, token *oauth2.Token, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	return fetchUserInfoResponse(client, logger, token, req)
}

// FetchUserInfoPost fetches and parses user info using a POST request with an
// empty JSON object body.
func FetchUserInfoPost(ctx context.Context, client *http.Client, logger models.Logger, token *oauth2.Token, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return fetchUserInfoResponse(client, logger, token, req)
}

func fetchUserInfoResponse(client *http.Client, logger models.Logger, token *oauth2.Token, req *http.Request) (map[string]any, error) {
	token.SetAuthHeader(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("failed to get user info: empty response")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && logger != nil {
			logger.Debug("failed to close oauth response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUserInfoResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}
	if len(body) > maxUserInfoResponseBytes {
		return nil, fmt.Errorf("user info response exceeds %d bytes", maxUserInfoResponseBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var info map[string]any
	if err := decoder.Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return info, nil
}

// GetStringField safely gets a string field from a map.
func GetStringField(data map[string]any, field string) string {
	if field == "" {
		return ""
	}
	if val, ok := data[field]; ok {
		if str, ok := val.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

// GetBoolField safely gets a boolean field from a map.
func GetBoolField(data map[string]any, field string) bool {
	if field == "" {
		return false
	}
	val, ok := data[field]
	if !ok {
		return false
	}
	switch typed := val.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return err == nil && parsed
	default:
		return false
	}
}

// GetIDField returns a canonical string representation of an identifier field.
// Numeric IDs are kept as integers instead of being formatted through float64.
func GetIDField(data map[string]any, field string) (string, error) {
	if field == "" {
		return "", fmt.Errorf("id field is empty")
	}
	val, ok := data[field]
	if !ok || val == nil {
		return "", fmt.Errorf("id field %q is missing", field)
	}

	var id string
	switch typed := val.(type) {
	case string:
		id = strings.TrimSpace(typed)
	case json.Number:
		id = typed.String()
	case float64:
		id = strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		id = strconv.Itoa(typed)
	case int64:
		id = strconv.FormatInt(typed, 10)
	default:
		return "", fmt.Errorf("id field %q has unsupported type %T", field, val)
	}

	if id == "" || id == "<nil>" || id == "null" {
		return "", fmt.Errorf("id field %q is empty", field)
	}

	return id, nil
}
