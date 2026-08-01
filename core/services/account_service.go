package services

import (
	"context"
	"time"

	"github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type accountService struct {
	config       *models.Config
	accountRepo  repositories.AccountRepository
	tokenRepo    repositories.TokenRepository
	serviceHooks *models.CoreServiceHooksConfig
}

func NewAccountService(
	config *models.Config,
	accountRepo repositories.AccountRepository,
	tokenRepo repositories.TokenRepository,
	serviceHooks *models.CoreServiceHooksConfig,
) services.AccountService {
	return &accountService{config: config, accountRepo: accountRepo, tokenRepo: tokenRepo, serviceHooks: serviceHooks}
}

func runAccountHooks(hooks []models.AccountHook, account *models.Account) error {
	for _, hook := range hooks {
		if err := hook(account); err != nil {
			return err
		}
	}
	return nil
}
func (s *accountService) Create(ctx context.Context, userID string, accountID string, providerID string, password *string) (*models.Account, error) {
	account := &models.Account{
		ID:         util.GenerateUUID(),
		UserID:     userID,
		AccountID:  accountID,
		ProviderID: providerID,
		Password:   password,
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runAccountHooks(s.serviceHooks.Accounts.BeforeCreateHooks(), account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runAccountHooks(s.serviceHooks.Accounts.AfterCreateHooks(), created); err != nil {
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

		if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
			if err := runAccountHooks(s.serviceHooks.Accounts.BeforeUpdateHooks(), existing); err != nil {
				return nil, err
			}
		}

		updated, err := s.accountRepo.Update(ctx, existing)
		if err != nil {
			return nil, err
		}

		if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
			if err := runAccountHooks(s.serviceHooks.Accounts.AfterUpdateHooks(), updated); err != nil {
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

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runAccountHooks(s.serviceHooks.Accounts.BeforeCreateHooks(), account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runAccountHooks(s.serviceHooks.Accounts.AfterCreateHooks(), created); err != nil {
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
	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runAccountHooks(s.serviceHooks.Accounts.BeforeUpdateHooks(), account); err != nil {
			return nil, err
		}
	}

	updatedAccount, err := s.accountRepo.Update(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runAccountHooks(s.serviceHooks.Accounts.AfterUpdateHooks(), updatedAccount); err != nil {
			return nil, err
		}
	}

	return updatedAccount, nil
}

func (s *accountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	return s.accountRepo.UpdateFields(ctx, userID, fields)
}
