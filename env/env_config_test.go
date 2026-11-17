package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func unsetEnv(t *testing.T, key string) {
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

func writeEnvFile(t *testing.T, dir, contents string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0o600))
}

func TestGetEnv(t *testing.T) {
	const presentKey = "AUTHULA_TEST_GET_ENV_PRESENT"
	const emptyKey = "AUTHULA_TEST_GET_ENV_EMPTY"
	const missingKey = "AUTHULA_TEST_GET_ENV_MISSING"

	t.Setenv(presentKey, "configured")
	t.Setenv(emptyKey, "")
	unsetEnv(t, missingKey)

	tests := []struct {
		name     string
		key      string
		fallback string
		want     string
	}{
		{name: "returns the configured value", key: presentKey, fallback: "fallback", want: "configured"},
		{name: "returns an empty configured value instead of the fallback", key: emptyKey, fallback: "fallback", want: ""},
		{name: "returns the fallback when the variable is missing", key: missingKey, fallback: "fallback", want: "fallback"},
		{name: "returns an empty fallback when the variable is missing", key: missingKey, fallback: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, GetEnv(tt.key, tt.fallback))
		})
	}
}

func TestLoadEnvConfig(t *testing.T) {
	const key = "AUTHULA_TEST_LOAD_ENV"

	tests := []struct {
		name        string
		goEnv       string
		envContents string
		presetValue *string
		wantValue   string
		wantPresent bool
	}{
		{
			name:        "loads .env in development",
			goEnv:       "development",
			envContents: key + "=from-file\n",
			wantValue:   "from-file",
			wantPresent: true,
		},
		{
			name:        "loads .env when GO_ENV is empty",
			goEnv:       "",
			envContents: key + "=from-file\n",
			wantValue:   "from-file",
			wantPresent: true,
		},
		{
			name:        "does not load .env in production",
			goEnv:       "production",
			envContents: key + "=from-file\n",
			wantPresent: false,
		},
		{
			name:        "does not override an existing variable",
			goEnv:       "development",
			envContents: key + "=from-file\n",
			presetValue: new("from-shell"),
			wantValue:   "from-shell",
			wantPresent: true,
		},
		{
			name:        "ignores a missing .env file",
			goEnv:       "development",
			wantPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("GO_ENV", tt.goEnv)

			if tt.envContents != "" {
				writeEnvFile(t, dir, tt.envContents)
			}

			if tt.presetValue != nil {
				t.Setenv(key, *tt.presetValue)
			} else {
				unsetEnv(t, key)
			}

			LoadEnvConfig()

			value, present := os.LookupEnv(key)
			require.Equal(t, tt.wantPresent, present)
			if tt.wantPresent {
				require.Equal(t, tt.wantValue, value)
			}
		})
	}
}
