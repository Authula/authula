package oauth2

import (
	"context"
	"strconv"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/constants"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	oauth2providers "github.com/GoBetterAuth/go-better-auth/oauth2-providers"
	"golang.org/x/oauth2"
)

type service struct {
	config                 *models.Config
	logger                 models.Logger
	userService            models.UserService
	accountService         models.AccountService
	sessionService         models.SessionService
	tokenService           models.TokenService
	oauth2ProviderRegistry *oauth2providers.OAuth2ProviderRegistry
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	accountService models.AccountService,
	sessionService models.SessionService,
	tokenService models.TokenService,
	oauth2ProviderRegistry *oauth2providers.OAuth2ProviderRegistry,
) *service {
	return &service{
		config:                 config,
		logger:                 logger,
		userService:            userService,
		accountService:         accountService,
		sessionService:         sessionService,
		tokenService:           tokenService,
		oauth2ProviderRegistry: oauth2ProviderRegistry,
	}
}

func (s *service) PrepareOAuth2Login(ctx context.Context, providerName string) (*models.OAuth2LoginResult, error) {
	// Get the provider to verify it exists and check if it requires PKCE
	provider, err := s.oauth2ProviderRegistry.Get(providerName)
	if err != nil {
		return nil, err
	}

	// Generate state token for CSRF protection
	state, err := s.tokenService.GenerateToken()
	if err != nil {
		return nil, err
	}

	var opts []oauth2.AuthCodeOption
	var verifier *string

	// Generate PKCE verifier and challenge if the provider requires PKCE
	if provider.RequiresPKCE() {
		pkceVerifier, challenge, err := util.GeneratePKCE()
		if err != nil {
			return nil, err
		}
		verifier = &pkceVerifier

		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
	}

	// Get the authorization URL
	authURL := provider.GetAuthURL(state, opts...)

	return &models.OAuth2LoginResult{
		AuthURL:  authURL,
		State:    state,
		Verifier: verifier,
	}, nil
}

func (s *service) SignInWithOAuth2(ctx context.Context, providerName string, code string, state string, verifier *string) (*models.SignInResult, error) {
	// Store the state and verifier from the request for use when exchanging the code
	// The state is validated by the handler, but we include the full logic here for completeness
	provider, err := s.oauth2ProviderRegistry.Get(providerName)
	if err != nil {
		return nil, err
	}

	// Build the options for the code exchange
	var opts []oauth2.AuthCodeOption
	if verifier != nil && *verifier != "" {
		// Add the PKCE code verifier if provided
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", *verifier))
	}

	// Exchange the authorization code for tokens
	oauthToken, err := provider.Exchange(ctx, code, opts...)
	if err != nil {
		s.logger.Error("failed to exchange oauth2 code", "provider", providerName, "error", err)
		return nil, constants.ErrOAuth2ExchangeFailed
	}

	// Get user info from the OAuth2 provider
	userInfo, err := provider.GetUserInfo(ctx, oauthToken)
	if err != nil {
		s.logger.Error("failed to get oauth2 user info", "provider", providerName, "error", err)
		return nil, constants.ErrOAuth2UserInfoFailed
	}

	// Check if an account already exists for this provider and user
	account, err := s.accountService.GetAccountByProviderAndAccountID(models.ProviderType(providerName), userInfo.ID)
	if err != nil {
		return nil, err
	}

	var user *models.User

	if account != nil {
		// Account exists, update the user and tokens
		user, err = s.userService.GetUserByID(account.UserID)
		if err != nil {
			return nil, err
		}

		// Encrypt and store the new access token
		encryptedAccessToken, err := s.tokenService.EncryptToken(oauthToken.AccessToken)
		if err != nil {
			s.logger.Error("failed to encrypt access token", "error", err)
			return nil, err
		}
		account.AccessToken = &encryptedAccessToken

		// Handle refresh token if provided
		if oauthToken.RefreshToken != "" {
			encryptedRefreshToken, err := s.tokenService.EncryptToken(oauthToken.RefreshToken)
			if err != nil {
				s.logger.Error("failed to encrypt refresh token", "error", err)
				return nil, err
			}
			account.RefreshToken = &encryptedRefreshToken
			account.RefreshTokenExpiresAt = extractRefreshTokenExpiry(oauthToken)
		} else {
			account.RefreshToken = nil
			account.RefreshTokenExpiresAt = nil
		}

		// Handle ID token if provided
		if value, ok := oauthToken.Extra("id_token").(string); ok {
			account.IDToken = &value
		} else {
			account.IDToken = nil
		}
		account.AccessTokenExpiresAt = &oauthToken.Expiry

		// Update the account with new tokens
		if err := s.accountService.UpdateAccount(account); err != nil {
			s.logger.Error("failed to update account tokens", "account_id", account.ID, "error", err)
		}
	} else {
		// Account doesn't exist, create a new user and account
		user, err = s.userService.GetUserByEmail(userInfo.Email)
		if err != nil {
			return nil, err
		}

		if user == nil {
			// Create a new user with info from the OAuth2 provider
			user = &models.User{
				Name:          userInfo.Name,
				Email:         userInfo.Email,
				Image:         &userInfo.Picture,
				EmailVerified: userInfo.Verified,
			}
			if err := s.userService.CreateUser(user); err != nil {
				return nil, err
			}
		} else {
			// User exists with this email but no OAuth2 account
			// Return error to prevent automatic account linking
			// TODO: users must use the account linking feature instead
			return nil, constants.ErrAccountLinkingRequired
		}

		// Encrypt the access token
		encryptedAccessToken, err := s.tokenService.EncryptToken(oauthToken.AccessToken)
		if err != nil {
			s.logger.Error("failed to encrypt access token", "error", err)
			return nil, err
		}

		// Handle refresh token if provided
		var refreshToken *string
		var refreshTokenExpiresAt *time.Time
		if oauthToken.RefreshToken != "" {
			encrypted, err := s.tokenService.EncryptToken(oauthToken.RefreshToken)
			if err != nil {
				s.logger.Error("failed to encrypt refresh token", "error", err)
				return nil, err
			}
			refreshToken = &encrypted
			refreshTokenExpiresAt = extractRefreshTokenExpiry(oauthToken)
		}

		// Create the new account
		account = &models.Account{
			UserID:                user.ID,
			AccountID:             userInfo.ID,
			ProviderID:            models.ProviderType(providerName),
			AccessToken:           &encryptedAccessToken,
			RefreshToken:          refreshToken,
			AccessTokenExpiresAt:  &oauthToken.Expiry,
			RefreshTokenExpiresAt: refreshTokenExpiresAt,
		}
		if err := s.accountService.CreateAccount(account); err != nil {
			return nil, err
		}
	}

	// Generate session token
	sessionToken, err := s.tokenService.GenerateToken()
	if err != nil {
		return nil, err
	}

	// Create session
	_, err = s.sessionService.CreateSession(user.ID, s.tokenService.HashToken(sessionToken))
	if err != nil {
		return nil, err
	}

	// Generate CSRF token if enabled
	var csrfToken *string
	if s.config.CSRF.Enabled {
		csrf, err := s.tokenService.GenerateToken()
		if err != nil {
			return nil, err
		}
		csrfToken = &csrf
	}

	return &models.SignInResult{
		Token:     sessionToken,
		User:      user,
		CSRFToken: csrfToken,
	}, nil
}

// GetValidAccessToken ensures the access token is valid and refreshes it if expired or near expiry.
func (s *service) GetValidAccessToken(ctx context.Context, account *models.Account, providerName string) (string, error) {
	// Consider token "expired" if less than 1 minute remains
	const refreshBefore = 1 * time.Minute
	now := time.Now()

	if account.AccessToken == nil || account.AccessTokenExpiresAt == nil || now.After(account.AccessTokenExpiresAt.Add(-refreshBefore)) {
		newToken, err := s.refreshOAuth2AccessToken(ctx, account, providerName)
		if err != nil {
			return "", err
		}
		return newToken, nil
	}

	accessToken, err := s.tokenService.DecryptToken(*account.AccessToken)
	if err != nil {
		s.logger.Error("failed to decrypt access token", "account_id", account.ID, "error", err)
		return "", err
	}

	return accessToken, nil
}

// RefreshOAuth2AccessToken refreshes the access token for a given account if a valid refresh token exists.
func (s *service) refreshOAuth2AccessToken(ctx context.Context, account *models.Account, providerName string) (string, error) {
	if account.RefreshToken == nil {
		return "", constants.ErrNoRefreshToken
	}

	refreshToken, err := s.tokenService.DecryptToken(*account.RefreshToken)
	if err != nil {
		s.logger.Error("failed to decrypt refresh token", "account_id", account.ID, "error", err)
		return "", err
	}

	provider, err := s.oauth2ProviderRegistry.Get(providerName)
	if err != nil {
		return "", err
	}

	t := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now(),
	}

	tokenSource := provider.GetConfig().TokenSource(ctx, t)
	newToken, err := tokenSource.Token()
	if err != nil {
		s.logger.Error("failed to refresh access token", "account_id", account.ID, "error", err)
		return "", err
	}

	encryptedAccessToken, err := s.tokenService.EncryptToken(newToken.AccessToken)
	if err != nil {
		s.logger.Error("failed to encrypt new access token", "account_id", account.ID, "error", err)
		return "", err
	}
	account.AccessToken = &encryptedAccessToken
	account.AccessTokenExpiresAt = &newToken.Expiry

	if newToken.RefreshToken != "" && newToken.RefreshToken != refreshToken {
		encryptedRefreshToken, err := s.tokenService.EncryptToken(newToken.RefreshToken)
		if err != nil {
			s.logger.Error("failed to encrypt new refresh token", "account_id", account.ID, "error", err)
			return "", err
		}
		account.RefreshToken = &encryptedRefreshToken
		account.RefreshTokenExpiresAt = extractRefreshTokenExpiry(newToken)
	}

	if err := s.accountService.UpdateAccount(account); err != nil {
		s.logger.Error("failed to update account with refreshed tokens", "account_id", account.ID, "error", err)
		return "", err
	}

	return newToken.AccessToken, nil
}

func extractRefreshTokenExpiry(token *oauth2.Token) *time.Time {
	if token == nil || token.RefreshToken == "" {
		return nil
	}

	// Try converting to float64 first
	if value, ok := token.Extra("refresh_token_expires_in").(float64); ok && value > 0 {
		t := time.Now().UTC().Add(time.Duration(value) * time.Second)
		return &t
	}

	// Else try string format
	if value, ok := token.Extra("refresh_token_expires_in").(string); ok && value != "" {
		if val, err := strconv.ParseInt(value, 10, 64); err == nil && val > 0 {
			t := time.Now().UTC().Add(time.Duration(val) * time.Second)
			return &t
		}
	}

	return nil
}
