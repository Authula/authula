package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
			info:      map[string]any{"id": "oid-1", "mail": "alice@contoso.com", "userPrincipalName": "alice@contoso.onmicrosoft.com", "displayName": "Alice"},
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

			got := microsoftUserInfo(tt.info)
			require.Equal(t, "oid-1", got.ProviderAccountID)
			require.Equal(t, tt.wantEmail, got.Email)
			require.Equal(t, "Alice", got.Name)
			require.False(t, got.EmailVerified, "Entra ID emails are never verified")
		})
	}
}

func TestNewMicrosoftProvider_TenantEndpoint(t *testing.T) {
	t.Parallel()

	require.True(t, strings.HasPrefix(NewMicrosoftProvider("id", "secret", "https://app/cb", "").GetConfig().Endpoint.AuthURL,
		"https://login.microsoftonline.com/common/"))
	require.True(t, strings.HasPrefix(NewMicrosoftProvider("id", "secret", "https://app/cb", "organizations").GetConfig().Endpoint.TokenURL,
		"https://login.microsoftonline.com/organizations/"))
}
