package usecases

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

// authorizeUseCase handles the authorization flow initiation
type authorizeUseCase struct {
	ProviderRegistry *services.ProviderRegistry
	Logger           models.Logger
	TrustedOrigins   []string
	FlowStateRepo    repositories.FlowStateRepository
	TokenService     rootservices.TokenService
	FlowStateTTL     time.Duration
}

func NewAuthorizeUseCase(
	registry *services.ProviderRegistry,
	logger models.Logger,
	trustedOrigins []string,
	flowStateRepo repositories.FlowStateRepository,
	tokenService rootservices.TokenService,
	flowStateTTL time.Duration,
) *authorizeUseCase {
	return &authorizeUseCase{
		ProviderRegistry: registry,
		Logger:           logger,
		TrustedOrigins:   trustedOrigins,
		FlowStateRepo:    flowStateRepo,
		TokenService:     tokenService,
		FlowStateTTL:     flowStateTTL,
	}
}

func (uc *authorizeUseCase) Authorize(ctx context.Context, req *types.AuthorizeRequest) (*AuthorizeResult, error) {
	providerID := strings.ToLower(strings.TrimSpace(req.ProviderID))
	oauthProvider, exists := uc.ProviderRegistry.Get(providerID)
	if !exists {
		return nil, constants.ErrProviderNotFound
	}

	var redirectTo *string
	if strings.TrimSpace(req.RedirectTo) != "" {
		normalized, err := util.ValidateRedirectTo(req.RedirectTo, uc.TrustedOrigins)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", constants.ErrInvalidRedirectTo, err)
		}
		redirectTo = &normalized
	}

	state, err := util.GenerateRandomString(32)
	if err != nil {
		uc.Logger.Error("failed to generate oauth state", "error", err)
		return nil, fmt.Errorf("failed to generate state")
	}

	binding, err := util.GenerateRandomString(32)
	if err != nil {
		uc.Logger.Error("failed to generate oauth binding", "error", err)
		return nil, fmt.Errorf("failed to generate binding")
	}

	purpose := types.OAuthFlowPurposeLogin
	var userID *string
	if req.ActorID != "" {
		purpose = types.OAuthFlowPurposeLink
		userID = &req.ActorID
	}

	var encryptedVerifier *string
	authOptions := make([]oauth2.AuthCodeOption, 0, 1)
	if oauthProvider.RequiresPKCE() {
		verifier, err := util.GenerateRandomString(43)
		if err != nil {
			uc.Logger.Error("failed to generate PKCE verifier", "error", err)
			return nil, fmt.Errorf("failed to generate PKCE verifier")
		}

		encrypted, err := uc.TokenService.Encrypt(verifier)
		if err != nil {
			uc.Logger.Error("failed to encrypt PKCE verifier", "error", err)
			return nil, fmt.Errorf("failed to encrypt PKCE verifier")
		}

		encryptedVerifier = &encrypted
		authOptions = append(authOptions, oauth2.S256ChallengeOption(verifier))
	}

	authorizationURL := oauthProvider.GetAuthURL(state, authOptions...)
	if err := validateAuthorizationURL(authorizationURL, oauthProvider.RequiresPKCE()); err != nil {
		uc.Logger.Error("oauth provider generated an unusable authorization URL", "provider", providerID, "error", err)
		return nil, err
	}

	now := time.Now().UTC()
	flowState := &types.OAuthFlowState{
		ID:                    util.GenerateUUID(),
		StateHash:             util.HashWithSalt(state, binding),
		Provider:              providerID,
		Purpose:               purpose,
		UserID:                userID,
		RedirectTo:            redirectTo,
		CodeVerifierEncrypted: encryptedVerifier,
		ExpiresAt:             now.Add(uc.FlowStateTTL),
	}

	if _, err := uc.FlowStateRepo.Create(ctx, flowState); err != nil {
		uc.Logger.Error("failed to persist oauth flow state", "provider", providerID, "error", err)
		return nil, fmt.Errorf("failed to persist oauth flow state: %w", err)
	}

	return &AuthorizeResult{
		AuthorizationURL: authorizationURL,
		Binding:          binding,
	}, nil
}

// validateAuthorizationURL asserts the OAuth 2.1 required parameters and rejects
// implicit or malformed authorization requests.
func validateAuthorizationURL(raw string, pkceRequired bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%w: invalid authorization URL", constants.ErrProviderFailure)
	}

	query := parsed.Query()
	if query.Get("response_type") != "code" {
		return fmt.Errorf("%w: authorization URL must use response_type=code", constants.ErrProviderFailure)
	}
	if !pkceRequired {
		return nil
	}
	if query.Get("code_challenge") == "" {
		return constants.ErrPKCEChallengeMissing
	}
	if query.Get("code_challenge_method") != "S256" {
		return constants.ErrPKCEChallengeMissing
	}
	return nil
}
