package services

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newAuthURLTestProvider(params map[string]string) *BaseProvider {
	return NewBaseProvider("test", &oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: "https://app.example.com/callback",
		Scopes:      []string{"openid"},
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://provider.example.com/authorize",
		},
	}, nil, true, params)
}

func parseAuthURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}

func TestBaseProviderGetAuthURLParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		params    map[string]string
		options   []oauth2.AuthCodeOption
		wantQuery map[string]string
		wantRaw   []string
	}{
		{
			name:   "configured parameters appear in the authorization URL",
			params: map[string]string{"prompt": "consent", "access_type": "offline"},
			wantQuery: map[string]string{
				"prompt":      "consent",
				"access_type": "offline",
			},
		},
		{
			name:   "caller options override configured parameters",
			params: map[string]string{"prompt": "consent"},
			options: []oauth2.AuthCodeOption{
				oauth2.SetAuthURLParam("prompt", "login"),
			},
			wantQuery: map[string]string{"prompt": "login"},
		},
		{
			name:   "multiple parameters are encoded correctly",
			params: map[string]string{"custom": "a b&c=d"},
			wantQuery: map[string]string{
				"custom": "a b&c=d",
			},
			wantRaw: []string{"custom=a+b%26c%3Dd"},
		},
		{
			name: "no configured parameters leaves the URL unchanged",
			wantQuery: map[string]string{
				"response_type": "code",
				"client_id":     "client-id",
				"redirect_uri":  "https://app.example.com/callback",
				"scope":         "openid",
				"state":         "state-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := newAuthURLTestProvider(tt.params)
			parsed := parseAuthURL(t, provider.GetAuthURL("state-value", tt.options...))
			query := parsed.Query()

			for key, want := range tt.wantQuery {
				require.Equal(t, want, query.Get(key), "query parameter %q", key)
			}
			for _, want := range tt.wantRaw {
				require.Contains(t, parsed.RawQuery, want)
			}

			if tt.params == nil {
				require.Len(t, query, len(tt.wantQuery))
			}
		})
	}
}
