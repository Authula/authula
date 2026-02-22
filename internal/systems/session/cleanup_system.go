package session

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type SessionCleanupSystem struct {
	logger         models.Logger
	config         models.SessionConfig
	sessionService services.SessionService
	stopCleanup    chan struct{}
	done           chan struct{}
}

func NewSessionCleanupSystem(
	logger models.Logger,
	config models.SessionConfig,
	sessionService services.SessionService,
) *SessionCleanupSystem {
	return &SessionCleanupSystem{
		logger:         logger,
		config:         config,
		sessionService: sessionService,
		stopCleanup:    make(chan struct{}),
		done:           make(chan struct{}),
	}
}

func (s *SessionCleanupSystem) Name() string {
	return "SessionCleanupSystem"
}

func (s *SessionCleanupSystem) Init(ctx context.Context) error {
	if !s.config.AutoCleanup {
		return nil
	}

	interval := s.config.CleanupInterval
	if interval <= 0 {
		interval = time.Minute
	}

	go s.runCleanupLoop(interval)

	return nil
}

func (s *SessionCleanupSystem) Close() error {
	if !s.config.AutoCleanup {
		return nil
	}

	close(s.stopCleanup)
	<-s.done
	return nil
}

func (s *SessionCleanupSystem) runCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case <-s.stopCleanup:
			s.logger.Debug("session cleanup loop stopped")
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			s.cleanup(ctx)
			cancel()
		}
	}
}

func (s *SessionCleanupSystem) cleanup(ctx context.Context) {
	if err := s.sessionService.DeleteAllExpired(ctx); err != nil {
		s.logger.Error("session expired cleanup failed", "error", err)
	}

	if err := s.enforceMaxSessionsPerUser(ctx); err != nil {
		s.logger.Error("session max sessions enforcement failed", "error", err)
	}
}

func (s *SessionCleanupSystem) enforceMaxSessionsPerUser(ctx context.Context) error {
	if s.config.MaxSessionsPerUser <= 0 {
		return nil
	}

	userIDs, err := s.sessionService.GetDistinctUserIDs(ctx)
	if err != nil {
		return err
	}

	for _, userID := range userIDs {
		if err := s.sessionService.DeleteOldestByUserID(ctx, userID, s.config.MaxSessionsPerUser); err != nil {
			s.logger.Error("failed to enforce max sessions for user", "user_id", userID, "error", err)
		}
	}

	return nil
}
