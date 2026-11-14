package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func testProviderConfig(url string) types.ProviderConfig {
	return types.ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://app.example.com/callback",
		UserInfoURL:  url,
	}
}

func newTestProviderOptions(server *httptest.Server) *ProviderOptions {
	return &ProviderOptions{HTTPClient: server.Client()}
}

func TestGoogleProviderGetUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		wantID     string
		wantEmail  string
		wantVerify bool
		wantErr    bool
	}{
		{
			name:       "numeric id is canonical",
			status:     http.StatusOK,
			body:       `{"id": 1234567890123456789, "email": "User@Example.com", "verified_email": true, "name": "User"}`,
			wantID:     "1234567890123456789",
			wantEmail:  "user@example.com",
			wantVerify: true,
		},
		{
			name:      "unverified email",
			status:    http.StatusOK,
			body:      `{"id": "google-1", "email": "user@example.com", "verified_email": false}`,
			wantID:    "google-1",
			wantEmail: "user@example.com",
		},
		{
			name:    "missing id",
			status:  http.StatusOK,
			body:    `{"email": "user@example.com", "verified_email": true}`,
			wantErr: true,
		},
		{
			name:    "nil-like id",
			status:  http.StatusOK,
			body:    `{"id": "<nil>", "email": "user@example.com"}`,
			wantErr: true,
		},
		{
			name:    "non 200",
			status:  http.StatusInternalServerError,
			wantErr: true,
		},
		{
			name:    "malformed json",
			status:  http.StatusOK,
			body:    `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			provider := NewGoogleOAuth2Provider(testProviderConfig(server.URL), newTestProviderOptions(server))
			info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", info)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info.ProviderAccountID != tt.wantID {
				t.Fatalf("ProviderAccountID = %q, want %q", info.ProviderAccountID, tt.wantID)
			}
			if info.Email != tt.wantEmail {
				t.Fatalf("Email = %q, want %q", info.Email, tt.wantEmail)
			}
			if info.EmailVerified != tt.wantVerify {
				t.Fatalf("EmailVerified = %v, want %v", info.EmailVerified, tt.wantVerify)
			}
		})
	}
}

func TestDiscordProviderGetUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		body         string
		wantName     string
		wantPicture  string
		wantVerified bool
	}{
		{
			name:         "empty avatar and username falls back to global name",
			body:         `{"id": "discord-1", "username": "", "global_name": "Global Name", "email": "user@example.com", "verified": true, "avatar": ""}`,
			wantName:     "Global Name",
			wantVerified: true,
		},
		{
			name:        "avatar builds a CDN URL and username is preferred",
			body:        `{"id": "discord-2", "username": "local", "global_name": "Global", "email": "user@example.com", "verified": false, "avatar": "abc"}`,
			wantName:    "local",
			wantPicture: "https://cdn.discordapp.com/avatars/discord-2/abc.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			provider := NewDiscordOAuth2Provider(testProviderConfig(server.URL), newTestProviderOptions(server))
			info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})
			require.NoError(t, err)
			require.Equal(t, tt.wantPicture, info.Picture)
			require.Equal(t, tt.wantName, info.Name)
			require.Equal(t, tt.wantVerified, info.EmailVerified)
		})
	}
}

func TestGitHubProviderEmailVerification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		profileEmail string
		emailsStatus int
		emailsBody   string
		wantEmail    string
		wantVerified bool
		wantErr      bool
	}{
		{
			name:         "verified primary email from emails endpoint",
			emailsStatus: http.StatusOK,
			emailsBody:   `[{"email": "verified@example.com", "primary": true, "verified": true}]`,
			wantEmail:    "verified@example.com",
			wantVerified: true,
		},
		{
			name:         "profile email fallback when emails endpoint fails",
			profileEmail: "Profile@Example.com",
			emailsStatus: http.StatusInternalServerError,
			wantEmail:    "profile@example.com",
		},
		{
			name:         "missing email when endpoint fails and profile is empty",
			emailsStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/user":
					_, _ = w.Write([]byte(`{"id": 9876543210987654321, "name": "GitHub User", "login": "gh", "email": "` + tt.profileEmail + `"}`))
				case "/emails":
					w.WriteHeader(tt.emailsStatus)
					_, _ = w.Write([]byte(tt.emailsBody))
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)

			provider := NewGitHubOAuth2Provider(testProviderConfig(server.URL+"/user"), newTestProviderOptions(server))
			provider.emailsURL = server.URL + "/emails"

			info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", info)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info.Email != tt.wantEmail {
				t.Fatalf("Email = %q, want %q", info.Email, tt.wantEmail)
			}
			if info.EmailVerified != tt.wantVerified {
				t.Fatalf("EmailVerified = %v, want %v", info.EmailVerified, tt.wantVerified)
			}
			if info.ProviderAccountID != "9876543210987654321" {
				t.Fatalf("ProviderAccountID = %q", info.ProviderAccountID)
			}
			if info.Name != "GitHub User" {
				t.Fatalf("Name = %q, want GitHub User", info.Name)
			}
		})
	}
}

func TestGenericProviderFieldMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		body          string
		userIDField   string
		emailField    string
		nameField     string
		pictureField  string
		verifiedField string
		wantID        string
		wantEmail     string
		wantName      string
		wantPicture   string
		wantVerified  bool
	}{
		{
			name:          "custom field mapping",
			body:          `{"sub": 42, "mail": "Person@Example.com", "display_name": "Person", "avatar_url": "https://img.example/p.png", "email_ok": true}`,
			userIDField:   "sub",
			emailField:    "mail",
			nameField:     "display_name",
			pictureField:  "avatar_url",
			verifiedField: "email_ok",
			wantID:        "42",
			wantEmail:     "person@example.com",
			wantName:      "Person",
			wantPicture:   "https://img.example/p.png",
			wantVerified:  true,
		},
		{
			name:        "default field mapping",
			body:        `{"sub": "provider-1", "email": "user@example.com", "name": "User", "picture": "https://img.example/u.png"}`,
			wantID:      "provider-1",
			wantEmail:   "user@example.com",
			wantName:    "User",
			wantPicture: "https://img.example/u.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			cfg := testProviderConfig(server.URL)
			cfg.AuthURL = server.URL + "/authorize"
			cfg.TokenURL = server.URL + "/token"
			cfg.UserIDField = tt.userIDField
			cfg.EmailField = tt.emailField
			cfg.NameField = tt.nameField
			cfg.PictureField = tt.pictureField
			cfg.EmailVerifiedField = tt.verifiedField

			provider := NewGenericOAuth2Provider("custom", cfg, newTestProviderOptions(server))
			info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})
			require.NoError(t, err)
			require.Equal(t, tt.wantID, info.ProviderAccountID)
			require.Equal(t, tt.wantEmail, info.Email)
			require.Equal(t, tt.wantName, info.Name)
			require.Equal(t, tt.wantPicture, info.Picture)
			require.Equal(t, tt.wantVerified, info.EmailVerified)
		})
	}
}

func TestFetchUserInfoPost(t *testing.T) {
	t.Parallel()

	var gotMethod, gotContentType, gotAuthorization, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotAuthorization = r.Header.Get("Authorization")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)

		_, _ = w.Write([]byte(`{"sub": "dbid:123"}`))
	}))
	t.Cleanup(server.Close)

	info, err := FetchUserInfoPost(context.Background(), server.Client(), nil, &oauth2.Token{AccessToken: "secret-token"}, server.URL)
	require.NoError(t, err)
	require.Equal(t, "dbid:123", info["sub"])
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "{}", gotBody)
	require.Equal(t, "application/json", gotContentType)
	require.Equal(t, "Bearer secret-token", gotAuthorization)
}

func TestProvidersRequirePKCEByDefault(t *testing.T) {
	t.Parallel()

	cfg := testProviderConfig("https://provider.example.com/user")

	tests := []struct {
		name     string
		provider func() types.OAuth2Provider
	}{
		{name: "discord", provider: func() types.OAuth2Provider { return NewDiscordOAuth2Provider(cfg, nil) }},
		{name: "dropbox", provider: func() types.OAuth2Provider { return NewDropboxOAuth2Provider(cfg, nil) }},
		{name: "github", provider: func() types.OAuth2Provider { return NewGitHubOAuth2Provider(cfg, nil) }},
		{name: "google", provider: func() types.OAuth2Provider { return NewGoogleOAuth2Provider(cfg, nil) }},
		{name: "generic", provider: func() types.OAuth2Provider { return NewGenericOAuth2Provider("custom", cfg, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.True(t, tt.provider().RequiresPKCE(), "%s must require PKCE by default", tt.name)
		})
	}
}

func TestProviderPKCEOptOutLogsWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pkce        *bool
		wantPKCE    bool
		wantWarning bool
	}{
		{name: "default enables PKCE", pkce: nil, wantPKCE: true, wantWarning: false},
		{name: "explicit true enables PKCE", pkce: new(true), wantPKCE: true, wantWarning: false},
		{name: "explicit false opts out and warns", pkce: new(false), wantPKCE: false, wantWarning: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := testProviderConfig("https://provider.example.com/user")
			cfg.PKCE = tt.pkce

			logger := &internaltests.LevelRecordingLogger{}
			provider := NewGoogleOAuth2Provider(cfg, &ProviderOptions{Logger: logger})

			require.Equal(t, tt.wantPKCE, provider.RequiresPKCE())

			records := logger.Records()
			if !tt.wantWarning {
				require.Empty(t, records)
				return
			}
			require.Len(t, records, 1)
			require.Equal(t, "warn", records[0].Level)
			require.Contains(t, records[0].Message, "not OAuth 2.1 compliant")
		})
	}
}

func TestFetchUserInfoFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		body          string
		delay         time.Duration
		clientTimeout time.Duration
	}{
		{
			name: "body limit",
			body: `{"id":"` + strings.Repeat("a", maxUserInfoResponseBytes) + `"}`,
		},
		{
			name:          "client timeout",
			body:          `{"id":"1"}`,
			delay:         100 * time.Millisecond,
			clientTimeout: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.delay > 0 {
					time.Sleep(tt.delay)
				}
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client := server.Client()
			if tt.clientTimeout > 0 {
				client.Timeout = tt.clientTimeout
			}

			_, err := FetchUserInfo(context.Background(), client, nil, &oauth2.Token{AccessToken: "token"}, server.URL)
			require.Error(t, err)
		})
	}
}
