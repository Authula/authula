package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"golang.org/x/oauth2"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

func (s *Service) SignInWithOAuth2(ctx context.Context, providerName string, code string, opts ...oauth2.AuthCodeOption) (*SignInResult, error) {
	provider, err := s.OAuth2ProviderRegistry.Get(providerName)
	if err != nil {
		return nil, err
	}

	oauthToken, err := provider.Exchange(ctx, code, opts...)
	if err != nil {
		slog.Error("failed to exchange oauth2 code", "provider", providerName, "error", err)
		return nil, ErrOAuth2ExchangeFailed
	}
	slog.Debug(fmt.Sprintf("Token: %s", oauthToken.AccessToken))

	userInfo, err := provider.GetUserInfo(ctx, oauthToken)
	if err != nil {
		slog.Error("failed to get oauth2 user info", "provider", providerName, "error", err)
		return nil, ErrOAuth2UserInfoFailed
	}

	account, err := s.AccountService.GetAccountByProviderAndAccountID(domain.ProviderType(providerName), userInfo.ID)
	if err != nil {
		return nil, err
	}

	var user *domain.User

	if account != nil {
		user, err = s.UserService.GetUserByID(account.UserID)
		if err != nil {
			return nil, err
		}

		encryptedAccessToken, err := s.TokenService.EncryptToken(oauthToken.AccessToken)
		if err != nil {
			slog.Error("failed to encrypt access token", "error", err)
			return nil, err
		}
		account.AccessToken = &encryptedAccessToken

		if oauthToken.RefreshToken != "" {
			encryptedRefreshToken, err := s.TokenService.EncryptToken(oauthToken.RefreshToken)
			if err != nil {
				slog.Error("failed to encrypt refresh token", "error", err)
				return nil, err
			}
			account.RefreshToken = &encryptedRefreshToken
			account.RefreshTokenExpiresAt = extractRefreshTokenExpiry(oauthToken)
		} else {
			account.RefreshToken = nil
			account.RefreshTokenExpiresAt = nil
		}

		if value, ok := oauthToken.Extra("id_token").(string); ok {
			account.IDToken = &value
		} else {
			account.IDToken = nil
		}
		account.AccessTokenExpiresAt = &oauthToken.Expiry

		if err := s.AccountService.UpdateAccount(account); err != nil {
			slog.Error("failed to update account tokens", "account_id", account.ID, "error", err)
		}
	} else {
		user, err = s.UserService.GetUserByEmail(userInfo.Email)
		if err != nil {
			return nil, err
		}

		if user == nil {
			user = &domain.User{
				Name:          userInfo.Name,
				Email:         userInfo.Email,
				Image:         &userInfo.Picture,
				EmailVerified: userInfo.Verified,
			}
			if err := s.UserService.CreateUser(user); err != nil {
				return nil, err
			}
		} else {
			// User exists but with a different provider.
			// Return error to prevent automatic account linking.
			// TODO: users must use the account linking feature instead.
			return nil, ErrAccountLinkingRequired
		}

		encryptedAccessToken, err := s.TokenService.EncryptToken(oauthToken.AccessToken)
		if err != nil {
			slog.Error("failed to encrypt access token", "error", err)
			return nil, err
		}

		var refreshToken *string
		var refreshTokenExpiresAt *time.Time
		if oauthToken.RefreshToken != "" {
			encrypted, err := s.TokenService.EncryptToken(oauthToken.RefreshToken)
			if err != nil {
				slog.Error("failed to encrypt refresh token", "error", err)
				return nil, err
			}
			refreshToken = &encrypted
			refreshTokenExpiresAt = extractRefreshTokenExpiry(oauthToken)
		}

		account = &domain.Account{
			UserID:                user.ID,
			AccountID:             userInfo.ID,
			ProviderID:            domain.ProviderType(providerName),
			AccessToken:           &encryptedAccessToken,
			RefreshToken:          refreshToken,
			AccessTokenExpiresAt:  &oauthToken.Expiry,
			RefreshTokenExpiresAt: refreshTokenExpiresAt,
		}
		if err := s.AccountService.CreateAccount(account); err != nil {
			return nil, err
		}
	}

	token, err := s.TokenService.GenerateToken()
	if err != nil {
		return nil, err
	}

	_, err = s.SessionService.CreateSession(user.ID, s.TokenService.HashToken(token))
	if err != nil {
		return nil, err
	}

	return &SignInResult{
		Token: token,
		User:  user,
	}, nil
}

// GetValidAccessToken ensures the access token is valid and refreshes it if expired or near expiry.
func (s *Service) GetValidAccessToken(ctx context.Context, account *domain.Account, providerName string) (string, error) {
	// Consider token "expired" if less than 1 minute remains
	const refreshBefore = 1 * time.Minute
	now := time.Now()

	if account.AccessToken == nil || account.AccessTokenExpiresAt == nil || now.After(account.AccessTokenExpiresAt.Add(-refreshBefore)) {
		newToken, err := s.RefreshOAuth2AccessToken(ctx, account, providerName)
		if err != nil {
			return "", err
		}
		return newToken, nil
	}

	accessToken, err := s.TokenService.DecryptToken(*account.AccessToken)
	if err != nil {
		slog.Error("failed to decrypt access token", "account_id", account.ID, "error", err)
		return "", err
	}

	return accessToken, nil
}

// RefreshOAuth2AccessToken refreshes the access token for a given account if a valid refresh token exists.
func (s *Service) RefreshOAuth2AccessToken(ctx context.Context, account *domain.Account, providerName string) (string, error) {
	if account.RefreshToken == nil {
		return "", ErrNoRefreshToken
	}

	refreshToken, err := s.TokenService.DecryptToken(*account.RefreshToken)
	if err != nil {
		slog.Error("failed to decrypt refresh token", "account_id", account.ID, "error", err)
		return "", err
	}

	provider, err := s.OAuth2ProviderRegistry.Get(providerName)
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
		slog.Error("failed to refresh access token", "account_id", account.ID, "error", err)
		return "", err
	}

	encryptedAccessToken, err := s.TokenService.EncryptToken(newToken.AccessToken)
	if err != nil {
		slog.Error("failed to encrypt new access token", "account_id", account.ID, "error", err)
		return "", err
	}
	account.AccessToken = &encryptedAccessToken
	account.AccessTokenExpiresAt = &newToken.Expiry

	if newToken.RefreshToken != "" && newToken.RefreshToken != refreshToken {
		encryptedRefreshToken, err := s.TokenService.EncryptToken(newToken.RefreshToken)
		if err != nil {
			slog.Error("failed to encrypt new refresh token", "account_id", account.ID, "error", err)
			return "", err
		}
		account.RefreshToken = &encryptedRefreshToken
		account.RefreshTokenExpiresAt = extractRefreshTokenExpiry(newToken)
	}

	if err := s.AccountService.UpdateAccount(account); err != nil {
		slog.Error("failed to update account with refreshed tokens", "account_id", account.ID, "error", err)
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
