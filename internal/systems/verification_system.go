package systems

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

type VerificationSystem struct {
	logger              models.Logger
	config              models.VerificationConfig
	verificationService services.VerificationService
	stopCleanup         chan struct{}
	done                chan struct{}
}

func NewVerificationSystem(
	logger models.Logger,
	config models.VerificationConfig,
	verificationService services.VerificationService,
) *VerificationSystem {
	return &VerificationSystem{
		verificationService: verificationService,
		config:              config,
		logger:              logger,
		stopCleanup:         make(chan struct{}),
		done:                make(chan struct{}),
	}
}

func (s *VerificationSystem) Name() string {
	return "Verification"
}

func (s *VerificationSystem) Init(ctx context.Context) error {
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

func (s *VerificationSystem) Close() error {
	if !s.config.AutoCleanup {
		return nil
	}

	close(s.stopCleanup)
	<-s.done
	return nil
}

func (s *VerificationSystem) runCleanupLoop(interval time.Duration) {
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
