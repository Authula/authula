package signout

import (
	"context"
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/internal/constants"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type service struct {
	config         *models.Config
	logger         models.Logger
	sessionService models.SessionService
	tokenService   models.TokenService
}

func New(
	config *models.Config,
	logger models.Logger,
	sessionService models.SessionService,
	tokenService models.TokenService,
) *service {
	return &service{
		config:         config,
		logger:         logger,
		sessionService: sessionService,
		tokenService:   tokenService,
	}
}

func (s *service) SignOut(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return constants.ErrInvalidToken
	}

	// Find session by token
	sess, err := s.sessionService.GetSessionByToken(s.tokenService.HashToken(sessionToken))
	if err != nil {
		s.logger.Error("failed to get session by token", "error", err)
		return fmt.Errorf("%w: %w", constants.ErrSessionNotFound, err)
	}
	if sess == nil {
		return constants.ErrSessionNotFound
	}

	// Delete the session
	if err := s.sessionService.DeleteSessionByID(sess.ID); err != nil {
		s.logger.Error("failed to delete session", "session_id", sess.ID, "error", err)
		return fmt.Errorf("%w: %w", constants.ErrSessionDeletionFailed, err)
	}

	return nil
}
