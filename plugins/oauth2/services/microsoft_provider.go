package services

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
)

const defaultMicrosoftUserInfoURL = "https://graph.microsoft.com/v1.0/me"

// MicrosoftProvider implements OAuth2Provider for Microsoft Entra ID (work/school and personal accounts)
type MicrosoftProvider struct {
	*BaseProvider
	userInfoURL string
}

// NewMicrosoftProvider creates a new Microsoft OAuth2 provider.
// cfg.TenantID may be "common" (default when empty), "organizations", "consumers" or a specific tenant ID/domain.
func NewMicrosoftProvider(cfg types.ProviderConfig, opts *ProviderOptions) *MicrosoftProvider {
	defaults := ProviderDefaults{
		Endpoint: endpoints.AzureAD(cfg.TenantID),
		Scopes: []string{
			"openid",
			"profile",
			"email",
			"offline_access",
			"User.Read",
		},
		UserInfoURL: defaultMicrosoftUserInfoURL,
	}

	return &MicrosoftProvider{
		BaseProvider: NewBaseProvider("microsoft", ResolveOAuth2Config(cfg, defaults), opts, cfg.PKCEEnabled()),
		userInfoURL:  ResolveUserInfoURL(cfg, defaultMicrosoftUserInfoURL),
	}
}

// GetUserInfo fetches and normalizes Microsoft Graph user information
func (p *MicrosoftProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*types.UserInfo, error) {
	info, err := FetchUserInfo(ctx, p.opts.httpClient(), p.opts.logger(), token, p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Microsoft user info: %w", err)
	}

	return microsoftUserInfo(info)
}

func microsoftUserInfo(info map[string]any) (*types.UserInfo, error) {
	// Graph "id" is the object ID (oid): stable and unique per user, unlike email or UPN.
	id, err := GetIDField(info, "id")
	if err != nil {
		return nil, fmt.Errorf("invalid Microsoft account id: %w", err)
	}

	// Personal accounts often have no "mail"; their sign-in name is the email address.
	email := GetStringField(info, "mail")
	if email == "" {
		email = GetStringField(info, "userPrincipalName")
	}

	return &types.UserInfo{
		ProviderAccountID: id,
		Email:             util.NormalizeEmail(email),
		// Entra ID lets tenant admins set any mail/UPN without verifying it, so it must never be trusted for account linking.
		EmailVerified: false,
		Name:          GetStringField(info, "displayName"),
		Raw:           info,
	}, nil
}
