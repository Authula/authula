package services

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/security"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type sessionService struct {
	repo    repositories.SessionRepository
	signer  security.TokenSigner
	dbHooks *models.CoreDatabaseHooksConfig
}

func NewSessionService(
	repo repositories.SessionRepository,
	signer security.TokenSigner,
	dbHooks *models.CoreDatabaseHooksConfig,
) services.SessionService {
	return &sessionService{
		repo:    repo,
		signer:  signer,
		dbHooks: dbHooks,
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

	if s.dbHooks != nil && s.dbHooks.Sessions != nil && s.dbHooks.Sessions.BeforeCreate != nil {
		if err := s.dbHooks.Sessions.BeforeCreate(session); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Sessions != nil && s.dbHooks.Sessions.AfterCreate != nil {
		if err := s.dbHooks.Sessions.AfterCreate(*created); err != nil {
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

func (s *sessionService) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	if s.dbHooks != nil && s.dbHooks.Sessions != nil && s.dbHooks.Sessions.BeforeUpdate != nil {
		if err := s.dbHooks.Sessions.BeforeUpdate(session); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.Update(ctx, session)
	if err != nil {
		return nil, err
	}

	if s.dbHooks != nil && s.dbHooks.Sessions != nil && s.dbHooks.Sessions.AfterUpdate != nil {
		if err := s.dbHooks.Sessions.AfterUpdate(*updated); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

func (s *sessionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *sessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	return s.repo.DeleteByUserID(ctx, userID)
}
