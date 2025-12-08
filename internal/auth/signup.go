package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// SignUpWithEmailAndPassword handles user registration with email and password
func (s *Service) SignUpWithEmailAndPassword(name string, email string, password string, callbackURL *string) (*SignUpResult, error) {
	existingUser, err := s.UserService.GetUserByEmail(email)
	if err != nil {
		slog.Error("failed to check existing user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	newUser := &domain.User{
		Name:          name,
		Email:         email,
		EmailVerified: !s.config.EmailPassword.RequireEmailVerification,
		Image:         nil,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.UserService.CreateUser(newUser); err != nil {
		slog.Error("failed to create user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrPasswordHashingFailed, err)
	}

	newAccount := &domain.Account{
		UserID:     newUser.ID,
		ProviderID: domain.ProviderEmail,
		Password:   &hashedPassword,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := s.AccountService.CreateAccount(newAccount); err != nil {
		slog.Error("failed to create account", "user_id", newUser.ID, "error", err)
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	var sessionToken string
	if s.config.EmailPassword.AutoSignIn {
		token, err := s.TokenService.GenerateToken()
		if err != nil {
			slog.Error("failed to generate session token", "error", err)
			return nil, fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
		}

		_, err = s.SessionService.CreateSession(newUser.ID, s.TokenService.HashToken(token))
		if err != nil {
			slog.Error("failed to create session", "user_id", newUser.ID, "error", err)
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		sessionToken = token
	}

	if s.config.EmailPassword.RequireEmailVerification && s.config.EmailVerification.SendOnSignUp {
		token, err := s.TokenService.GenerateToken()
		if err != nil {
			slog.Error("failed to generate verification token", "error", err)
			return nil, fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
		}

		ver := &domain.Verification{
			UserID:     &newUser.ID,
			Identifier: newUser.Email,
			Token:      s.TokenService.HashToken(token),
			Type:       domain.TypeEmailVerification,
			ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
		}
		if err := s.VerificationService.CreateVerification(ver); err != nil {
			slog.Error("failed to create verification", "user_id", newUser.ID, "error", err)
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
					slog.Error("failed to send verification email", "user_id", newUser.ID, "error", err)
				}
			}()
		}
	}

	s.callHook(s.config.EventHooks.OnUserSignedUp, newUser)

	return &SignUpResult{
		Token: sessionToken,
		User:  newUser,
	}, nil
}

// hashPassword hashes a password using configured or default method
func (s *Service) hashPassword(password string) (string, error) {
	if s.config.EmailPassword.Password != nil && s.config.EmailPassword.Password.Hash != nil {
		return s.config.EmailPassword.Password.Hash(password)
	}
	return util.HashPassword(password)
}
