package signup

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
	accountService      models.AccountService
	sessionService      models.SessionService
	tokenService        models.TokenService
	verificationService models.VerificationService
	passwordService     models.PasswordService
	eventEmitter        models.EventEmitter
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	accountService models.AccountService,
	sessionService models.SessionService,
	tokenService models.TokenService,
	verificationService models.VerificationService,
	passwordService models.PasswordService,
	eventEmitter models.EventEmitter,
) *service {
	return &service{
		config:              config,
		logger:              logger,
		userService:         userService,
		accountService:      accountService,
		sessionService:      sessionService,
		tokenService:        tokenService,
		verificationService: verificationService,
		passwordService:     passwordService,
		eventEmitter:        eventEmitter,
	}
}

func (s *service) SignUpWithEmailAndPassword(ctx context.Context, name string, email string, password string, callbackURL *string) (*models.SignUpResult, error) {
	existingUser, err := s.userService.GetUserByEmail(email)
	if err != nil {
		s.logger.Error("failed to check existing user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, constants.ErrUserAlreadyExists
	}

	newUser := &models.User{
		Name:          name,
		Email:         email,
		EmailVerified: !s.config.EmailPassword.RequireEmailVerification,
		Image:         nil,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.userService.CreateUser(newUser); err != nil {
		s.logger.Error("failed to create user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrPasswordHashingFailed, err)
	}

	newAccount := &models.Account{
		UserID:     newUser.ID,
		ProviderID: models.ProviderEmail,
		Password:   &hashedPassword,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := s.accountService.CreateAccount(newAccount); err != nil {
		s.logger.Error("failed to create account", "user_id", newUser.ID, "error", err)
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	var sessionToken string
	if s.config.EmailPassword.AutoSignIn {
		token, err := s.tokenService.GenerateToken()
		if err != nil {
			s.logger.Error("failed to generate session token", "error", err)
			return nil, fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
		}

		_, err = s.sessionService.CreateSession(newUser.ID, s.tokenService.HashToken(token))
		if err != nil {
			s.logger.Error("failed to create session", "user_id", newUser.ID, "error", err)
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		sessionToken = token
	}

	if s.config.EmailPassword.RequireEmailVerification && s.config.EmailVerification.SendOnSignUp {
		token, err := s.tokenService.GenerateToken()
		if err != nil {
			s.logger.Error("failed to generate verification token", "error", err)
			return nil, fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
		}

		ver := &models.Verification{
			UserID:     &newUser.ID,
			Identifier: newUser.Email,
			Token:      s.tokenService.HashToken(token),
			Type:       models.TypeEmailVerification,
			ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
		}
		if err := s.verificationService.CreateVerification(ver); err != nil {
			s.logger.Error("failed to create verification", "user_id", newUser.ID, "error", err)
			return nil, fmt.Errorf("failed to create verification: %w", err)
		}

		if s.config.EmailVerification.SendVerificationEmail != nil {
			url := util.BuildVerificationURL(
				s.config.BaseURL,
				s.config.BasePath,
				token,
				callbackURL,
			)
			go func() {
				if err := s.config.EmailVerification.SendVerificationEmail(*newUser, url, token); err != nil {
					s.logger.Error("failed to send verification email", "user_id", newUser.ID, "error", err)
				}
			}()
		}
	}

	s.eventEmitter.OnUserSignedUp(*newUser)

	var csrfToken *string = nil
	if s.config.CSRF.Enabled {
		csrfTokenGenerated, err := s.tokenService.GenerateToken()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
		}
		csrfToken = &csrfTokenGenerated
	}

	return &models.SignUpResult{
		Token:     sessionToken,
		User:      newUser,
		CSRFToken: csrfToken,
	}, nil
}

func (s *service) hashPassword(password string) (string, error) {
	if s.config.EmailPassword.Password.Hash != nil {
		return s.config.EmailPassword.Password.Hash(password)
	}
	return s.passwordService.HashPassword(password)
}
