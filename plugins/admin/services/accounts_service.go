package services

import (
	"context"
	"fmt"
	"strings"

	coreerrors "github.com/Authula/authula/core/errors"
	corerepositories "github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/types"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
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

func (s *AccountsService) GetByID(ctx context.Context, actor *models.Actor, accountID string) (*models.Account, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	return s.accountRepo.GetByID(ctx, accountID)
}

func (s *AccountsService) GetByUserID(ctx context.Context, actor *models.Actor, userID string) ([]models.Account, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, adminconstants.ErrUserIDRequired
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, coreerrors.ErrNotFound
	}

	return s.accountRepo.GetAllByUserID(ctx, userID)
}

func (s *AccountsService) Create(ctx context.Context, actor *models.Actor, userID string, request types.CreateAccountRequest) (*models.Account, error) {
	userID = strings.TrimSpace(userID)
	request.ProviderID = strings.TrimSpace(strings.ToLower(request.ProviderID))
	request.AccountID = strings.TrimSpace(request.AccountID)
	if request.Scope != nil {
		trimmed := strings.TrimSpace(*request.Scope)
		request.Scope = &trimmed
	}

	if userID == "" {
		return nil, adminconstants.ErrUserIDRequired
	}
	if request.ProviderID == "" || request.AccountID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, coreerrors.ErrNotFound
	}

	existing, err := s.accountRepo.GetByProviderAndAccountID(ctx, request.ProviderID, request.AccountID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, coreerrors.ErrConflict
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

func (s *AccountsService) Update(ctx context.Context, actor *models.Actor, accountID string, request types.UpdateAccountRequest) (*models.Account, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, coreerrors.ErrBadRequest
	}

	if request.ProviderID == nil &&
		request.AccountID == nil &&
		request.AccessToken == nil &&
		request.RefreshToken == nil &&
		request.IDToken == nil &&
		request.AccessTokenExpiresAt == nil &&
		request.RefreshTokenExpiresAt == nil &&
		request.Scope == nil &&
		request.Password == nil {
		return nil, coreerrors.ErrBadRequest
	}

	if request.ProviderID != nil {
		trimmed := strings.TrimSpace(strings.ToLower(*request.ProviderID))
		request.ProviderID = &trimmed
	}
	if request.AccountID != nil {
		trimmed := strings.TrimSpace(*request.AccountID)
		request.AccountID = &trimmed
	}
	if request.Scope != nil {
		trimmed := strings.TrimSpace(*request.Scope)
		request.Scope = &trimmed
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, coreerrors.ErrNotFound
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

func (s *AccountsService) Delete(ctx context.Context, actor *models.Actor, accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return coreerrors.ErrBadRequest
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return coreerrors.ErrNotFound
	}

	return s.accountRepo.Delete(ctx, accountID)
}
