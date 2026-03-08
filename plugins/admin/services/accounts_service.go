package services

import (
	"context"
	"fmt"

	corerepositories "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	adminconstants "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/constants"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	rootservices "github.com/GoBetterAuth/go-better-auth/v2/services"
)

type AccountsService struct {
	accountRepo     corerepositories.AccountRepository
	userRepo        corerepositories.UserRepository
	passwordService rootservices.PasswordService
}

func NewAccountsService(
	accountRepo corerepositories.AccountRepository,
	userRepo corerepositories.UserRepository,
	passwordService rootservices.PasswordService,
) *AccountsService {
	return &AccountsService{accountRepo: accountRepo, userRepo: userRepo, passwordService: passwordService}
}

func (s *AccountsService) GetByID(ctx context.Context, accountID string) (*models.Account, error) {
	return s.accountRepo.GetByID(ctx, accountID)
}

func (s *AccountsService) GetByUserID(ctx context.Context, userID string) ([]models.Account, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, adminconstants.ErrNotFound
	}

	return s.accountRepo.GetAllByUserID(ctx, userID)
}

func (s *AccountsService) Create(ctx context.Context, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, adminconstants.ErrNotFound
	}

	existing, err := s.accountRepo.GetByProviderAndAccountID(ctx, request.ProviderID, request.AccountID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, adminconstants.ErrConflict
	}

	password := request.Password
	if password != nil {
		if s.passwordService == nil {
			return nil, fmt.Errorf("password service unavailable")
		}
		hashed, err := s.passwordService.Hash(*password)
		if err != nil {
			return nil, err
		}
		password = &hashed
	}

	account := &models.Account{
		ID:                    util.GenerateUUID(),
		UserID:                userID,
		AccountID:             request.AccountID,
		ProviderID:            request.ProviderID,
		AccessToken:           request.AccessToken,
		RefreshToken:          request.RefreshToken,
		IDToken:               request.IDToken,
		AccessTokenExpiresAt:  request.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: request.RefreshTokenExpiresAt,
		Scope:                 request.Scope,
		Password:              password,
	}

	return s.accountRepo.Create(ctx, account)
}

func (s *AccountsService) Update(ctx context.Context, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, adminconstants.ErrNotFound
	}

	if request.ProviderID != nil {
		account.ProviderID = *request.ProviderID
	}
	if request.AccountID != nil {
		account.AccountID = *request.AccountID
	}
	if request.AccessToken != nil {
		account.AccessToken = request.AccessToken
	}
	if request.RefreshToken != nil {
		account.RefreshToken = request.RefreshToken
	}
	if request.IDToken != nil {
		account.IDToken = request.IDToken
	}
	if request.AccessTokenExpiresAt != nil {
		account.AccessTokenExpiresAt = request.AccessTokenExpiresAt
	}
	if request.RefreshTokenExpiresAt != nil {
		account.RefreshTokenExpiresAt = request.RefreshTokenExpiresAt
	}
	if request.Scope != nil {
		account.Scope = request.Scope
	}
	if request.Password != nil {
		if s.passwordService == nil {
			return nil, fmt.Errorf("password service unavailable")
		}
		hashed, err := s.passwordService.Hash(*request.Password)
		if err != nil {
			return nil, err
		}
		account.Password = &hashed
	}

	return s.accountRepo.Update(ctx, account)
}

func (s *AccountsService) Delete(ctx context.Context, accountID string) error {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return adminconstants.ErrNotFound
	}

	return s.accountRepo.Delete(ctx, accountID)
}
