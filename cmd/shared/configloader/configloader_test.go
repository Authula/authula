package configloader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func TestLoadPassesSecretsToConfig(t *testing.T) {
	unsetEnvForTest(t, env.EnvSecrets)

	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(util.Dedent(`
		secret = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

		secrets = [
			{ version = 2, value = "ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA=" },
			{ version = 1, value = "YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=" },
		]
	`)), 0o600))

	config, exists, err := Load(path)
	require.NoError(t, err)
	require.True(t, exists)

	require.Len(t, config.Secrets, 2)
	assert.Equal(t, 2, config.Secrets[0].Version)
	assert.Equal(t, "ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA=", config.Secrets[0].Value)
	assert.Equal(t, 1, config.Secrets[1].Version)
	assert.Equal(t, "YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE=", config.Secrets[1].Value)
}

func TestLoadRejectsInvalidSecrets(t *testing.T) {
	unsetEnvForTest(t, env.EnvSecrets)

	tests := []struct {
		name string
		file string
	}{
		{
			name: "duplicate versions",
			file: util.Dedent(`
				secrets = [
					{ version = 1, value = "one" },
					{ version = 1, value = "two" },
				]
			`),
		},
		{
			name: "non-positive version",
			file: util.Dedent(`
				secrets = [{ version = 0, value = "one" }]
			`),
		},
		{
			name: "empty value",
			file: util.Dedent(`
				secrets = [{ version = 1, value = "" }]
			`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(path, []byte(tt.file), 0o600))

			assert.Panics(t, func() {
				_, _, _ = Load(path)
			})
		})
	}
}

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
