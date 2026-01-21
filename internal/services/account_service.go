package services

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type accountService struct {
	config      *models.Config
	accountRepo repositories.AccountRepository
	tokenRepo   repositories.TokenRepository
	hooks       *models.CoreDatabaseHooks
}

func NewAccountService(
	config *models.Config,
	accountRepo repositories.AccountRepository,
	tokenRepo repositories.TokenRepository,
	hooks *models.CoreDatabaseHooks,
) services.AccountService {
	return &accountService{config: config, accountRepo: accountRepo, tokenRepo: tokenRepo, hooks: hooks}
}
func (s *accountService) Create(ctx context.Context, userID string, accountID string, providerID string, password *string) (*models.Account, error) {
	account := &models.Account{
		ID:         util.GenerateUUID(),
		UserID:     userID,
		AccountID:  accountID,
		ProviderID: providerID,
		Password:   password,
	}

	if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.BeforeCreate != nil {
		if err := s.hooks.Accounts.BeforeCreate(account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.AfterCreate != nil {
		if err := s.hooks.Accounts.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *accountService) CreateOAuth2(ctx context.Context, userID string, providerAccountID string, provider string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	encryptedAccessToken, err := s.tokenRepo.Encrypt(accessToken)
	if err != nil {
		return nil, err
	}

	var encryptedRefreshToken *string
	if refreshToken != nil {
		encrypted, err := s.tokenRepo.Encrypt(*refreshToken)
		if err != nil {
			return nil, err
		}
		encryptedRefreshToken = &encrypted
	}

	existing, err := s.accountRepo.GetByProviderAndAccountID(ctx, provider, providerAccountID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// Update existing account
		existing.AccessToken = &encryptedAccessToken
		existing.RefreshToken = encryptedRefreshToken
		existing.AccessTokenExpiresAt = accessTokenExpiresAt
		existing.RefreshTokenExpiresAt = refreshTokenExpiresAt
		existing.Scope = scope

		if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.BeforeUpdate != nil {
			if err := s.hooks.Accounts.BeforeUpdate(existing); err != nil {
				return nil, err
			}
		}

		updated, err := s.accountRepo.Update(ctx, existing)
		if err != nil {
			return nil, err
		}

		if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.AfterUpdate != nil {
			if err := s.hooks.Accounts.AfterUpdate(*updated); err != nil {
				return nil, err
			}
		}

		return updated, nil
	}

	account := &models.Account{
		ID:                    util.GenerateUUID(),
		UserID:                userID,
		AccountID:             providerAccountID,
		ProviderID:            provider,
		AccessToken:           &encryptedAccessToken,
		RefreshToken:          encryptedRefreshToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
		Password:              nil,
		Scope:                 scope,
	}

	if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.BeforeCreate != nil {
		if err := s.hooks.Accounts.BeforeCreate(account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.AfterCreate != nil {
		if err := s.hooks.Accounts.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *accountService) GetByUserID(ctx context.Context, userID string) (*models.Account, error) {
	return s.accountRepo.GetByUserID(ctx, userID)
}

func (s *accountService) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.Account, error) {
	return s.accountRepo.GetByUserIDAndProvider(ctx, userID, provider)
}

func (s *accountService) GetByProviderAndAccountID(ctx context.Context, provider, accountID string) (*models.Account, error) {
	return s.accountRepo.GetByProviderAndAccountID(ctx, provider, accountID)
}

func (s *accountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.BeforeUpdate != nil {
		if err := s.hooks.Accounts.BeforeUpdate(account); err != nil {
			return nil, err
		}
	}

	updatedAccount, err := s.accountRepo.Update(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Accounts != nil && s.hooks.Accounts.AfterUpdate != nil {
		if err := s.hooks.Accounts.AfterUpdate(*updatedAccount); err != nil {
			return nil, err
		}
	}

	return updatedAccount, nil
}

func (s *accountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	return s.accountRepo.UpdateFields(ctx, userID, fields)
}
