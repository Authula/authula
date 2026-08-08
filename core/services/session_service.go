package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/core/security"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type sessionService struct {
	repo         repositories.SessionRepository
	signer       security.TokenSigner
	serviceHooks *models.CoreServiceHooksConfig
	logger       models.Logger
}

func NewSessionService(
	repo repositories.SessionRepository,
	signer security.TokenSigner,
	serviceHooks *models.CoreServiceHooksConfig,
	logger models.Logger,
) services.SessionService {
	return &sessionService{
		repo:         repo,
		signer:       signer,
		serviceHooks: serviceHooks,
		logger:       logger,
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

	if s.serviceHooks != nil && s.serviceHooks.Sessions != nil {
		if err := runHooks(s.serviceHooks.Sessions.BeforeCreateHooks(), session); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Sessions != nil {
		runAfterHooks(s.logger, "after create session", s.serviceHooks.Sessions.AfterCreateHooks(), created)
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
	if s.serviceHooks != nil && s.serviceHooks.Sessions != nil {
		if err := runHooks(s.serviceHooks.Sessions.BeforeUpdateHooks(), session); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.Update(ctx, session)
	if err != nil {
		return nil, err
	}

	if s.serviceHooks != nil && s.serviceHooks.Sessions != nil {
		runAfterHooks(s.logger, "after update session", s.serviceHooks.Sessions.AfterUpdateHooks(), updated)
	}

	return updated, nil
}

func (s *sessionService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *sessionService) DeleteAllByUserID(ctx context.Context, userID string) error {
	return s.repo.DeleteByUserID(ctx, userID)
}

func (s *sessionService) DeleteAllExpired(ctx context.Context) error {
	return s.repo.DeleteExpired(ctx)
}

func (s *sessionService) GetDistinctUserIDs(ctx context.Context) ([]string, error) {
	return s.repo.GetDistinctUserIDs(ctx)
}

func (s *sessionService) DeleteOldestByUserID(ctx context.Context, userID string, maxCount int) error {
	return s.repo.DeleteOldestByUserID(ctx, userID, maxCount)
}
