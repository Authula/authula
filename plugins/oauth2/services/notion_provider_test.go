package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// notionTokenResponse mirrors the documented Notion token response. Optional
// fields are omitted when empty so tests can model partial responses.
type notionTokenResponse struct {
	AccessToken          string            `json:"access_token"`
	TokenType            string            `json:"token_type,omitempty"`
	RefreshToken         string            `json:"refresh_token,omitempty"`
	BotID                string            `json:"bot_id,omitempty"`
	WorkspaceIcon        string            `json:"workspace_icon,omitempty"`
	WorkspaceName        string            `json:"workspace_name,omitempty"`
	WorkspaceID          string            `json:"workspace_id,omitempty"`
	Owner                *notionTokenOwner `json:"owner,omitempty"`
	DuplicatedTemplateID string            `json:"duplicated_template_id,omitempty"`
	RequestID            string            `json:"request_id,omitempty"`
}

type notionTokenOwner struct {
	Type string          `json:"type,omitempty"`
	User notionTokenUser `json:"user"`
}

type notionTokenUser struct {
	Type      string            `json:"type,omitempty"`
	Person    notionTokenPerson `json:"person"`
	Name      string            `json:"name,omitempty"`
	AvatarURL string            `json:"avatar_url,omitempty"`
	ID        string            `json:"id,omitempty"`
	Object    string            `json:"object,omitempty"`
}

type notionTokenPerson struct {
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified"`
}

func newNotionTokenResponse() notionTokenResponse {
	return notionTokenResponse{
		AccessToken:   "notion-access-token",
		TokenType:     "bearer",
		RefreshToken:  "notion-refresh-token",
		BotID:         "3c90c3cc-0d44-4b50-8888-8dd25736052a",
		WorkspaceIcon: "https://img.example/ws.png",
		WorkspaceName: "My Workspace",
		WorkspaceID:   "3c90c3cc-0d44-4b50-8888-8dd25736052a",
		Owner: &notionTokenOwner{
			Type: "user",
			User: notionTokenUser{
				Type: "person",
				Person: notionTokenPerson{
					Email:         "john.doe@example.com",
					EmailVerified: true,
				},
				Name:      "John Doe",
				AvatarURL: "https://img.example/a.png",
				ID:        "notion-user-1",
				Object:    "user",
			},
		},
		DuplicatedTemplateID: "3c90c3cc-0d44-4b50-8888-8dd25736052a",
		RequestID:            "3c90c3cc-0d44-4b50-8888-8dd25736052a",
	}
}

// asMap returns the response the way oauth2.Token keeps token extras.
func (r notionTokenResponse) asMap(t *testing.T) map[string]any {
	t.Helper()

	data, err := json.Marshal(r)
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(data, &fields))
	return fields
}

// ownerObject is the decoded owner object expected in UserInfo.Raw.
func (r notionTokenResponse) ownerObject() map[string]any {
	if r.Owner == nil {
		return nil
	}

	return map[string]any{
		"type": r.Owner.Type,
		"user": map[string]any{
			"type": r.Owner.User.Type,
			"person": map[string]any{
				"email":          r.Owner.User.Person.Email,
				"email_verified": r.Owner.User.Person.EmailVerified,
			},
			"name":       r.Owner.User.Name,
			"avatar_url": r.Owner.User.AvatarURL,
			"id":         r.Owner.User.ID,
			"object":     r.Owner.User.Object,
		},
	}
}

// rawUserInfo is the allowlisted payload expected in UserInfo.Raw.
func (r notionTokenResponse) rawUserInfo() map[string]any {
	return map[string]any{
		"bot_id":                 r.BotID,
		"duplicated_template_id": r.DuplicatedTemplateID,
		"owner":                  r.ownerObject(),
		"request_id":             r.RequestID,
		"workspace_icon":         r.WorkspaceIcon,
		"workspace_id":           r.WorkspaceID,
		"workspace_name":         r.WorkspaceName,
	}
}

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
			"name":       "John Doe",
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

	tests := []struct {
		name       string
		options    []oauth2.AuthCodeOption
		wantQuery  map[string]string
		wantAbsent []string
	}{
		{
			name: "carries the authorize parameters",
			wantQuery: map[string]string{
				"owner":         "user",
				"response_type": "code",
				"client_id":     "client-id",
				"redirect_uri":  notionTestRedirectURL,
				"state":         "state-value",
			},
			wantAbsent: []string{"scope", "code_challenge", "code_challenge_method"},
		},
		{
			name: "keeps caller supplied options",
			options: []oauth2.AuthCodeOption{
				oauth2.AccessTypeOffline,
				oauth2.SetAuthURLParam("prompt", "consent"),
			},
			wantQuery: map[string]string{
				"access_type": "offline",
				"prompt":      "consent",
				"owner":       "user",
				"state":       "state-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			is := assert.New(t)

			parsed, err := url.Parse(provider.GetAuthURL("state-value", tt.options...))
			require.NoError(t, err)

			is.Equal("https://api.notion.com/v1/oauth/authorize", parsed.Scheme+"://"+parsed.Host+parsed.Path)

			query := parsed.Query()
			for key, want := range tt.wantQuery {
				is.Equal(want, query.Get(key), "query parameter %q", key)
			}
			for _, key := range tt.wantAbsent {
				is.False(query.Has(key), "Notion has no %q parameter", key)
			}

			is.Contains(parsed.RawQuery, "redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback")
		})
	}
}

func TestNotionProviderExchange(t *testing.T) {
	t.Parallel()

	documented := newNotionTokenResponse()

	type capturedRequest struct {
		method  string
		path    string
		headers http.Header
		body    map[string]string
	}

	tests := []struct {
		name         string
		response     notionTokenResponse
		wantType     string
		wantRefresh  string
		wantExtras   map[string]any
		wantNoExtras []string
	}{
		{
			name:        "documented payload",
			response:    documented,
			wantType:    documented.TokenType,
			wantRefresh: documented.RefreshToken,
			wantExtras:  documented.asMap(t),
		},
		{
			name:         "only the access token",
			response:     notionTokenResponse{AccessToken: "notion-access-token"},
			wantExtras:   map[string]any{"access_token": "notion-access-token"},
			wantNoExtras: []string{"owner", "refresh_token", "token_type", "workspace_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
				_ = json.NewEncoder(w).Encode(tt.response)
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

			is.Equal(tt.response.AccessToken, token.AccessToken)
			is.Equal(tt.wantType, token.TokenType)
			is.Equal(tt.wantRefresh, token.RefreshToken)

			for key, want := range tt.wantExtras {
				is.Equal(want, token.Extra(key), "token must keep %q", key)
			}
			for _, key := range tt.wantNoExtras {
				is.Nil(token.Extra(key), "token must not carry %q", key)
			}
		})
	}
}

func TestNotionProviderExchangeThenGetUserInfo(t *testing.T) {
	t.Parallel()

	documented := newNotionTokenResponse()

	tests := []struct {
		name         string
		mutate       func(*notionTokenResponse)
		wantEmail    string
		wantVerified bool
	}{
		{
			name:         "documented payload",
			wantEmail:    documented.Owner.User.Person.Email,
			wantVerified: documented.Owner.User.Person.EmailVerified,
		},
		{
			name: "normalizes an unverified email",
			mutate: func(response *notionTokenResponse) {
				response.Owner.User.Person.Email = "  John.Doe@Example.COM  "
				response.Owner.User.Person.EmailVerified = false
			},
			wantEmail:    "john.doe@example.com",
			wantVerified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := newNotionTokenResponse()
			if tt.mutate != nil {
				tt.mutate(&response)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			t.Cleanup(server.Close)

			provider := newNotionTestProvider(server)

			token, err := provider.Exchange(context.Background(), "auth-code")
			require.NoError(t, err)

			info, err := provider.GetUserInfo(context.Background(), token)
			require.NoError(t, err)

			is := assert.New(t)
			is.Equal(response.Owner.User.ID, info.ProviderAccountID)
			is.Equal(tt.wantEmail, info.Email)
			is.Equal(tt.wantVerified, info.EmailVerified)
			is.Equal(response.Owner.User.Name, info.Name)
			is.Equal(response.Owner.User.AvatarURL, info.Picture)
			is.Equal(response.rawUserInfo(), info.Raw)
		})
	}
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
			body:       `{"refresh_token":"refresh"}`,
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
	is.Equal("John Doe", info.Name)
	is.Equal("https://img.example/a.png", info.Picture)
	is.Equal(map[string]any{"owner": owner}, info.Raw)
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
			token:        notionToken(notionOwner(map[string]any{"email": "john.doe@example.com", "email_verified": false})),
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
				"user": map[string]any{"person": map[string]any{"email": "john.doe@example.com"}},
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
			assert.Equal(t, "john.doe@example.com", info.Email)
		})
	}
}

func TestNotionProviderGetUserInfoRawPayload(t *testing.T) {
	t.Parallel()

	owner := notionOwner(map[string]any{"email": "john.doe@example.com", "email_verified": true})
	fixture := newNotionTokenResponse()

	tests := []struct {
		name        string
		extra       map[string]any
		wantRaw     map[string]any
		notContains []string
	}{
		{
			name:        "keeps only the documented non-credential fields",
			extra:       fixture.asMap(t),
			wantRaw:     fixture.rawUserInfo(),
			notContains: []string{"access_token", "refresh_token", "token_type"},
		},
		{
			name: "drops fields missing from the allowlist",
			extra: map[string]any{
				"access_token":      "secret-access-token",
				"refresh_token":     "secret-refresh-token",
				"some_future_field": "secret-future-value",
				"owner":             owner,
			},
			wantRaw:     map[string]any{"owner": owner},
			notContains: []string{"some_future_field"},
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

			is := assert.New(t)

			info, err := provider.GetUserInfo(context.Background(), (&oauth2.Token{}).WithExtra(tt.extra))
			require.NoError(t, err)

			is.Equal(tt.wantRaw, info.Raw)
			for _, key := range tt.notContains {
				_, ok := info.Raw[key]
				is.False(ok, "raw payload must not carry %q", key)
			}
			is.NotContains(fmt.Sprint(info.Raw), "secret-", "raw payload must never carry a credential value")
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
