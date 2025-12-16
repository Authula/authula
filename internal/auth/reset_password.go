package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// ResetPassword initiates a password reset by sending a verification email
func (s *Service) ResetPassword(email string, callbackURL *string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	user, err := s.UserService.GetUserByEmail(email)
	if err != nil {
		slog.Error("failed to get user by email", "email", email, "error", err)
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		slog.Info("password reset requested for non-existent email", "email", email)
		return nil
	}

	token, err := s.TokenService.GenerateToken()
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
	}

	resetTokenExpiry := s.config.EmailPassword.ResetTokenExpiry
	if resetTokenExpiry == 0 {
		resetTokenExpiry = time.Hour
	}

	ver := &models.Verification{
		UserID:     &user.ID,
		Identifier: user.Email,
		Token:      s.TokenService.HashToken(token),
		Type:       models.TypePasswordReset,
		ExpiresAt:  time.Now().UTC().Add(resetTokenExpiry),
	}
	if err := s.VerificationService.CreateVerification(ver); err != nil {
		slog.Error("failed to create verification record", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to create verification: %w", err)
	}

	if s.config.EmailPassword.SendResetPasswordEmail != nil {
		callbackURL := fmt.Sprintf("%s?token=%s", *callbackURL, token)
		url := util.BuildVerificationURL(
			s.config.BaseURL,
			s.config.BasePath,
			token,
			&callbackURL,
		)
		go func() {
			if err := s.config.EmailPassword.SendResetPasswordEmail(*user, url, token); err != nil {
				slog.Error("failed to send verification email", "user_id", user.ID, "error", err)
			}
		}()
	}

	return nil
}
