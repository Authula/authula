package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

const (
	notionAuthURL    = "https://api.notion.com/v1/oauth/authorize"
	notionTokenURL   = "https://api.notion.com/v1/oauth/token"
	notionAPIVersion = "2026-03-11"
)

// NotionOAuth2Provider implements OAuth2Provider for Notion
type NotionOAuth2Provider struct {
	*BaseProvider
	clientID     string
	clientSecret string
}

var notionDefaults = ProviderDefaults{
	Endpoint: oauth2.Endpoint{
		AuthURL:  notionAuthURL,
		TokenURL: notionTokenURL,
		// Exchange below builds its own request, so this is documentation of
		// Notion's requirement rather than something x/oauth2 acts on. It still
		// applies if the inherited BaseProvider.Exchange is ever used.
		AuthStyle: oauth2.AuthStyleInHeader,
	},
	// Notion has no scope parameter: capabilities are configured on the
	// integration itself, so the authorize URL carries no scope at all.
}

// NewNotionOAuth2Provider creates a new Notion OAuth2 provider.
func NewNotionOAuth2Provider(cfg types.ProviderConfig, opts *ProviderOptions) *NotionOAuth2Provider {
	return &NotionOAuth2Provider{
		BaseProvider: NewBaseProvider("notion", ResolveOAuth2Config(cfg, notionDefaults), opts, false),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}
}

// GetAuthURL adds owner=user, which makes Notion return the authorizing person
// rather than a workspace-owned token.
func (p *NotionOAuth2Provider) GetAuthURL(state string, opts ...oauth2.AuthCodeOption) string {
	authOpts := make([]oauth2.AuthCodeOption, 0, len(opts)+1)
	authOpts = append(authOpts, opts...)
	authOpts = append(authOpts, oauth2.SetAuthURLParam("owner", "user"))

	return p.BaseProvider.GetAuthURL(state, authOpts...)
}

// Exchange exchanges the authorization code for a token.
//
// Notion requires HTTP Basic credentials and a JSON body, while x/oauth2 always
// sends a form-encoded body, so the request is built here. The authorize-step
// options cannot be applied to it either: oauth2.AuthCodeOption only exposes an
// unexported setter, and Notion needs none of them.
func (p *NotionOAuth2Provider) Exchange(ctx context.Context, code string, _ ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	payload := map[string]string{
		"grant_type": "authorization_code",
		"code":       code,
	}
	if redirectURL := p.GetConfig().RedirectURL; redirectURL != "" {
		payload["redirect_uri"] = redirectURL
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Notion token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.GetConfig().Endpoint.TokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create Notion token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.SetBasicAuth(p.clientID, p.clientSecret)

	resp, err := p.opts.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange Notion authorization code: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxUserInfoResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read Notion token response: %w", err)
	}
	if len(responseBody) > maxUserInfoResponseBytes {
		return nil, fmt.Errorf("notion token response exceeds %d bytes", maxUserInfoResponseBytes)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to exchange Notion authorization code: status %d", resp.StatusCode)
	}

	// The whole response is kept on the token: Notion has no userinfo endpoint,
	// so the owner object travels with the token and is read in GetUserInfo.
	var raw map[string]any
	if err := json.Unmarshal(responseBody, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Notion token response: %w", err)
	}

	accessToken := GetStringField(raw, "access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("failed to exchange Notion authorization code: response is missing access_token")
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: GetStringField(raw, "refresh_token"),
	}

	return token.WithExtra(raw), nil
}

// GetUserInfo maps the owner object returned with the token. Notion exposes no
// endpoint for the authorizing person, so this performs no HTTP request.
func (p *NotionOAuth2Provider) GetUserInfo(_ context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	rawOwner := token.Extra("owner")
	if rawOwner == nil {
		return nil, fmt.Errorf("failed to get Notion user info: token response is missing owner")
	}

	owner, ok := rawOwner.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to get Notion user info: owner has unexpected type %T", rawOwner)
	}

	// A workspace-owned token carries no person: there is no id and no email to
	// authenticate with, so it cannot be used for login.
	if ownerType := GetStringField(owner, "type"); ownerType != "user" {
		return nil, fmt.Errorf("failed to get Notion user info: owner type %q is not a user", ownerType)
	}

	user, ok := owner["user"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to get Notion user info: owner is missing the user object")
	}

	id, err := GetIDField(user, "id")
	if err != nil {
		return nil, fmt.Errorf("invalid Notion account id: %w", err)
	}

	person, ok := user["person"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to get Notion user info: user is missing the person object")
	}

	cleanRaw := make(map[string]any)
	// Extract non-sensitive properties from Notion's response from the root of the response object.
	keysToKeep := []string{
		"bot_id",
		"duplicated_template_id",
		"owner",
		"workspace_id", "workspace_name", "workspace_icon",
	}
	for _, key := range keysToKeep {
		if val := token.Extra(key); val != nil {
			cleanRaw[key] = val
		}
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             util.NormalizeEmail(GetStringField(person, "email")),
		EmailVerified:     GetBoolField(person, "email_verified"),
		Name:              GetStringField(user, "name"),
		Picture:           GetStringField(user, "avatar_url"),
		Raw:               cleanRaw,
	}, nil
}
