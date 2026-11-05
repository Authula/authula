package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/oauth2/types"
)

func TestMicrosoftUserInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		info      map[string]any
		wantEmail string
	}{
		{
			name:      "uses mail when present",
			info:      map[string]any{"id": "oid-1", "mail": "Alice@Contoso.com", "userPrincipalName": "alice@contoso.onmicrosoft.com", "displayName": "Alice"},
			wantEmail: "alice@contoso.com",
		},
		{
			name:      "falls back to userPrincipalName for personal accounts",
			info:      map[string]any{"id": "oid-1", "mail": nil, "userPrincipalName": "alice@outlook.com", "displayName": "Alice"},
			wantEmail: "alice@outlook.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := microsoftUserInfo(tt.info)
			require.NoError(t, err)
			require.Equal(t, "oid-1", got.ProviderAccountID)
			require.Equal(t, tt.wantEmail, got.Email)
			require.Equal(t, "Alice", got.Name)
			require.False(t, got.EmailVerified, "Entra ID emails are never verified")
		})
	}
}

func TestMicrosoftUserInfo_MissingID(t *testing.T) {
	t.Parallel()

	_, err := microsoftUserInfo(map[string]any{"mail": "alice@contoso.com"})
	require.Error(t, err)
}

func TestNewMicrosoftProvider_TenantEndpoint(t *testing.T) {
	t.Parallel()

	cfg := types.ProviderConfig{ClientID: "id", ClientSecret: "secret", RedirectURL: "https://app/cb"}
	require.True(t, strings.HasPrefix(NewMicrosoftProvider(cfg, nil).GetConfig().Endpoint.AuthURL,
		"https://login.microsoftonline.com/common/"))

	cfg.TenantID = "organizations"
	require.True(t, strings.HasPrefix(NewMicrosoftProvider(cfg, nil).GetConfig().Endpoint.TokenURL,
		"https://login.microsoftonline.com/organizations/"))
	require.True(t, NewMicrosoftProvider(cfg, nil).RequiresPKCE())
}
