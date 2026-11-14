package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestNewDropboxOAuth2Provider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		cfg             types.ProviderConfig
		wantAuthURL     string
		wantTokenURL    string
		wantUserInfoURL string
		wantScopes      []string
		wantPKCE        bool
	}{
		{
			name: "defaults",
			cfg: types.ProviderConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://app.example.com/callback",
			},
			wantAuthURL:     endpoints.Dropbox.AuthURL,
			wantTokenURL:    endpoints.Dropbox.TokenURL,
			wantUserInfoURL: defaultDropboxUserInfoURL,
			wantScopes:      []string{"openid", "profile", "email"},
			wantPKCE:        true,
		},
		{
			name: "custom overrides",
			cfg: types.ProviderConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://app.example.com/callback",
				AuthURL:      "https://auth.example.com/authorize",
				TokenURL:     "https://token.example.com/token",
				UserInfoURL:  "https://api.example.com/userinfo",
				Scopes:       []string{"openid", "email"},
				PKCE:         new(false),
			},
			wantAuthURL:     "https://auth.example.com/authorize",
			wantTokenURL:    "https://token.example.com/token",
			wantUserInfoURL: "https://api.example.com/userinfo",
			wantScopes:      []string{"openid", "email"},
			wantPKCE:        false,
		},
		{
			name: "empty scopes fall back to defaults",
			cfg: types.ProviderConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://app.example.com/callback",
				Scopes:       []string{},
			},
			wantAuthURL:     endpoints.Dropbox.AuthURL,
			wantTokenURL:    endpoints.Dropbox.TokenURL,
			wantUserInfoURL: defaultDropboxUserInfoURL,
			wantScopes:      []string{"openid", "profile", "email"},
			wantPKCE:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewDropboxOAuth2Provider(tt.cfg, nil)

			require.Equal(t, "dropbox", provider.Name())
			require.Equal(t, tt.wantAuthURL, provider.GetConfig().Endpoint.AuthURL)
			require.Equal(t, tt.wantTokenURL, provider.GetConfig().Endpoint.TokenURL)
			require.Equal(t, tt.wantUserInfoURL, provider.userInfoURL)
			require.Equal(t, tt.wantScopes, provider.GetConfig().Scopes)
			require.Equal(t, tt.wantPKCE, provider.RequiresPKCE())

			authURL, err := url.Parse(provider.GetAuthURL("state"))
			require.NoError(t, err)
			require.Equal(t, strings.Join(tt.wantScopes, " "), authURL.Query().Get("scope"))
		})
	}
}

func TestDropboxProviderGetUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		status          int
		body            string
		wantID          string
		wantEmail       string
		wantName        string
		wantVerified    bool
		wantErr         bool
		wantErrContains string
	}{
		{
			name:         "builds the name from structured claims",
			status:       http.StatusOK,
			body:         `{"sub":"dbid:123","email":"User@Example.com","email_verified":true,"given_name":"Jane","family_name":"Doe"}`,
			wantID:       "dbid:123",
			wantEmail:    "user@example.com",
			wantName:     "Jane Doe",
			wantVerified: true,
		},
		{
			name:      "trims structured name claims",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:123","given_name":"  Jane  ","family_name":"  Doe  "}`,
			wantID:    "dbid:123",
			wantName:  "Jane Doe",
			wantEmail: "",
		},
		{
			name:      "given name only",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1","given_name":"Jane"}`,
			wantID:    "dbid:1",
			wantName:  "Jane",
			wantEmail: "",
		},
		{
			name:      "family name only",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1","family_name":"Doe"}`,
			wantID:    "dbid:1",
			wantName:  "Doe",
			wantEmail: "",
		},
		{
			name:      "given name with a null family name",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1","given_name":"John","family_name":null}`,
			wantID:    "dbid:1",
			wantName:  "John",
			wantEmail: "",
		},
		{
			name:      "family name with a null given name",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1","given_name":null,"family_name":"Doe"}`,
			wantID:    "dbid:1",
			wantName:  "Doe",
			wantEmail: "",
		},
		{
			name:      "both name claims are null",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1","given_name":null,"family_name":null}`,
			wantID:    "dbid:1",
			wantName:  "",
			wantEmail: "",
		},
		{
			name:      "no name claims leaves the name empty",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1"}`,
			wantID:    "dbid:1",
			wantName:  "",
			wantEmail: "",
		},
		{
			name:         "normalizes email and accepts a string verification flag",
			status:       http.StatusOK,
			body:         `{"sub":"dbid:1","email":"  User@Example.com  ","email_verified":"true","given_name":"Jane","family_name":"Doe"}`,
			wantID:       "dbid:1",
			wantEmail:    "user@example.com",
			wantName:     "Jane Doe",
			wantVerified: true,
		},
		{
			name:      "unverified email",
			status:    http.StatusOK,
			body:      `{"sub":"dbid:1","email":"jane@example.com","email_verified":false,"given_name":"Jane","family_name":"Doe"}`,
			wantID:    "dbid:1",
			wantEmail: "jane@example.com",
			wantName:  "Jane Doe",
		},
		{
			name:      "numeric sub is preserved",
			status:    http.StatusOK,
			body:      `{"sub":1234567890123456789,"given_name":"Jane","family_name":"Doe"}`,
			wantID:    "1234567890123456789",
			wantName:  "Jane Doe",
			wantEmail: "",
		},
		{
			name:            "missing sub",
			status:          http.StatusOK,
			body:            `{"given_name":"Jane","family_name":"Doe"}`,
			wantErr:         true,
			wantErrContains: "invalid Dropbox account id",
		},
		{
			name:            "empty sub",
			status:          http.StatusOK,
			body:            `{"sub":""}`,
			wantErr:         true,
			wantErrContains: "invalid Dropbox account id",
		},
		{
			name:            "null sub",
			status:          http.StatusOK,
			body:            `{"sub":null}`,
			wantErr:         true,
			wantErrContains: "invalid Dropbox account id",
		},
		{
			name:            "unsupported sub type",
			status:          http.StatusOK,
			body:            `{"sub":["dbid:1"]}`,
			wantErr:         true,
			wantErrContains: "invalid Dropbox account id",
		},
		{
			name:            "non 200",
			status:          http.StatusUnauthorized,
			wantErr:         true,
			wantErrContains: "failed to fetch Dropbox user info",
		},
		{
			name:            "malformed json",
			status:          http.StatusOK,
			body:            `{`,
			wantErr:         true,
			wantErrContains: "failed to fetch Dropbox user info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status := tt.status
			if status == 0 {
				status = http.StatusOK
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			provider := NewDropboxOAuth2Provider(testProviderConfig(server.URL), newTestProviderOptions(server))
			info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErrContains)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantID, info.ProviderAccountID)
			require.Equal(t, tt.wantEmail, info.Email)
			require.Equal(t, tt.wantName, info.Name)
			require.Equal(t, tt.wantVerified, info.EmailVerified)
			require.Empty(t, info.Picture)
			require.Contains(t, info.Raw, "sub")
		})
	}
}

func TestDropboxProviderGetUserInfo_RequestShape(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotContentType, gotBody, gotAuthorization string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotAuthorization = r.Header.Get("Authorization")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}
		gotBody = string(body)

		_, _ = w.Write([]byte(`{"sub":"dbid:123"}`))
	}))
	t.Cleanup(server.Close)

	provider := NewDropboxOAuth2Provider(testProviderConfig(server.URL), newTestProviderOptions(server))
	info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "secret-token"})

	require.NoError(t, err)
	require.Equal(t, "dbid:123", info.ProviderAccountID)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/", gotPath)
	require.Equal(t, "{}", gotBody)
	require.Equal(t, "application/json", gotContentType)
	require.Equal(t, "Bearer secret-token", gotAuthorization)
}
