package auth

import (
	"fmt"
	"log/slog"

	"github.com/GoBetterAuth/go-better-auth/models"
)

// ChangePassword completes a password reset with a verification token and new password
func (s *Service) ChangePassword(rawToken, newPassword string) error {
	if rawToken == "" {
		return ErrInvalidToken
	}
	if newPassword == "" {
		return fmt.Errorf("new password is required")
	}

	ver, err := s.VerificationService.GetVerificationByToken(s.TokenService.HashToken(rawToken))
	if err != nil {
		slog.Error("failed to get verification token", "error", err)
		return fmt.Errorf("%w: %w", ErrVerificationNotFound, err)
	}
	if ver == nil || ver.Type != models.TypePasswordReset {
		return ErrVerificationInvalid
	}

	if s.VerificationService.IsExpired(ver) {
		return ErrVerificationExpired
	}

	if ver.UserID == nil {
		return ErrUserNotFound
	}

	user, err := s.UserService.GetUserByID(*ver.UserID)
	if err != nil {
		slog.Error("failed to get user", "user_id", *ver.UserID, "error", err)
		return fmt.Errorf("%w: %w", ErrUserNotFound, err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	acc, err := s.AccountService.GetAccountByUserID(user.ID)
	if err != nil {
		slog.Error("failed to get account", "user_id", user.ID, "error", err)
		return fmt.Errorf("%w: %w", ErrAccountNotFound, err)
	}
	if acc == nil {
		return ErrAccountNotFound
	}

	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return fmt.Errorf("%w: %w", ErrPasswordHashingFailed, err)
	}

	acc.Password = &hashedPassword
	if err := s.AccountService.UpdateAccount(acc); err != nil {
		slog.Error("failed to update account", "account_id", acc.ID, "error", err)
		return fmt.Errorf("failed to update account: %w", err)
	}

	if err := s.VerificationService.DeleteVerification(ver.ID); err != nil {
		slog.Warn("failed to delete verification", "verification_id", ver.ID, "error", err)
	}

	s.callHook(s.config.EventHooks.OnPasswordChanged, user)
	s.emitEvent(models.EventPasswordChanged, user)

	return nil
}
