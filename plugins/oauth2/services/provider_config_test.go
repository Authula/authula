package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestResolveAuthURLParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      types.ProviderConfig
		defaults ProviderDefaults
		want     map[string]string
	}{
		{
			name:     "provider defaults are applied",
			defaults: ProviderDefaults{AuthURLParams: map[string]string{"prompt": "consent"}},
			want:     map[string]string{"prompt": "consent"},
		},
		{
			name:     "operator configuration overrides defaults",
			cfg:      types.ProviderConfig{AuthURLParams: map[string]string{"prompt": "login"}},
			defaults: ProviderDefaults{AuthURLParams: map[string]string{"prompt": "consent"}},
			want:     map[string]string{"prompt": "login"},
		},
		{
			name:     "empty operator value removes a default",
			cfg:      types.ProviderConfig{AuthURLParams: map[string]string{"prompt": ""}},
			defaults: ProviderDefaults{AuthURLParams: map[string]string{"prompt": "consent"}},
			want:     map[string]string{},
		},
		{
			name:     "whitespace operator value removes a default",
			cfg:      types.ProviderConfig{AuthURLParams: map[string]string{"prompt": "   "}},
			defaults: ProviderDefaults{AuthURLParams: map[string]string{"prompt": "consent"}},
			want:     map[string]string{},
		},
		{
			name:     "reserved defaults are filtered",
			defaults: ProviderDefaults{AuthURLParams: map[string]string{"state": "attacker", "prompt": "consent"}},
			want:     map[string]string{"prompt": "consent"},
		},
		{
			name:     "reserved configuration is filtered",
			cfg:      types.ProviderConfig{AuthURLParams: map[string]string{"State": "attacker", "prompt": "consent"}},
			defaults: ProviderDefaults{},
			want:     map[string]string{"prompt": "consent"},
		},
		{
			name:     "blank configuration keys are filtered",
			cfg:      types.ProviderConfig{AuthURLParams: map[string]string{"": "value", "   ": "value", "prompt": "consent"}},
			defaults: ProviderDefaults{},
			want:     map[string]string{"prompt": "consent"},
		},
		{
			name: "nil maps resolve to an empty map",
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolved := ResolveAuthURLParams(tt.cfg, tt.defaults)

			require.NotNil(t, resolved)
			require.Equal(t, tt.want, resolved)
		})
	}
}

func TestResolveAuthURLParamsDoesNotMutateDefaults(t *testing.T) {
	t.Parallel()

	defaults := ProviderDefaults{
		AuthURLParams: map[string]string{"prompt": "consent", "access_type": "offline"},
	}
	cfg := types.ProviderConfig{
		AuthURLParams: map[string]string{"prompt": "login", "access_type": ""},
	}

	_ = ResolveAuthURLParams(cfg, defaults)

	require.Equal(t, map[string]string{"prompt": "consent", "access_type": "offline"}, defaults.AuthURLParams)
}

func TestValidateAuthURLParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
	}{
		{
			name:   "accepts ordinary provider specific keys",
			params: map[string]string{"prompt": "consent", "access_type": "offline", "owner": "user"},
		},
		{
			name:   "accepts nil and empty maps",
			params: nil,
		},
		{
			name:    "rejects a blank name",
			params:  map[string]string{"": "value"},
			wantErr: true,
		},
		{
			name:    "rejects a whitespace only name",
			params:  map[string]string{"   ": "value"},
			wantErr: true,
		},
		{
			name:    "rejects response_type",
			params:  map[string]string{"response_type": "token"},
			wantErr: true,
		},
		{
			name:    "rejects client_id",
			params:  map[string]string{"client_id": "other"},
			wantErr: true,
		},
		{
			name:    "rejects redirect_uri",
			params:  map[string]string{"redirect_uri": "https://evil.example.com"},
			wantErr: true,
		},
		{
			name:    "rejects state",
			params:  map[string]string{"state": "fixed"},
			wantErr: true,
		},
		{
			name:    "rejects code_challenge",
			params:  map[string]string{"code_challenge": "challenge"},
			wantErr: true,
		},
		{
			name:    "rejects code_challenge_method",
			params:  map[string]string{"code_challenge_method": "plain"},
			wantErr: true,
		},
		{
			name:    "rejects scope",
			params:  map[string]string{"scope": "openid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAuthURLParams(tt.params)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
