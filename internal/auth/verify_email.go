package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// VerifyEmailToken handles all email verification types: verification, password reset confirmation, and email change confirmation
func (s *Service) VerifyEmailToken(rawToken string) (*VerifyEmailResult, error) {
	if rawToken == "" {
		return nil, ErrInvalidToken
	}

	ver, err := s.VerificationService.GetVerificationByToken(s.TokenService.HashToken(rawToken))
	if err != nil {
		slog.Error("failed to get verification token", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrVerificationNotFound, err)
	}
	if ver == nil {
		return nil, ErrVerificationNotFound
	}

	if s.VerificationService.IsExpired(ver) {
		return nil, ErrVerificationExpired
	}

	switch ver.Type {
	case models.TypeEmailVerification:
		return s.handleEmailVerification(ver)
	case models.TypePasswordReset:
		return s.handlePasswordResetConfirmation(ver)
	case models.TypeEmailChange:
		return s.handleEmailChange(ver)
	default:
		return nil, fmt.Errorf("unknown verification type: %s", ver.Type)
	}
}

// handleEmailVerification verifies a user's email address
func (s *Service) handleEmailVerification(ver *models.Verification) (*VerifyEmailResult, error) {
	if ver.UserID == nil {
		return nil, ErrUserNotFound
	}

	user, err := s.UserService.GetUserByID(*ver.UserID)
	if err != nil {
		slog.Error("failed to get user", "user_id", *ver.UserID, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrUserNotFound, err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.EmailVerified = true
	if err := s.UserService.UpdateUser(user); err != nil {
		slog.Error("failed to update user", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if err := s.VerificationService.DeleteVerification(ver.ID); err != nil {
		slog.Warn("failed to delete verification", "verification_id", ver.ID, "error", err)
	}

	s.callHook(s.config.EventHooks.OnEmailVerified, user)
	s.emitEvent(models.EventEmailVerified, user)

	return &VerifyEmailResult{
		Message: "Email verified successfully",
		User:    user,
	}, nil
}

// handlePasswordResetConfirmation confirms a password reset token
func (s *Service) handlePasswordResetConfirmation(ver *models.Verification) (*VerifyEmailResult, error) {
	// Just confirm that the token is valid
	// The actual password reset happens in ResetPassword
	return &VerifyEmailResult{
		Message: "Password reset token verified successfully",
	}, nil
}

// handleEmailChange confirms an email change
func (s *Service) handleEmailChange(ver *models.Verification) (*VerifyEmailResult, error) {
	if ver.UserID == nil {
		return nil, ErrUserNotFound
	}

	user, err := s.UserService.GetUserByID(*ver.UserID)
	if err != nil {
		slog.Error("failed to get user", "user_id", *ver.UserID, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrUserNotFound, err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.Email = ver.Identifier
	if err := s.UserService.UpdateUser(user); err != nil {
		slog.Error("failed to update user email", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to update user email: %w", err)
	}

	if err := s.VerificationService.DeleteVerification(ver.ID); err != nil {
		slog.Warn("failed to delete verification", "verification_id", ver.ID, "error", err)
	}

	s.callHook(s.config.EventHooks.OnEmailChanged, user)
	s.emitEvent(models.EventEmailChanged, user)

	return &VerifyEmailResult{
		Message: "Email changed successfully",
		User:    user,
	}, nil
}

// SendVerificationEmail generates a verification token and sends a verification email to the user
func (s *Service) SendVerificationEmail(userID string, callbackURL *string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	u, err := s.UserService.GetUserByID(userID)
	if err != nil {
		slog.Error("failed to get user by ID", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return ErrUserNotFound
	}

	token, err := s.TokenService.GenerateToken()
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
	}

	ver := &models.Verification{
		UserID:     &u.ID,
		Identifier: u.Email,
		Token:      s.TokenService.HashToken(token),
		Type:       models.TypeEmailVerification,
		ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
	}
	if err := s.VerificationService.CreateVerification(ver); err != nil {
		slog.Error("failed to create verification record", "user_id", u.ID, "error", err)
		return fmt.Errorf("failed to create verification: %w", err)
	}

	if s.config.EmailVerification.SendVerificationEmail != nil {
		url := util.BuildVerificationURL(
			s.config.BaseURL,
			s.config.BasePath,
			token,
			callbackURL,
		)
		go func() {
			if err := s.config.EmailVerification.SendVerificationEmail(*u, url, token); err != nil {
				slog.Error("failed to send verification email", "user_id", u.ID, "error", err)
			}
		}()
	}

	return nil
}
