package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

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

			provider := NewGoogleProvider(testProviderConfig(server.URL), newTestProviderOptions(server))
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

func TestDiscordProviderEmptyAvatarAndNameFallback(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id": "discord-1", "username": "", "global_name": "Global Name", "email": "user@example.com", "verified": true, "avatar": ""}`))
	}))
	t.Cleanup(server.Close)

	provider := NewDiscordOAuth2Provider(testProviderConfig(server.URL), newTestProviderOptions(server))
	info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Picture != "" {
		t.Fatalf("Picture = %q, want empty", info.Picture)
	}
	if info.Name != "Global Name" {
		t.Fatalf("Name = %q, want Global Name", info.Name)
	}
	if !info.EmailVerified {
		t.Fatal("expected verified email")
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sub": 42, "mail": "Person@Example.com", "display_name": "Person", "avatar_url": "https://img.example/p.png", "email_ok": true}`))
	}))
	t.Cleanup(server.Close)

	cfg := testProviderConfig(server.URL)
	cfg.AuthURL = server.URL + "/authorize"
	cfg.TokenURL = server.URL + "/token"
	cfg.UserIDField = "sub"
	cfg.EmailField = "mail"
	cfg.NameField = "display_name"
	cfg.PictureField = "avatar_url"
	cfg.EmailVerifiedField = "email_ok"

	provider := NewGenericOAuth2Provider("custom", cfg, newTestProviderOptions(server))
	info, err := provider.GetUserInfo(context.Background(), &oauth2.Token{AccessToken: "token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ProviderAccountID != "42" || info.Email != "person@example.com" || info.Name != "Person" || !info.EmailVerified {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestFetchUserInfoBodyLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"` + strings.Repeat("a", maxUserInfoResponseBytes) + `"}`))
	}))
	t.Cleanup(server.Close)

	_, err := FetchUserInfo(context.Background(), server.Client(), nil, &oauth2.Token{AccessToken: "token"}, server.URL)
	if err == nil {
		t.Fatal("expected a body limit error")
	}
}

func TestFetchUserInfoHonoursClientTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	t.Cleanup(server.Close)

	client := server.Client()
	client.Timeout = 10 * time.Millisecond

	_, err := FetchUserInfo(context.Background(), client, nil, &oauth2.Token{AccessToken: "token"}, server.URL)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
}
