package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptionKeys(t *testing.T) {
	t.Parallel()

	secretKey := []byte("secret-key")
	primaryKey := []byte("primary-key")
	olderKey := []byte("older-key")

	tests := []struct {
		name           string
		config         *Config
		wantCurrent    []byte
		wantDecryptors [][]byte
	}{
		{
			name:           "empty config has no keys",
			config:         &Config{},
			wantCurrent:    nil,
			wantDecryptors: nil,
		},
		{
			name:           "single secret is the current key",
			config:         &Config{SecretKey: secretKey},
			wantCurrent:    secretKey,
			wantDecryptors: nil,
		},
		{
			name: "first versioned secret is current and all secrets are decryptors",
			config: &Config{
				Secrets: []SecretConfig{
					{Version: 2, Key: primaryKey},
					{Version: 1, Key: olderKey},
				},
			},
			wantCurrent:    primaryKey,
			wantDecryptors: [][]byte{primaryKey, olderKey},
		},
		{
			name: "single secret is appended as an additional decryptor",
			config: &Config{
				SecretKey: secretKey,
				Secrets: []SecretConfig{
					{Version: 2, Key: primaryKey},
					{Version: 1, Key: olderKey},
				},
			},
			wantCurrent:    primaryKey,
			wantDecryptors: [][]byte{primaryKey, olderKey, secretKey},
		},
		{
			name: "versioned secrets stay usable when the single secret is empty",
			config: &Config{
				Secrets: []SecretConfig{
					{Version: 1, Key: primaryKey},
				},
			},
			wantCurrent:    primaryKey,
			wantDecryptors: [][]byte{primaryKey},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			current, decryptors := tt.config.EncryptionKeys()

			require.Equal(t, tt.wantCurrent, current)
			require.Equal(t, tt.wantDecryptors, decryptors)
		})
	}
}

func TestServiceHooks(t *testing.T) {
	t.Parallel()

	user := &User{ID: "user-1"}

	tests := []struct {
		name     string
		register func(hooks *ServiceHooks[User], fn ServiceHook[User])
		list     func(hooks *ServiceHooks[User]) []ServiceHook[User]
	}{
		{
			name:     "before create",
			register: (*ServiceHooks[User]).RegisterBeforeCreate,
			list:     (*ServiceHooks[User]).BeforeCreateHooks,
		},
		{
			name:     "after create",
			register: (*ServiceHooks[User]).RegisterAfterCreate,
			list:     (*ServiceHooks[User]).AfterCreateHooks,
		},
		{
			name:     "before update",
			register: (*ServiceHooks[User]).RegisterBeforeUpdate,
			list:     (*ServiceHooks[User]).BeforeUpdateHooks,
		},
		{
			name:     "after update",
			register: (*ServiceHooks[User]).RegisterAfterUpdate,
			list:     (*ServiceHooks[User]).AfterUpdateHooks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hooks := &ServiceHooks[User]{}
			firstCalled := false
			secondCalled := false

			tt.register(hooks, func(hookUser *User) error {
				firstCalled = true
				require.Same(t, user, hookUser)
				return nil
			})
			tt.register(hooks, func(*User) error {
				secondCalled = true
				return errors.New("second hook failed")
			})

			registered := tt.list(hooks)
			require.Len(t, registered, 2)

			require.NoError(t, registered[0](user))
			require.Error(t, registered[1](user))
			require.True(t, firstCalled)
			require.True(t, secondCalled)
		})
	}
}

func TestServiceHooksStartEmpty(t *testing.T) {
	t.Parallel()

	hooks := &ServiceHooks[User]{}

	require.Empty(t, hooks.BeforeCreateHooks())
	require.Empty(t, hooks.AfterCreateHooks())
	require.Empty(t, hooks.BeforeUpdateHooks())
	require.Empty(t, hooks.AfterUpdateHooks())
}
