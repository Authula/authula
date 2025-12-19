package sendverificationemail

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
	tokenService        models.TokenService
	verificationService models.VerificationService
	mailerService       models.MailerService
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	tokenService models.TokenService,
	verificationService models.VerificationService,
	mailerService models.MailerService,
) *service {
	return &service{
		config:              config,
		logger:              logger,
		userService:         userService,
		tokenService:        tokenService,
		verificationService: verificationService,
		mailerService:       mailerService,
	}
}

func (s *service) SendEmailVerification(ctx context.Context, userID string, callbackURL *string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		s.logger.Error("failed to get user by ID", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return constants.ErrUserNotFound
	}

	token, err := s.tokenService.GenerateToken()
	if err != nil {
		s.logger.Error("failed to generate token", "error", err)
		return fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
	}

	ver := &models.Verification{
		UserID:     &user.ID,
		Identifier: user.Email,
		Token:      s.tokenService.HashToken(token),
		Type:       models.TypeEmailVerification,
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

	if s.config.EmailVerification.SendVerificationEmail != nil {
		if err := s.config.EmailVerification.SendVerificationEmail(*user, url, token); err != nil {
			s.logger.Error("failed to send verification email", "user_id", user.ID, "error", err)
		}
	} else {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.mailerService.Send(
				ctx,
				user.Email,
				"Verify Your Email",
				"Verify your email address",
				util.CreateVerificationEmailBody(*user, url),
			)
		}()
	}

	return nil
}
