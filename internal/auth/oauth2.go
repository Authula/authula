package auth

import (
	"context"
	"log/slog"

	"golang.org/x/oauth2"

	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

func (s *Service) GetOAuth2AuthURL(providerName string, state string, opts ...oauth2.AuthCodeOption) (string, error) {
	provider, err := s.OAuth2ProviderRegistry.Get(providerName)
	if err != nil {
		return "", err
	}
	return provider.GetAuthURL(state, opts...), nil
}

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

	userInfo, err := s.getOAuth2UserInfo(ctx, providerName, oauthToken)
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

		slog.Debug("AccessToken: %s, RefreshToken: %s", oauthToken.AccessToken, oauthToken.RefreshToken)
		encryptedAccessToken, err := s.TokenService.EncryptToken(oauthToken.AccessToken)
		if err != nil {
			slog.Error("failed to encrypt access token", "error", err)
			return nil, err
		}
		encryptedRefreshToken, err := s.TokenService.EncryptToken(oauthToken.RefreshToken)
		if err != nil {
			slog.Error("failed to encrypt refresh token", "error", err)
			return nil, err
		}
		account.AccessToken = &encryptedAccessToken
		account.RefreshToken = &encryptedRefreshToken
		// account.IDToken = ... // Extract ID token if available
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
		}

		encryptedAccessToken, err := s.TokenService.EncryptToken(oauthToken.AccessToken)
		if err != nil {
			slog.Error("failed to encrypt access token", "error", err)
			return nil, err
		}
		encryptedRefreshToken, err := s.TokenService.EncryptToken(oauthToken.RefreshToken)
		if err != nil {
			slog.Error("failed to encrypt refresh token", "error", err)
			return nil, err
		}
		account = &domain.Account{
			UserID:               user.ID,
			AccountID:            userInfo.ID,
			ProviderID:           domain.ProviderType(providerName),
			AccessToken:          &encryptedAccessToken,
			RefreshToken:         &encryptedRefreshToken,
			AccessTokenExpiresAt: &oauthToken.Expiry,
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

func (s *Service) getOAuth2UserInfo(ctx context.Context, providerName string, token *oauth2.Token) (*domain.OAuth2UserInfo, error) {
	provider, err := s.OAuth2ProviderRegistry.Get(providerName)
	if err != nil {
		return nil, err
	}
	return provider.GetUserInfo(ctx, token)
}
