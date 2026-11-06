package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/constants"
)

func TestCookieOptions_flowCookieSameSite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		secure bool
		want   http.SameSite
	}{
		{
			name:   "secure deployments allow the cross-site form post",
			secure: true,
			want:   http.SameSiteNoneMode,
		},
		{
			name:   "plain HTTP falls back to Lax because None requires Secure",
			secure: false,
			want:   http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := CookieOptions{Secure: tt.secure, SameSite: http.SameSiteStrictMode}

			assert.Equal(t, tt.want, opts.flowCookieSameSite())
		})
	}
}

func TestCookieOptions_FlowCookiesUseFlowSameSite(t *testing.T) {
	t.Parallel()

	flowCookies := []string{constants.CookieState, constants.CookieRedirectTo, constants.CookieVerifier}

	tests := []struct {
		name   string
		secure bool
		want   http.SameSite
	}{
		{name: "secure", secure: true, want: http.SameSiteNoneMode},
		{name: "insecure", secure: false, want: http.SameSiteLaxMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := CookieOptions{
				Secure:   tt.secure,
				Domain:   "api.example.com",
				SameSite: http.SameSiteStrictMode,
				MaxAge:   5 * time.Minute,
			}

			t.Run("set", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				for _, name := range flowCookies {
					opts.set(recorder, name, "value-"+name)
				}

				cookies := recorder.Result().Cookies()
				require.Len(t, cookies, len(flowCookies))
				for _, cookie := range cookies {
					assert.Equal(t, tt.want, cookie.SameSite, "%s must use the flow SameSite mode", cookie.Name)
					assert.Equal(t, tt.secure, cookie.Secure, "%s must follow the configured Secure flag", cookie.Name)
					assert.True(t, cookie.HttpOnly, "%s must stay HttpOnly", cookie.Name)
					assert.Equal(t, "api.example.com", cookie.Domain)
					assert.Equal(t, "/", cookie.Path)
					assert.Equal(t, 300, cookie.MaxAge)
				}
			})

			t.Run("clear", func(t *testing.T) {
				recorder := httptest.NewRecorder()
				opts.clearFlowCookies(recorder)

				cookies := recorder.Result().Cookies()
				require.Len(t, cookies, len(flowCookies))
				for _, cookie := range cookies {
					assert.Equal(t, tt.want, cookie.SameSite, "%s must use the flow SameSite mode when cleared", cookie.Name)
					assert.Equal(t, tt.secure, cookie.Secure, "%s must follow the configured Secure flag when cleared", cookie.Name)
					assert.Less(t, cookie.MaxAge, 0, "%s must be expired", cookie.Name)
				}
			})
		})
	}
}
