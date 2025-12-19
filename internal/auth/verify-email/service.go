package verifyemail

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
	tokenService        models.TokenService
	verificationService models.VerificationService
	eventEmitter        models.EventEmitter
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	tokenService models.TokenService,
	verificationService models.VerificationService,
	eventEmitter models.EventEmitter,
) *service {
	return &service{
		config:              config,
		logger:              logger,
		userService:         userService,
		tokenService:        tokenService,
		verificationService: verificationService,
		eventEmitter:        eventEmitter,
	}
}

func (s *service) VerifyEmail(ctx context.Context, rawToken string) (*models.VerifyEmailResult, error) {
	if rawToken == "" {
		return nil, constants.ErrInvalidToken
	}

	ver, err := s.verificationService.GetVerificationByToken(s.tokenService.HashToken(rawToken))
	if err != nil {
		s.logger.Error("failed to get verification token", "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrVerificationNotFound, err)
	}
	if ver == nil {
		return nil, constants.ErrVerificationNotFound
	}

	if s.verificationService.IsExpired(ver) {
		return nil, constants.ErrVerificationExpired
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
func (s *service) handleEmailVerification(ver *models.Verification) (*models.VerifyEmailResult, error) {
	if ver.UserID == nil {
		return nil, constants.ErrUserNotFound
	}

	user, err := s.userService.GetUserByID(*ver.UserID)
	if err != nil {
		s.logger.Error("failed to get user", "user_id", *ver.UserID, "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrUserNotFound, err)
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
	}

	user.EmailVerified = true
	if err := s.userService.UpdateUser(user); err != nil {
		s.logger.Error("failed to update user", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if err := s.verificationService.DeleteVerification(ver.ID); err != nil {
		s.logger.Warn("failed to delete verification", "verification_id", ver.ID, "error", err)
	}

	s.eventEmitter.OnEmailVerified(*user)

	return &models.VerifyEmailResult{
		Message: "Email verified successfully",
		User:    user,
	}, nil
}

// handlePasswordResetConfirmation confirms a password reset token
func (s *service) handlePasswordResetConfirmation(ver *models.Verification) (*models.VerifyEmailResult, error) {
	// Just confirm that the token is valid
	// The actual password reset happens in ResetPassword
	return &models.VerifyEmailResult{
		Message: "Password reset token verified successfully",
	}, nil
}

// handleEmailChange confirms an email change
func (s *service) handleEmailChange(ver *models.Verification) (*models.VerifyEmailResult, error) {
	if ver.UserID == nil {
		return nil, constants.ErrUserNotFound
	}

	user, err := s.userService.GetUserByID(*ver.UserID)
	if err != nil {
		s.logger.Error("failed to get user", "user_id", *ver.UserID, "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrUserNotFound, err)
	}
	if user == nil {
		return nil, constants.ErrUserNotFound
	}

	user.Email = ver.Identifier
	if err := s.userService.UpdateUser(user); err != nil {
		s.logger.Error("failed to update user email", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to update user email: %w", err)
	}

	if err := s.verificationService.DeleteVerification(ver.ID); err != nil {
		s.logger.Warn("failed to delete verification", "verification_id", ver.ID, "error", err)
	}

	s.eventEmitter.OnEmailChanged(*user)

	return &models.VerifyEmailResult{
		Message: "Email changed successfully",
		User:    user,
	}, nil
}
