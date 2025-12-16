package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// EmailChange initiates an email change by sending a verification email to the new address
func (s *Service) EmailChange(userID string, newEmail string, callbackURL *string) error {
	user, err := s.UserService.GetUserByID(userID)
	if err != nil {
		slog.Error("failed to get user", "user_id", userID, "error", err)
		return fmt.Errorf("%w: %w", ErrUserNotFound, err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	existingUser, err := s.UserService.GetUserByEmail(newEmail)
	if err != nil {
		slog.Error("failed to check email", "email", newEmail, "error", err)
		return fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil && existingUser.ID != userID {
		return ErrEmailAlreadyExists
	}

	token, err := s.TokenService.GenerateToken()
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
	}

	ver := &models.Verification{
		UserID:     &user.ID,
		Identifier: newEmail, // store new email in identifier
		Token:      s.TokenService.HashToken(token),
		Type:       models.TypeEmailChange,
		ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
	}
	if err := s.VerificationService.CreateVerification(ver); err != nil {
		slog.Error("failed to create verification record", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to create verification: %w", err)
	}

	if s.config.User.ChangeEmail.Enabled {
		url := util.BuildVerificationURL(
			s.config.BaseURL,
			s.config.BasePath,
			token,
			callbackURL,
		)
		go func() {
			if err := s.config.User.ChangeEmail.SendEmailChangeVerificationEmail(*user, newEmail, url, token); err != nil {
				slog.Error("failed to send email change verification", "user_id", user.ID, "error", err)
			}
		}()
	}

	return nil
}
