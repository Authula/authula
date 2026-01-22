package usecases

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/services"
	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
)

// AuthorizeUseCase handles the authorization flow initiation
type AuthorizeUseCase struct {
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
	TrustedOrigins   []string
	HMACKey          []byte
}

// NewAuthorizeUseCase creates a new authorize usecase
func NewAuthorizeUseCase(
	registry *services.ProviderRegistry,
	logger models.Logger,
	trustedOrigins []string,
	hmacKey []byte,
) *AuthorizeUseCase {
	return &AuthorizeUseCase{
		ProviderRegistry: registry,
		Logger:           logger,
		TrustedOrigins:   trustedOrigins,
		HMACKey:          hmacKey,
	}
}

// Authorize initiates an OAuth2 authorization flow
func (uc *AuthorizeUseCase) Authorize(ctx context.Context, req *types.AuthorizeRequest) (*AuthorizeResult, error) {
	oauthProvider, exists := uc.ProviderRegistry.Get(req.ProviderID)
	if !exists {
		return nil, fmt.Errorf("provider %s not found", req.ProviderID)
	}

	state, err := services.GenerateRandomString(32)
	if err != nil {
		uc.Logger.Error(fmt.Sprintf("Failed to generate state: %v", err))
		return nil, fmt.Errorf("failed to generate state")
	}

	signedState, err := services.SignCookie(state, uc.HMACKey)
	if err != nil {
		uc.Logger.Error(fmt.Sprintf("Failed to sign state: %v", err))
		return nil, fmt.Errorf("failed to sign state")
	}

	redirectTo := req.RedirectTo
	if redirectTo == "" {
		redirectTo = "/"
	}

	signedRedirectTo, err := services.SignCookie(redirectTo, uc.HMACKey)
	if err != nil {
		uc.Logger.Error(fmt.Sprintf("Failed to sign redirect URL: %v", err))
		return nil, fmt.Errorf("failed to sign redirect URL")
	}

	var verifierCookie *string
	if oauthProvider.RequiresPKCE() {
		verifier, _, err := services.GeneratePKCEChallenge()
		if err != nil {
			uc.Logger.Error(fmt.Sprintf("Failed to generate PKCE challenge: %v", err))
			return nil, fmt.Errorf("failed to generate PKCE challenge")
		}

		signedVerifier, err := services.SignCookie(verifier, uc.HMACKey)
		if err != nil {
			uc.Logger.Error(fmt.Sprintf("Failed to sign verifier: %v", err))
			return nil, fmt.Errorf("failed to sign verifier")
		}

		verifierCookie = &signedVerifier
	}

	authURL := oauthProvider.GetAuthURL(state)

	return &AuthorizeResult{
		AuthorizationURL: authURL,
		StateCookie:      signedState,
		VerifierCookie:   verifierCookie,
		RedirectCookie:   signedRedirectTo,
	}, nil
}
