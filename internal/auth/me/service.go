package me

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type service struct {
	config         *models.Config
	logger         models.Logger
	userService    models.UserService
	sessionService models.SessionService
}

func New(
	config *models.Config,
	logger models.Logger,
	userService models.UserService,
	sessionService models.SessionService,
) *service {
	return &service{
		config:         config,
		logger:         logger,
		userService:    userService,
		sessionService: sessionService,
	}
}

// GetMe retrieves the current user and their session
func (s *service) GetMe(ctx context.Context, userID string) (*models.MeResult, error) {
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	session, err := s.sessionService.GetSessionByUserID(userID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	return &models.MeResult{
		User:    user,
		Session: session,
	}, nil
}
