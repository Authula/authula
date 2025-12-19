package emailchange

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/constants"
	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type service struct {
	config              *models.Config
	logger              models.Logger
	userService         models.UserService
	verificationService models.VerificationService
	tokenService        models.TokenService
	mailerService       models.MailerService
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	verificationService models.VerificationService,
	tokenService models.TokenService,
	mailerService models.MailerService,
) *service {
	return &service{
		config:              config,
		logger:              logger,
		userService:         userService,
		verificationService: verificationService,
		tokenService:        tokenService,
		mailerService:       mailerService,
	}
}

func (s *service) EmailChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error {
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		s.logger.Error("failed to get user", "user_id", userID, "error", err)
		return fmt.Errorf("%w: %w", constants.ErrUserNotFound, err)
	}
	if user == nil {
		return constants.ErrUserNotFound
	}

	existingUser, err := s.userService.GetUserByEmail(newEmail)
	if err != nil {
		s.logger.Error("failed to check email", "email", newEmail, "error", err)
		return fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil && existingUser.ID != userID {
		return constants.ErrEmailAlreadyExists
	}

	token, err := s.tokenService.GenerateToken()
	if err != nil {
		s.logger.Error("failed to generate token", "error", err)
		return fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
	}

	ver := &models.Verification{
		UserID:     &user.ID,
		Identifier: newEmail, // store new email in identifier
		Token:      s.tokenService.HashToken(token),
		Type:       models.TypeEmailChange,
		ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
	}
	if err := s.verificationService.CreateVerification(ver); err != nil {
		s.logger.Error("failed to create verification record", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to create verification: %w", err)
	}

	url := util.BuildVerificationURL(
		s.config.BaseURL,
		s.config.BasePath,
		token,
		callbackURL,
	)

	if s.config.User.ChangeEmail.SendEmailChangeVerificationEmail != nil {
		if err := s.config.User.ChangeEmail.SendEmailChangeVerificationEmail(*user, newEmail, url, token); err != nil {
			s.logger.Error("failed to send email change verification", "user_id", user.ID, "error", err)
		}
	} else {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.mailerService.Send(
				ctx,
				newEmail,
				"Verify Your New Email Address",
				"Verify your new email address",
				util.CreateEmailChangeVerificationBody(*user, newEmail, url),
			)
		}()
	}

	return nil
}
