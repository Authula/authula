package usecases

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
)

// authorizeUseCase handles the authorization flow initiation
type authorizeUseCase struct {
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
	TrustedOrigins   []string
	HMACKey          []byte
}

func NewAuthorizeUseCase(
	registry *services.ProviderRegistry,
	logger models.Logger,
	trustedOrigins []string,
	hmacKey []byte,
) *authorizeUseCase {
	return &authorizeUseCase{
		ProviderRegistry: registry,
		Logger:           logger,
		TrustedOrigins:   trustedOrigins,
		HMACKey:          hmacKey,
	}
}

func (uc *authorizeUseCase) Authorize(ctx context.Context, req *types.AuthorizeRequest) (*AuthorizeResult, error) {
	providerID := strings.ToLower(strings.TrimSpace(req.ProviderID))
	oauthProvider, exists := uc.ProviderRegistry.Get(providerID)
	if !exists {
		return nil, constants.ErrProviderNotFound
	}

	redirectTo, err := services.ValidateRedirectTo(req.RedirectTo, uc.TrustedOrigins)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", constants.ErrInvalidRedirectTo, err)
	}

	state, err := services.GenerateRandomString(32)
	if err != nil {
		uc.Logger.Error("failed to generate oauth state", "error", err)
		return nil, fmt.Errorf("failed to generate state")
	}

	purpose := services.StatePurposeLogin
	if req.ActorID != "" {
		purpose = services.StatePurposeLink
	}

	signedState, err := services.SignState(services.StatePayload{
		State:    state,
		Provider: providerID,
		UserID:   req.ActorID,
		Purpose:  purpose,
	}, uc.HMACKey)
	if err != nil {
		uc.Logger.Error("failed to sign oauth state", "error", err)
		return nil, fmt.Errorf("failed to sign state")
	}

	signedRedirectTo, err := services.SignCookie(redirectTo, uc.HMACKey)
	if err != nil {
		uc.Logger.Error("failed to sign redirect URL", "error", err)
		return nil, fmt.Errorf("failed to sign redirect URL")
	}

	var verifierCookie *string
	authOptions := make([]oauth2.AuthCodeOption, 0, 1)
	if oauthProvider.RequiresPKCE() {
		verifier, err := services.GeneratePKCEVerifier()
		if err != nil {
			uc.Logger.Error("failed to generate PKCE verifier", "error", err)
			return nil, fmt.Errorf("failed to generate PKCE verifier")
		}

		signedVerifier, err := services.SignCookie(verifier, uc.HMACKey)
		if err != nil {
			uc.Logger.Error("failed to sign PKCE verifier", "error", err)
			return nil, fmt.Errorf("failed to sign PKCE verifier")
		}

		verifierCookie = &signedVerifier
		authOptions = append(authOptions, oauth2.S256ChallengeOption(verifier))
	}

	return &AuthorizeResult{
		AuthorizationURL: oauthProvider.GetAuthURL(state, authOptions...),
		StateCookie:      signedState,
		VerifierCookie:   verifierCookie,
		RedirectCookie:   signedRedirectTo,
	}, nil
}
