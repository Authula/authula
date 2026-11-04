package services

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// MicrosoftProvider implements OAuth2Provider for Microsoft Entra ID (work/school and personal accounts)
type MicrosoftProvider struct {
	*BaseProvider
}

// NewMicrosoftProvider creates a new Microsoft OAuth2 provider.
// tenantID may be "common" (default when empty), "organizations", "consumers" or a specific tenant ID/domain.
func NewMicrosoftProvider(clientID, clientSecret, redirectURL, tenantID string) *MicrosoftProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid",
			"profile",
			"email",
			"offline_access",
			"User.Read",
		},
		Endpoint: endpoints.AzureAD(tenantID),
	}

	return &MicrosoftProvider{
		BaseProvider: NewBaseProvider("microsoft", config),
	}
}

// GetUserInfo fetches and normalizes Microsoft Graph user information
func (p *MicrosoftProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, token, "https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Microsoft user info: %w", err)
	}

	return microsoftUserInfo(info), nil
}

func microsoftUserInfo(info map[string]any) *types.UserInfo {
	// Personal accounts often have no "mail"; their sign-in name is the email address.
	email := GetStringField(info, "mail")
	if email == "" {
		email = GetStringField(info, "userPrincipalName")
	}

	return &types.UserInfo{
		// Graph "id" is the object ID (oid): stable and unique per user, unlike email or UPN.
		ProviderAccountID: GetStringField(info, "id"),
		Email:             email,
		// Entra ID lets tenant admins set any mail/UPN without verifying it, so it must never be trusted for account linking.
		EmailVerified: false,
		Name:          GetStringField(info, "displayName"),
		Raw:           info,
	}
}
