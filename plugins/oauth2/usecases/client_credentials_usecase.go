package usecases

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	rootservices "github.com/Authula/authula/services"
)

type clientCredentialsUseCase struct {
	clients       services.ClientService
	issuer        rootservices.ClientCredentialsTokenIssuer
	allowedScopes []string
	ttl           time.Duration
	logger        models.Logger
}

// NewClientCredentialsUseCase returns the OAuth 2.0 client credentials grant use
// case. allowedScopes is the deployment-wide scope ceiling.
func NewClientCredentialsUseCase(
	clients services.ClientService,
	issuer rootservices.ClientCredentialsTokenIssuer,
	allowedScopes []string,
	ttl time.Duration,
	logger models.Logger,
) ClientCredentialsUseCase {
	return &clientCredentialsUseCase{
		clients:       clients,
		issuer:        issuer,
		allowedScopes: allowedScopes,
		ttl:           ttl,
		logger:        logger,
	}
}

func (u *clientCredentialsUseCase) IssueToken(ctx context.Context, req *types.TokenRequest) (*types.TokenResult, error) {
	client, err := u.clients.Authenticate(ctx, req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(client.AllowedGrants, types.OAuthGrantTypeClientCredentials) {
		return nil, constants.ErrUnauthorizedClient
	}

	scopes, err := u.resolveScopes(client, req.Scope)
	if err != nil {
		return nil, err
	}

	token, err := u.issuer.IssueClientCredentialsToken(ctx, rootservices.ClientCredentialsTokenRequest{
		ClientID:       client.ClientID,
		OrganizationID: organizationIDForClient(client),
		Scopes:         scopes,
		ExpiresIn:      u.ttl,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", constants.ErrTokenIssuanceFailed, err)
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	return &types.TokenResult{
		AccessToken: token.AccessToken,
		TokenType:   tokenType,
		ExpiresIn:   token.ExpiresIn,
		Scopes:      scopes,
	}, nil
}

func (u *clientCredentialsUseCase) resolveScopes(client *types.OAuthClient, raw *string) ([]string, error) {
	if clientHasWildcardScope(client.AllowedScopes) {
		u.logger.Warn(
			"oauth client allowed_scopes contains a wildcard; machine tokens never carry wildcards",
			"client_id", client.ClientID,
		)
	}

	if raw == nil {
		return []string{}, nil
	}

	requested, ok := types.ParseScope(*raw)
	if !ok || len(requested) == 0 {
		return nil, constants.ErrInvalidScope
	}

	for _, scope := range requested {
		if types.IsOIDCScope(scope) {
			return nil, constants.ErrInvalidScope
		}
	}

	if !types.ScopeSubset(requested, client.AllowedScopes) {
		return nil, constants.ErrInvalidScope
	}
	if !types.ScopeSubset(requested, u.allowedScopes) {
		return nil, constants.ErrInvalidScope
	}

	return requested, nil
}

func clientHasWildcardScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.Contains(scope, "*") {
			return true
		}
	}
	return false
}

func organizationIDForClient(client *types.OAuthClient) string {
	if client.OrganizationID != nil && *client.OrganizationID != "" {
		return *client.OrganizationID
	}
	if client.OwnerType == types.OAuthClientOwnerTypeOrganization {
		return client.OwnerID
	}
	return ""
}
