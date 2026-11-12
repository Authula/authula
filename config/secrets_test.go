package config

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	"github.com/Authula/authula/models"
)

const (
	customSecretOne = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
	customSecretTwo = "ZmVkY2JhOTg3NjU0MzIxMGZlZGNiYTk4NzY1NDMyMTA="
)

var (
	customSecretOneRaw = mustDecodeBase64(customSecretOne)
	customSecretTwoRaw = mustDecodeBase64(customSecretTwo)
)

func mustDecodeBase64(value string) []byte {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}

	return raw
}

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

func setOptionalEnv(t *testing.T, key string, value *string) {
	t.Helper()

	if value == nil {
		unsetEnv(t, key)
		return
	}

	t.Setenv(key, *value)
}

func applyConfigEnv(t *testing.T, goEnvironment string, secret, secrets *string) {
	t.Helper()

	if goEnvironment == "" {
		unsetEnv(t, env.EnvGoEnvironment)
	} else {
		t.Setenv(env.EnvGoEnvironment, goEnvironment)
	}

	setOptionalEnv(t, env.EnvSecret, secret)
	setOptionalEnv(t, env.EnvSecrets, secrets)
}

func requirePanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected NewConfig to panic")
		}
	}()

	fn()
}

func TestNewConfigSecrets(t *testing.T) {
	wantOrdered := []models.SecretConfig{
		{Version: 2, Value: customSecretTwo, Key: customSecretTwoRaw},
		{Version: 1, Value: customSecretOne, Key: customSecretOneRaw},
	}

	tests := []struct {
		name       string
		envSecrets *string
		options    []ConfigOption
		want       []models.SecretConfig
	}{
		{
			name:    "uses secrets from option",
			options: []ConfigOption{WithSecrets(wantOrdered)},
			want:    wantOrdered,
		},
		{
			name:       "parses and trims secrets from environment",
			envSecrets: new(" 2:" + customSecretTwo + " , 1:" + customSecretOne + " "),
			want:       wantOrdered,
		},
		{
			name:       "environment overrides option",
			envSecrets: new("3:" + customSecretTwo),
			options: []ConfigOption{WithSecrets([]models.SecretConfig{
				{Version: 1, Value: customSecretOne},
			})},
			want: []models.SecretConfig{
				{Version: 3, Value: customSecretTwo, Key: customSecretTwoRaw},
			},
		},
		{
			name:       "empty environment keeps option",
			envSecrets: new(""),
			options: []ConfigOption{WithSecrets([]models.SecretConfig{
				{Version: 1, Value: customSecretOne},
			})},
			want: []models.SecretConfig{
				{Version: 1, Value: customSecretOne, Key: customSecretOneRaw},
			},
		},
		{
			name:    "unset environment keeps option",
			options: []ConfigOption{WithSecrets(wantOrdered)},
			want:    wantOrdered,
		},
		{
			name:       "allows version gaps",
			envSecrets: new("1:" + customSecretOne + ",3:" + customSecretTwo),
			want: []models.SecretConfig{
				{Version: 1, Value: customSecretOne, Key: customSecretOneRaw},
				{Version: 3, Value: customSecretTwo, Key: customSecretTwoRaw},
			},
		},
		{
			name:       "keeps the first entry as current",
			envSecrets: new("7:" + customSecretOne + ",2:" + customSecretTwo),
			want: []models.SecretConfig{
				{Version: 7, Value: customSecretOne, Key: customSecretOneRaw},
				{Version: 2, Value: customSecretTwo, Key: customSecretTwoRaw},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyConfigEnv(t, "", nil, tt.envSecrets)

			config := NewConfig(tt.options...)

			assert.Equal(t, tt.want, config.Secrets)
		})
	}
}

func TestNewConfigSecret(t *testing.T) {
	defaultRaw := mustDecodeBase64(defaultSecret)

	tests := []struct {
		name    string
		env     *string
		options []ConfigOption
		want    string
		wantKey []byte
	}{
		{
			name:    "trims secret from option",
			options: []ConfigOption{WithSecret("  " + customSecretOne + "\n")},
			want:    customSecretOne,
			wantKey: customSecretOneRaw,
		},
		{
			name:    "uses and trims secret from environment",
			env:     new("  " + customSecretOne + "\n"),
			options: []ConfigOption{WithSecret("")},
			want:    customSecretOne,
			wantKey: customSecretOneRaw,
		},
		{
			name:    "falls back to the default outside production",
			want:    defaultSecret,
			wantKey: defaultRaw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyConfigEnv(t, "", tt.env, nil)

			config := NewConfig(tt.options...)

			assert.Equal(t, tt.want, config.Secret)
			assert.Equal(t, tt.wantKey, config.SecretKey)
			assert.Empty(t, config.Secrets)
		})
	}
}

func TestNewConfigSecretsValidationPanics(t *testing.T) {
	tests := []struct {
		name       string
		envSecrets *string
		options    []ConfigOption
	}{
		{
			name:       "rejects a zero version from the environment",
			envSecrets: new("0:" + customSecretOne),
		},
		{
			name:       "rejects a negative version from the environment",
			envSecrets: new("-1:" + customSecretOne),
		},
		{
			name:       "rejects a non-numeric version from the environment",
			envSecrets: new("abc:" + customSecretOne),
		},
		{
			name:       "rejects a duplicate version from the environment",
			envSecrets: new("1:" + customSecretOne + ",1:" + customSecretTwo),
		},
		{
			name:       "rejects an empty value from the environment",
			envSecrets: new("1:"),
		},
		{
			name:       "rejects a missing colon from the environment",
			envSecrets: new("1"),
		},
		{
			name:       "rejects an empty entry from the environment",
			envSecrets: new("1:" + customSecretOne + ","),
		},
		{
			name:    "rejects an empty single secret",
			options: []ConfigOption{WithSecret(" ")},
		},
		{
			name:    "rejects a non-base64 single secret",
			options: []ConfigOption{WithSecret("not-base64!!")},
		},
		{
			name:    "rejects a hex single secret",
			options: []ConfigOption{WithSecret("3c80f410baec8ff24679ee33deeeede9929a788a48ed51d2fa74f5b213d4")},
		},
		{
			name:    "rejects an unpadded single secret",
			options: []ConfigOption{WithSecret(base64.RawStdEncoding.EncodeToString(customSecretOneRaw))},
		},
		{
			name:    "rejects a single secret with the wrong length",
			options: []ConfigOption{WithSecret(base64.StdEncoding.EncodeToString([]byte("too-short")))},
		},
		{
			name: "rejects a non-base64 versioned secret",
			options: []ConfigOption{WithSecrets([]models.SecretConfig{
				{Version: 1, Value: "not-base64!!"},
			})},
		},
		{
			name: "rejects a hex versioned secret",
			options: []ConfigOption{WithSecrets([]models.SecretConfig{
				{Version: 1, Value: "3c80f410baec8ff24679ee33deeeede9929a788a48ed51d2fa74f5b213d4"},
			})},
		},
		{
			name: "rejects an unpadded versioned secret",
			options: []ConfigOption{WithSecrets([]models.SecretConfig{
				{Version: 1, Value: base64.RawStdEncoding.EncodeToString(customSecretOneRaw)},
			})},
		},
		{
			name: "rejects a versioned secret with the wrong length",
			options: []ConfigOption{WithSecrets([]models.SecretConfig{
				{Version: 1, Value: base64.StdEncoding.EncodeToString([]byte("too-short"))},
			})},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyConfigEnv(t, "", nil, tt.envSecrets)

			requirePanics(t, func() {
				_ = NewConfig(tt.options...)
			})
		})
	}
}

func TestNewConfigProductionSecrets(t *testing.T) {
	tests := []struct {
		name       string
		envSecret  *string
		envSecrets *string
		options    []ConfigOption
		wantPanic  bool
		wantSecret string
		want       []models.SecretConfig
	}{
		{
			name:      "rejects the default secret",
			wantPanic: true,
		},
		{
			name:      "rejects a default versioned secret",
			envSecret: new(customSecretOne),
			options: []ConfigOption{
				WithSecret(""),
				WithSecrets([]models.SecretConfig{
					{Version: 1, Value: "  " + defaultSecret + "  "},
				}),
			},
			wantPanic: true,
		},
		{
			name:       "accepts custom single and versioned secrets",
			envSecret:  new(customSecretOne),
			envSecrets: new("2:" + customSecretTwo + ",1:" + customSecretOne),
			options:    []ConfigOption{WithSecret("")},
			wantSecret: customSecretOne,
			want: []models.SecretConfig{
				{Version: 2, Value: customSecretTwo, Key: customSecretTwoRaw},
				{Version: 1, Value: customSecretOne, Key: customSecretOneRaw},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyConfigEnv(t, "production", tt.envSecret, tt.envSecrets)

			if tt.wantPanic {
				requirePanics(t, func() {
					_ = NewConfig(tt.options...)
				})
				return
			}

			config := NewConfig(tt.options...)

			assert.Equal(t, tt.wantSecret, config.Secret)
			assert.Equal(t, tt.want, config.Secrets)
		})
	}
}
