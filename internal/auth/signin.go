package auth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/pkg/domain"
)

// SignInWithEmailAndPassword handles email/password authentication
func (s *Service) SignInWithEmailAndPassword(email string, password string, callbackURL *string) (*SignInResult, error) {
	user, err := s.UserService.GetUserByEmail(email)
	if err != nil {
		slog.Error("failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrUserNotFound, err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	acc, err := s.AccountService.GetAccountByUserID(user.ID)
	if err != nil {
		slog.Error("failed to get account", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrAccountNotFound, err)
	}
	if acc == nil {
		return nil, ErrInvalidCredentials
	}

	if acc.Password == nil {
		return nil, ErrInvalidCredentials
	}

	isValid, err := util.VerifyPassword(password, *acc.Password)
	if err != nil || !isValid {
		return nil, ErrInvalidCredentials
	}

	existingSession, err := s.SessionService.GetSessionByUserID(user.ID)
	if err != nil {
		slog.Error("failed to get existing session", "user_id", user.ID, "error", err)
	} else if existingSession != nil {
		if err := s.SessionService.DeleteSessionByID(existingSession.ID); err != nil {
			slog.Warn("failed to delete existing session", "session_id", existingSession.ID, "error", err)
		}
	}

	token, err := s.TokenService.GenerateToken()
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return nil, fmt.Errorf("%w: %w", ErrTokenGenerationFailed, err)
	}

	_, err = s.SessionService.CreateSession(user.ID, s.TokenService.HashToken(token))
	if err != nil {
		slog.Error("failed to create session", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("%w: %w", ErrSessionCreationFailed, err)
	}

	s.callHook(s.config.EventHooks.OnUserLoggedIn, user)

	if s.config.EmailVerification.SendOnSignIn && !user.EmailVerified {
		token, err := s.TokenService.GenerateToken()
		if err != nil {
			slog.Error("failed to generate verification token", "error", err)
			// Don't fail the signin, just log
		} else {
			ver := &domain.Verification{
				UserID:     &user.ID,
				Identifier: user.Email,
				Token:      s.TokenService.HashToken(token),
				Type:       domain.TypeEmailVerification,
				ExpiresAt:  time.Now().UTC().Add(s.config.EmailVerification.ExpiresIn),
			}
			if err := s.VerificationService.CreateVerification(ver); err != nil {
				slog.Error("failed to create verification", "user_id", user.ID, "error", err)
				// Don't fail the signin
			} else if s.config.EmailVerification.SendVerificationEmail != nil {
				url := util.BuildVerificationURL(
					s.config.BaseURL,
					s.config.BasePath,
					token,
					callbackURL,
				)
				go func() {
					if err := s.config.EmailVerification.SendVerificationEmail(*user, url, token); err != nil {
						slog.Error("failed to send verification email", "user_id", user.ID, "error", err)
					}
				}()
			}
		}
	}

	return &SignInResult{
		User:  user,
		Token: token,
	}, nil
}
