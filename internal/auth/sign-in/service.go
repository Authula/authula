package signin

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
	mailerService       models.MailerService
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
	mailerService models.MailerService,
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
		mailerService:       mailerService,
		passwordService:     passwordService,
		eventEmitter:        eventEmitter,
	}
}

func (s *service) SignInWithEmailAndPassword(ctx context.Context, email, password string, callbackURL *string) (*models.SignInResult, error) {
	user, err := s.userService.GetUserByEmail(email)
	if err != nil {
		s.logger.Error("failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrUserNotFound, err)
	}
	if user == nil {
		return nil, constants.ErrInvalidCredentials
	}

	acc, err := s.accountService.GetAccountByUserID(user.ID)
	if err != nil {
		s.logger.Error("failed to get account", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrAccountNotFound, err)
	}
	if acc == nil {
		return nil, constants.ErrInvalidCredentials
	}

	if acc.Password == nil {
		return nil, constants.ErrInvalidCredentials
	}

	isValid, err := s.verifyPassword(password, *acc.Password)
	if err != nil || !isValid {
		return nil, constants.ErrInvalidCredentials
	}

	existingSession, err := s.sessionService.GetSessionByUserID(user.ID)
	if err != nil {
		s.logger.Error("failed to get existing session", "user_id", user.ID, "error", err)
	} else if existingSession != nil {
		if err := s.sessionService.DeleteSessionByID(existingSession.ID); err != nil {
			s.logger.Warn("failed to delete existing session", "session_id", existingSession.ID, "error", err)
		}
	}

	token, err := s.tokenService.GenerateToken()
	if err != nil {
		s.logger.Error("failed to generate token", "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
	}

	_, err = s.sessionService.CreateSession(user.ID, s.tokenService.HashToken(token))
	if err != nil {
		s.logger.Error("failed to create session", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("%w: %w", constants.ErrSessionCreationFailed, err)
	}

	if s.config.EmailVerification.SendOnSignIn && !user.EmailVerified {
		token, err := s.tokenService.GenerateToken()
		if err != nil {
			s.logger.Error("failed to generate verification token", "error", err)
			// Don't fail the signin, just log
		} else {
			ver := &models.Verification{
				UserID:     &user.ID,
				Identifier: user.Email,
				Token:      s.tokenService.HashToken(token),
				Type:       models.TypeEmailVerification,
				ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
			}
			if err := s.verificationService.CreateVerification(ver); err != nil {
				s.logger.Error("failed to create verification", "user_id", user.ID, "error", err)
				// Don't fail the signin
			} else {
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
			}
		}
	}

	s.eventEmitter.OnUserLoggedIn(*user)

	var csrfToken *string = nil
	if s.config.CSRF.Enabled {
		csrfTokenGenerated, err := s.tokenService.GenerateToken()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", constants.ErrTokenGenerationFailed, err)
		}
		csrfToken = &csrfTokenGenerated
	}

	return &models.SignInResult{
		User:      user,
		Token:     token,
		CSRFToken: csrfToken,
	}, nil
}

func (s *service) verifyPassword(password string, hashedPassword string) (bool, error) {
	if s.config.EmailPassword.Password.Verify != nil {
		valid := s.config.EmailPassword.Password.Verify(password, hashedPassword)
		if valid {
			return true, nil
		}
		return false, nil
	}
	return s.passwordService.VerifyPassword(password, hashedPassword)
}
