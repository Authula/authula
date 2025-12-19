package resetpassword

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

func (s *service) ResetPassword(ctx context.Context, email string, callbackURL *string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}

	user, err := s.userService.GetUserByEmail(email)
	if err != nil {
		s.logger.Error("failed to get user by email", "email", email, "error", err)
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		s.logger.Info("password reset requested for non-existent email", "email", email)
		return nil
	}

	token, err := s.tokenService.GenerateToken()
	if err != nil {
		s.logger.Error("failed to generate token", "error", err)
		return fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
	}

	resetTokenExpiry := s.config.EmailPassword.ResetTokenExpiry

	ver := &models.Verification{
		UserID:     &user.ID,
		Identifier: user.Email,
		Token:      s.tokenService.HashToken(token),
		Type:       models.TypePasswordReset,
		ExpiresAt:  time.Now().UTC().Add(resetTokenExpiry),
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

	if s.config.EmailPassword.SendResetPasswordEmail != nil {
		if err := s.config.EmailPassword.SendResetPasswordEmail(*user, url, token); err != nil {
			s.logger.Error("failed to send verification email", "user_id", user.ID, "error", err)
		}
	} else {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.mailerService.Send(
				ctx,
				user.Email,
				"Reset Your Password",
				"Reset your password",
				util.CreateResetPasswordEmailBody(*user, url),
			)
		}()
	}

	return nil
}
