package configloader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Authula/authula/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKeepsDefaultsForOmittedKeys(t *testing.T) {
	tests := []struct {
		name                 string
		file                 string
		wantHttpOnly         bool
		wantMaxSessions      int
		wantAllowCredentials bool
		wantAllowedOrigins   []string
	}{
		{
			name: "file without a session or security section",
			file: util.Dedent(`
				app_name = "demo"
			`),
			wantHttpOnly:         true,
			wantMaxSessions:      5,
			wantAllowCredentials: true,
			wantAllowedOrigins:   []string{"*"},
		},
		{
			name: "session section that sets only the cookie name",
			file: util.Dedent(`
				[session]
				cookie_name = "sid"
			`),
			wantHttpOnly:         true,
			wantMaxSessions:      5,
			wantAllowCredentials: true,
			wantAllowedOrigins:   []string{"*"},
		},
		{
			name: "explicit values still win",
			file: util.Dedent(`
				[session]
				http_only = false
				max_sessions_per_user = 2

				[security.cors]
				allow_credentials = false
				allowed_origins = ["https://example.com"]
			`),
			wantHttpOnly:         false,
			wantMaxSessions:      2,
			wantAllowCredentials: false,
			wantAllowedOrigins:   []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(path, []byte(tt.file), 0o600))

			config, exists, err := Load(path)
			require.NoError(t, err)
			require.True(t, exists)

			assert.Equal(t, tt.wantHttpOnly, config.Session.HttpOnly, "session.http_only")
			assert.Equal(t, tt.wantMaxSessions, config.Session.MaxSessionsPerUser, "session.max_sessions_per_user")
			assert.Equal(t, tt.wantAllowCredentials, config.Security.CORS.AllowCredentials, "security.cors.allow_credentials")
			assert.Equal(t, tt.wantAllowedOrigins, config.Security.CORS.AllowedOrigins, "security.cors.allowed_origins")
		})
	}
}
