package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/types"
)

// The browser only deletes a cookie when the clear carries the same scoping attributes it was set with.
func TestAdminPlugin_ClearOriginalCookieMirrorsSessionCookieAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		session      models.SessionConfig
		wantSameSite http.SameSite
	}{
		{
			name:         "host-only lax cookie",
			session:      models.SessionConfig{CookieName: "authula.session_token", HttpOnly: true, SameSite: "lax"},
			wantSameSite: http.SameSiteLaxMode,
		},
		{
			name:         "domain-scoped secure strict cookie",
			session:      models.SessionConfig{CookieName: "authula.session_token", Domain: "example.com", HttpOnly: true, Secure: true, SameSite: "strict"},
			wantSameSite: http.SameSiteStrictMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			globalConfig := &models.Config{Session: tt.session}
			plugin := New(types.AdminPluginConfig{})
			plugin.pluginCtx = &models.PluginContext{GetConfig: func() *models.Config { return globalConfig }}

			w := httptest.NewRecorder()
			plugin.clearOriginalCookie(w, tt.session.CookieName)

			cookies := w.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}
			c := cookies[0]

			if c.Name != tt.session.CookieName+adminconstants.OriginalSessionCookieSuffix {
				t.Errorf("Name = %q", c.Name)
			}
			if c.Value != "" || c.MaxAge != -1 {
				t.Errorf("expected expired cookie, got Value=%q MaxAge=%d", c.Value, c.MaxAge)
			}
			if c.Path != "/" {
				t.Errorf("Path = %q, want /", c.Path)
			}
			if c.Domain != tt.session.Domain {
				t.Errorf("Domain = %q, want %q", c.Domain, tt.session.Domain)
			}
			if c.HttpOnly != tt.session.HttpOnly {
				t.Errorf("HttpOnly = %v, want %v", c.HttpOnly, tt.session.HttpOnly)
			}
			if c.Secure != tt.session.Secure {
				t.Errorf("Secure = %v, want %v", c.Secure, tt.session.Secure)
			}
			if c.SameSite != tt.wantSameSite {
				t.Errorf("SameSite = %v, want %v", c.SameSite, tt.wantSameSite)
			}
		})
	}
}
