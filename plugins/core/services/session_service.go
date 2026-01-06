package services

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/repositories"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/security"
	"github.com/GoBetterAuth/go-better-auth/plugins/core/types"
	"github.com/GoBetterAuth/go-better-auth/services"
)

type sessionService struct {
	repo   repositories.SessionRepository
	signer security.TokenSigner
	hooks  *types.CoreDatabaseHooks
}

func NewSessionService(repo repositories.SessionRepository, signer security.TokenSigner, hooks *types.CoreDatabaseHooks) services.SessionService {
	return &sessionService{
		repo:   repo,
		signer: signer,
		hooks:  hooks,
	}
}

func (s *sessionService) Create(
	ctx context.Context,
	userID string,
	hashedToken string,
	ipAddress *string,
	userAgent *string,
	maxAge time.Duration,
) (*models.Session, error) {
	if hashedToken == "" {
		return nil, fmt.Errorf("hashedToken cannot be empty")
	}

	session := &models.Session{
		ID:        util.GenerateUUID(),
		UserID:    userID,
		Token:     hashedToken,
		ExpiresAt: time.Now().UTC().Add(maxAge),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if s.hooks != nil && s.hooks.Sessions != nil && s.hooks.Sessions.BeforeCreate != nil {
		if err := s.hooks.Sessions.BeforeCreate(session); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	if s.hooks != nil && s.hooks.Sessions != nil && s.hooks.Sessions.AfterCreate != nil {
		if err := s.hooks.Sessions.AfterCreate(*created); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func (s *sessionService) GetByID(ctx context.Context, id string) (*models.Session, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *sessionService) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *sessionService) GetByToken(ctx context.Context, hashedToken string) (*models.Session, error) {
	return s.repo.GetByToken(ctx, hashedToken)
}

func (s *sessionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *sessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	return s.repo.DeleteByUserID(ctx, userID)
}
