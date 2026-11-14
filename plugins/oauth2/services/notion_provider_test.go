package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
)

const notionTestRedirectURL = "https://app.example.com/callback"

func newNotionTestProvider(server *httptest.Server) *NotionOAuth2Provider {
	return NewNotionOAuth2Provider(types.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  notionTestRedirectURL,
		TokenURL:     server.URL + "/v1/oauth/token",
	}, newTestProviderOptions(server))
}

func notionOwner(person map[string]any) map[string]any {
	return map[string]any{
		"type": "user",
		"user": map[string]any{
			"object":     "user",
			"id":         "notion-user-1",
			"name":       "Alice",
			"avatar_url": "https://img.example/a.png",
			"type":       "person",
			"person":     person,
		},
	}
}

func notionToken(owner any) *oauth2.Token {
	return (&oauth2.Token{AccessToken: "token"}).WithExtra(map[string]any{"owner": owner})
}

func TestNotionProviderGetAuthURL(t *testing.T) {
	t.Parallel()

	provider := NewNotionOAuth2Provider(types.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  notionTestRedirectURL,
	}, nil)

	t.Run("carries the authorize parameters", func(t *testing.T) {
		t.Parallel()

		is := assert.New(t)

		parsed, err := url.Parse(provider.GetAuthURL("state-value"))
		require.NoError(t, err)

		is.Equal("https://api.notion.com/v1/oauth/authorize", parsed.Scheme+"://"+parsed.Host+parsed.Path)

		query := parsed.Query()
		is.Equal("user", query.Get("owner"))
		is.Equal("code", query.Get("response_type"))
		is.Equal("client-id", query.Get("client_id"))
		is.Equal(notionTestRedirectURL, query.Get("redirect_uri"))
		is.Equal("state-value", query.Get("state"))
		is.Contains(parsed.RawQuery, "redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback")
	})

	t.Run("omits scope and PKCE parameters", func(t *testing.T) {
		t.Parallel()

		is := assert.New(t)

		parsed, err := url.Parse(provider.GetAuthURL("state-value"))
		require.NoError(t, err)

		query := parsed.Query()
		is.False(query.Has("scope"), "Notion has no scope parameter")
		is.False(query.Has("code_challenge"))
		is.False(query.Has("code_challenge_method"))
	})

	t.Run("keeps caller supplied options", func(t *testing.T) {
		t.Parallel()

		is := assert.New(t)

		parsed, err := url.Parse(provider.GetAuthURL("state-value",
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("prompt", "consent"),
		))
		require.NoError(t, err)

		query := parsed.Query()
		is.Equal("offline", query.Get("access_type"))
		is.Equal("consent", query.Get("prompt"))
		is.Equal("user", query.Get("owner"))
		is.Equal("state-value", query.Get("state"))
	})
}

func TestNotionProviderExchange(t *testing.T) {
	t.Parallel()

	const ownerJSON = `{"type":"user","user":{"object":"user","id":"notion-user-1","name":"Alice","avatar_url":"https://img.example/a.png","type":"person","person":{"email":"alice@example.com","email_verified":true}}}`

	type capturedRequest struct {
		method  string
		path    string
		headers http.Header
		body    map[string]string
	}

	captured := make(chan capturedRequest, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		captured <- capturedRequest{
			method:  r.Method,
			path:    r.URL.Path,
			headers: r.Header.Clone(),
			body:    body,
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"notion-access-token","token_type":"bearer","refresh_token":"notion-refresh-token","owner":` + ownerJSON + `}`))
	}))
	t.Cleanup(server.Close)

	token, err := newNotionTestProvider(server).Exchange(context.Background(), "auth-code")
	require.NoError(t, err)

	got := <-captured
	is := assert.New(t)

	is.Equal(http.MethodPost, got.method)
	is.Equal("/v1/oauth/token", got.path)
	is.Equal("Basic "+base64.StdEncoding.EncodeToString([]byte("client-id:client-secret")), got.headers.Get("Authorization"))
	is.Equal("application/json", got.headers.Get("Content-Type"))
	is.Equal("application/json", got.headers.Get("Accept"))
	is.Equal("2026-03-11", got.headers.Get("Notion-Version"))

	is.Equal("authorization_code", got.body["grant_type"])
	is.Equal("auth-code", got.body["code"])
	is.Equal(notionTestRedirectURL, got.body["redirect_uri"])

	is.Equal("notion-access-token", token.AccessToken)
	is.Equal("notion-refresh-token", token.RefreshToken)
	is.Equal("bearer", token.TokenType)

	owner, ok := token.Extra("owner").(map[string]any)
	require.True(t, ok, "owner must survive on the token")
	is.Equal("user", owner["type"])
	user, ok := owner["user"].(map[string]any)
	require.True(t, ok)
	is.Equal("notion-user-1", user["id"])
}

func TestNotionProviderExchangeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		oversized  bool
		wantErrSub string
	}{
		{
			name:       "non 200 status",
			status:     http.StatusUnauthorized,
			body:       `{"object":"error","code":"invalid_grant","message":"invalid code"}`,
			wantErrSub: "status 401",
		},
		{
			name:       "malformed json body",
			status:     http.StatusOK,
			body:       `{`,
			wantErrSub: "parse Notion token response",
		},
		{
			name:       "missing access token",
			status:     http.StatusOK,
			body:       `{"token_type":"bearer","refresh_token":"refresh"}`,
			wantErrSub: "access_token",
		},
		{
			name:       "response body too large",
			status:     http.StatusOK,
			oversized:  true,
			wantErrSub: "token response exceeds 1048576 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := tt.body
			if tt.oversized {
				body = `{"access_token":"` + strings.Repeat("a", maxUserInfoResponseBytes) + `"}`
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(server.Close)

			token, err := newNotionTestProvider(server).Exchange(context.Background(), "auth-code")
			require.ErrorContains(t, err, tt.wantErrSub)
			require.Nil(t, token)
			if tt.oversized {
				require.NotContains(t, err.Error(), "parse Notion token response",
					"an oversized body must fail on size, not on JSON parsing")
			}
		})
	}
}

func TestNotionProviderGetUserInfoMakesNoHttpRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("GetUserInfo must not make an HTTP request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	owner := notionOwner(map[string]any{
		"email":          "  Alice@Example.COM  ",
		"email_verified": true,
	})

	info, err := newNotionTestProvider(server).GetUserInfo(context.Background(), notionToken(owner))
	require.NoError(t, err)

	is := assert.New(t)
	is.Equal("notion-user-1", info.ProviderAccountID)
	is.Equal("alice@example.com", info.Email)
	is.True(info.EmailVerified)
	is.Equal("Alice", info.Name)
	is.Equal("https://img.example/a.png", info.Picture)
	is.Equal(owner, info.Raw)
}

func TestNotionProviderGetUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		token        *oauth2.Token
		wantVerified bool
		wantErrSub   string
	}{
		{
			name:         "unverified email",
			token:        notionToken(notionOwner(map[string]any{"email": "alice@example.com", "email_verified": false})),
			wantVerified: false,
		},
		{
			name:       "workspace owner",
			token:      notionToken(map[string]any{"type": "workspace", "workspace_id": "ws-1"}),
			wantErrSub: `owner type "workspace"`,
		},
		{
			name:       "token without owner",
			token:      &oauth2.Token{AccessToken: "token"},
			wantErrSub: "missing owner",
		},
		{
			name:       "owner without user object",
			token:      notionToken(map[string]any{"type": "user"}),
			wantErrSub: "missing the user object",
		},
		{
			name: "user without person object",
			token: notionToken(map[string]any{
				"type": "user",
				"user": map[string]any{"id": "notion-user-1"},
			}),
			wantErrSub: "missing the person object",
		},
		{
			name: "user without id",
			token: notionToken(map[string]any{
				"type": "user",
				"user": map[string]any{"person": map[string]any{"email": "alice@example.com"}},
			}),
			wantErrSub: "invalid Notion account id",
		},
	}

	provider := NewNotionOAuth2Provider(types.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  notionTestRedirectURL,
	}, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info, err := provider.GetUserInfo(context.Background(), tt.token)
			if tt.wantErrSub != "" {
				require.ErrorContains(t, err, tt.wantErrSub)
				require.Nil(t, info)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantVerified, info.EmailVerified)
			assert.Equal(t, "alice@example.com", info.Email)
		})
	}
}

func TestNotionProviderRequiresPKCE(t *testing.T) {
	t.Parallel()

	provider := NewNotionOAuth2Provider(types.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  notionTestRedirectURL,
	}, nil)

	assert.False(t, provider.RequiresPKCE(), "Notion documents no code_challenge support")
}
