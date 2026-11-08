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

var (
	customSecretOne = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	customSecretTwo = base64.StdEncoding.EncodeToString([]byte("fedcba9876543210fedcba9876543210"))
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

func requirePanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected NewConfig to panic")
		}
	}()

	fn()
}

func TestNewConfigSecretsFromOption(t *testing.T) {
	unsetEnv(t, env.EnvSecrets)

	want := []models.SecretConfig{
		{Version: 2, Value: customSecretTwo},
		{Version: 1, Value: customSecretOne},
	}

	config := NewConfig(WithSecrets(want))

	assert.Equal(t, want, config.Secrets)
}

func TestNewConfigSecretsFromEnv(t *testing.T) {
	t.Setenv(env.EnvSecrets, " 2:"+customSecretTwo+" , 1:"+customSecretOne+" ")

	config := NewConfig()

	assert.Equal(t, []models.SecretConfig{
		{Version: 2, Value: customSecretTwo},
		{Version: 1, Value: customSecretOne},
	}, config.Secrets)
}

func TestNewConfigSecretsEnvOverridesTOML(t *testing.T) {
	t.Setenv(env.EnvSecrets, "3:"+customSecretTwo)

	config := NewConfig(WithSecrets([]models.SecretConfig{
		{Version: 1, Value: customSecretOne},
	}))

	assert.Equal(t, []models.SecretConfig{
		{Version: 3, Value: customSecretTwo},
	}, config.Secrets)
}

func TestNewConfigSecretsEnvEmptyClearsTOML(t *testing.T) {
	t.Setenv(env.EnvSecrets, "")

	config := NewConfig(WithSecrets([]models.SecretConfig{
		{Version: 1, Value: customSecretOne},
	}))

	assert.Empty(t, config.Secrets)
}

func TestNewConfigSecretsEnvUnsetKeepsTOML(t *testing.T) {
	unsetEnv(t, env.EnvSecrets)

	want := []models.SecretConfig{{Version: 1, Value: customSecretOne}}
	config := NewConfig(WithSecrets(want))

	assert.Equal(t, want, config.Secrets)
}

func TestNewConfigSecretsEnvValidationPanics(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "zero version", raw: "0:" + customSecretOne},
		{name: "negative version", raw: "-1:" + customSecretOne},
		{name: "non-numeric version", raw: "abc:" + customSecretOne},
		{name: "duplicate version", raw: "1:" + customSecretOne + ",1:" + customSecretTwo},
		{name: "empty value", raw: "1:"},
		{name: "missing colon", raw: "1"},
		{name: "empty entry", raw: "1:" + customSecretOne + ","},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(env.EnvSecrets, tt.raw)
			requirePanics(t, func() {
				_ = NewConfig()
			})
		})
	}
}

func TestNewConfigSecretsAllowsVersionGaps(t *testing.T) {
	t.Setenv(env.EnvSecrets, "1:"+customSecretOne+",3:"+customSecretTwo)

	config := NewConfig()

	require.Len(t, config.Secrets, 2)
	assert.Equal(t, 1, config.Secrets[0].Version)
	assert.Equal(t, 3, config.Secrets[1].Version)
}

func TestNewConfigSecretsFirstEntryIsCurrent(t *testing.T) {
	t.Setenv(env.EnvSecrets, "7:"+customSecretOne+",2:"+customSecretTwo)

	config := NewConfig()

	require.NotEmpty(t, config.Secrets)
	assert.Equal(t, models.SecretConfig{Version: 7, Value: customSecretOne}, config.Secrets[0])
}

func TestNewConfigTrimsSecret(t *testing.T) {
	unsetEnv(t, env.EnvSecret)

	config := NewConfig(WithSecret("  " + customSecretOne + "\n"))

	assert.Equal(t, customSecretOne, config.Secret)
}

func TestNewConfigProductionRejectsDefaultSecretWithSecrets(t *testing.T) {
	t.Setenv(env.EnvGoEnvironment, "production")
	unsetEnv(t, env.EnvSecret)
	t.Setenv(env.EnvSecrets, "1:"+customSecretOne)

	requirePanics(t, func() {
		_ = NewConfig()
	})
}

func TestNewConfigProductionRejectsDefaultVersionedSecret(t *testing.T) {
	t.Setenv(env.EnvGoEnvironment, "production")
	t.Setenv(env.EnvSecret, customSecretOne)
	unsetEnv(t, env.EnvSecrets)

	requirePanics(t, func() {
		_ = NewConfig(WithSecrets([]models.SecretConfig{
			{Version: 1, Value: "  " + defaultSecret + "  "},
		}))
	})
}

func TestNewConfigProductionAcceptsCustomSecrets(t *testing.T) {
	t.Setenv(env.EnvGoEnvironment, "production")
	t.Setenv(env.EnvSecret, customSecretOne)
	t.Setenv(env.EnvSecrets, "2:"+customSecretTwo+",1:"+customSecretOne)

	config := NewConfig(WithSecret(""))

	assert.Equal(t, customSecretOne, config.Secret)
	assert.Equal(t, []models.SecretConfig{
		{Version: 2, Value: customSecretTwo},
		{Version: 1, Value: customSecretOne},
	}, config.Secrets)
}

func TestNewConfigNonProductionUsesDefaultSecretFallback(t *testing.T) {
	unsetEnv(t, env.EnvGoEnvironment)
	unsetEnv(t, env.EnvSecret)
	unsetEnv(t, env.EnvSecrets)

	config := NewConfig()

	assert.Equal(t, defaultSecret, config.Secret)
	assert.Empty(t, config.Secrets)

	raw, err := base64.StdEncoding.DecodeString(config.Secret)
	require.NoError(t, err)
	assert.Len(t, raw, 32)
}
