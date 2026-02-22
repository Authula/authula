package verification

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type VerificationCleanupSystem struct {
	logger              models.Logger
	config              models.VerificationConfig
	verificationService services.VerificationService
	stopCleanup         chan struct{}
	done                chan struct{}
}

func NewVerificationCleanupSystem(
	logger models.Logger,
	config models.VerificationConfig,
	verificationService services.VerificationService,
) *VerificationCleanupSystem {
	return &VerificationCleanupSystem{
		verificationService: verificationService,
		config:              config,
		logger:              logger,
		stopCleanup:         make(chan struct{}),
		done:                make(chan struct{}),
	}
}

func (s *VerificationCleanupSystem) Name() string {
	return "VerificationCleanupSystem"
}

func (s *VerificationCleanupSystem) Init(ctx context.Context) error {
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

func (s *VerificationCleanupSystem) Close() error {
	if !s.config.AutoCleanup {
		return nil
	}

	close(s.stopCleanup)
	<-s.done
	return nil
}

func (s *VerificationCleanupSystem) runCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case <-s.stopCleanup:
			s.logger.Debug("Verification cleanup loop stopped")
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := s.verificationService.DeleteExpired(ctx); err != nil {
				s.logger.Error("Verification expired cleanup failed", "error", err)
			}
			cancel()
		}
	}
}
