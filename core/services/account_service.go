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
	logger       models.Logger
}

func NewAccountService(
	config *models.Config,
	accountRepo repositories.AccountRepository,
	tokenRepo repositories.TokenRepository,
	serviceHooks *models.CoreServiceHooksConfig,
	logger models.Logger,
) services.AccountService {
	return &accountService{config: config, accountRepo: accountRepo, tokenRepo: tokenRepo, serviceHooks: serviceHooks, logger: logger}
}
func (s *accountService) Create(ctx context.Context, userID string, accountID string, providerID string, password *string) (*models.Account, error) {
	accountID = normalizeEmailProviderAccountID(providerID, accountID)

	account := &models.Account{
		ID:         util.GenerateUUID(),
		UserID:     userID,
		AccountID:  accountID,
		ProviderID: providerID,
		Password:   password,
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runHooks(s.serviceHooks.Accounts.BeforeCreateHooks(), account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		runAfterHooks(s.logger, "after create account", s.serviceHooks.Accounts.AfterCreateHooks(), created)
	}

	return created, nil
}

func (s *accountService) CreateOAuth2(ctx context.Context, userID string, providerAccountID string, provider string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	if err := validateOAuthTokenExpiry(accessTokenExpiresAt); err != nil {
		return nil, err
	}

	encryptedAccessToken, err := s.tokenRepo.Encrypt(accessToken)
	if err != nil {
		return nil, err
	}

	encryptedRefreshToken, err := s.encryptOptionalToken(refreshToken)
	if err != nil {
		return nil, err
	}

	existing, err := s.accountRepo.GetByProviderAndAccountID(ctx, provider, providerAccountID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.UserID != userID {
			return nil, ErrAccountLinkedToAnotherUser
		}

		existing.AccessToken = &encryptedAccessToken
		if encryptedRefreshToken != nil {
			existing.RefreshToken = encryptedRefreshToken
		}
		existing.AccessTokenExpiresAt = accessTokenExpiresAt
		if refreshTokenExpiresAt != nil {
			existing.RefreshTokenExpiresAt = refreshTokenExpiresAt
		}
		if scope != nil {
			existing.Scope = scope
		}

		return s.updateAccount(ctx, existing)
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
		if err := runHooks(s.serviceHooks.Accounts.BeforeCreateHooks(), account); err != nil {
			return nil, err
		}
	}

	created, err := s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		runAfterHooks(s.logger, "after create account", s.serviceHooks.Accounts.AfterCreateHooks(), created)
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

// UpdateOAuth2Tokens updates the OAuth2 token fields of a single account identified by
// its account ID. Tokens are encrypted at write time. When the provider omits a refresh
// token, refresh-token expiry, or scope, the stored value is kept.
func (s *accountService) UpdateOAuth2Tokens(ctx context.Context, accountID string, accessToken string, refreshToken *string, accessTokenExpiresAt *time.Time, refreshTokenExpiresAt *time.Time, scope *string) (*models.Account, error) {
	if accessToken == "" {
		return nil, ErrTokenExpired
	}
	if err := validateOAuthTokenExpiry(accessTokenExpiresAt); err != nil {
		return nil, err
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}

	encryptedAccessToken, err := s.tokenRepo.Encrypt(accessToken)
	if err != nil {
		return nil, err
	}

	encryptedRefreshToken, err := s.encryptOptionalToken(refreshToken)
	if err != nil {
		return nil, err
	}

	account.AccessToken = &encryptedAccessToken
	if encryptedRefreshToken != nil {
		account.RefreshToken = encryptedRefreshToken
	}
	account.AccessTokenExpiresAt = accessTokenExpiresAt
	if refreshTokenExpiresAt != nil {
		account.RefreshTokenExpiresAt = refreshTokenExpiresAt
	}
	if scope != nil {
		account.Scope = scope
	}

	return s.updateAccount(ctx, account)
}

func (s *accountService) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	return s.updateAccount(ctx, account)
}

func (s *accountService) updateAccount(ctx context.Context, account *models.Account) (*models.Account, error) {
	if account != nil {
		account.AccountID = normalizeEmailProviderAccountID(account.ProviderID, account.AccountID)
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		if err := runHooks(s.serviceHooks.Accounts.BeforeUpdateHooks(), account); err != nil {
			return nil, err
		}
	}

	updatedAccount, err := s.accountRepo.Update(ctx, account)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Accounts != nil {
		runAfterHooks(s.logger, "after update account", s.serviceHooks.Accounts.AfterUpdateHooks(), updatedAccount)
	}

	return updatedAccount, nil
}

func (s *accountService) encryptOptionalToken(token *string) (*string, error) {
	if token == nil || *token == "" {
		return nil, nil
	}

	encrypted, err := s.tokenRepo.Encrypt(*token)
	if err != nil {
		return nil, err
	}
	return &encrypted, nil
}

func validateOAuthTokenExpiry(expiresAt *time.Time) error {
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return ErrTokenExpired
	}
	return nil
}

func (s *accountService) UpdateFields(ctx context.Context, userID string, fields map[string]any) error {
	return s.accountRepo.UpdateFields(ctx, userID, fields)
}

// normalizeEmailProviderAccountID keeps email/password accounts keyed by the
// normalized email, matching users.email.
func normalizeEmailProviderAccountID(providerID, accountID string) string {
	if providerID == models.AuthProviderEmail.String() {
		return util.NormalizeEmail(accountID)
	}
	return accountID
}
