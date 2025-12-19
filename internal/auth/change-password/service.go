package changepassword

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/internal/constants"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type service struct {
	config              *models.Config
	logger              models.Logger
	userService         models.UserService
	accountService      models.AccountService
	verificationService models.VerificationService
	tokenService        models.TokenService
	passwordService     models.PasswordService
	eventEmitter        models.EventEmitter
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	accountService models.AccountService,
	verificationService models.VerificationService,
	tokenService models.TokenService,
	passwordService models.PasswordService,
	eventEmitter models.EventEmitter,
) *service {
	return &service{
		config:              config,
		logger:              logger,
		userService:         userService,
		accountService:      accountService,
		verificationService: verificationService,
		tokenService:        tokenService,
		passwordService:     passwordService,
		eventEmitter:        eventEmitter,
	}
}

func (s *service) ChangePassword(ctx context.Context, rawToken string, newPassword string) error {
	if rawToken == "" {
		return constants.ErrMissingToken
	}
	if newPassword == "" {
		return fmt.Errorf("new password is required")
	}

	ver, err := s.verificationService.GetVerificationByToken(s.tokenService.HashToken(rawToken))
	if err != nil {
		s.logger.Error("failed to get verification token", "error", err)
		return fmt.Errorf("%w: %w", constants.ErrVerificationNotFound, err)
	}
	if ver == nil || ver.Type != models.TypePasswordReset {
		return constants.ErrVerificationInvalid
	}

	if s.verificationService.IsExpired(ver) {
		return constants.ErrVerificationExpired
	}

	if ver.UserID == nil {
		return constants.ErrUserNotFound
	}

	user, err := s.userService.GetUserByID(*ver.UserID)
	if err != nil {
		s.logger.Error("failed to get user", "user_id", *ver.UserID, "error", err)
		return fmt.Errorf("%w: %w", constants.ErrUserNotFound, err)
	}
	if user == nil {
		return constants.ErrUserNotFound
	}

	acc, err := s.accountService.GetAccountByUserID(user.ID)
	if err != nil {
		s.logger.Error("failed to get account", "user_id", user.ID, "error", err)
		return fmt.Errorf("%w: %w", constants.ErrAccountNotFound, err)
	}
	if acc == nil {
		return constants.ErrAccountNotFound
	}

	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		return fmt.Errorf("%w: %w", constants.ErrPasswordHashingFailed, err)
	}

	acc.Password = &hashedPassword
	if err := s.accountService.UpdateAccount(acc); err != nil {
		s.logger.Error("failed to update account", "account_id", acc.ID, "error", err)
		return fmt.Errorf("failed to update account: %w", err)
	}

	if err := s.verificationService.DeleteVerification(ver.ID); err != nil {
		s.logger.Warn("failed to delete verification", "verification_id", ver.ID, "error", err)
	}

	s.eventEmitter.OnPasswordChanged(*user)

	return nil
}

func (s *service) hashPassword(password string) (string, error) {
	if s.config.EmailPassword.Password.Hash != nil {
		return s.config.EmailPassword.Password.Hash(password)
	}
	return s.passwordService.HashPassword(password)
}
