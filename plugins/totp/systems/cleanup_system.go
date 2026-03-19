package systems

import (
	"context"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/totp/types"
)

// TrustedDeviceRepository is the subset of the TOTP repository required by the cleanup system.
type TrustedDeviceRepository interface {
	DeleteExpiredTrustedDevices(ctx context.Context) error
}

// TrustedDevicesCleanupSystem periodically deletes expired trusted device records.
type TrustedDevicesCleanupSystem struct {
	logger      models.Logger
	config      *types.TOTPPluginConfig
	repo        TrustedDeviceRepository
	stopCleanup chan struct{}
	done        chan struct{}
}

func NewTrustedDevicesCleanupSystem(
	logger models.Logger,
	config *types.TOTPPluginConfig,
	repo TrustedDeviceRepository,
) *TrustedDevicesCleanupSystem {
	return &TrustedDevicesCleanupSystem{
		logger:      logger,
		config:      config,
		repo:        repo,
		stopCleanup: make(chan struct{}),
		done:        make(chan struct{}),
	}
}

func (s *TrustedDevicesCleanupSystem) Name() string {
	return "TrustedDevicesCleanupSystem"
}

func (s *TrustedDevicesCleanupSystem) Init(ctx context.Context) error {
	if !s.config.TrustedDevicesAutoCleanup {
		return nil
	}

	interval := s.config.TrustedDevicesCleanupInterval
	if interval <= 0 {
		interval = time.Hour
	}

	go s.runCleanupLoop(interval)

	return nil
}

func (s *TrustedDevicesCleanupSystem) Close() error {
	if !s.config.TrustedDevicesAutoCleanup {
		return nil
	}

	close(s.stopCleanup)
	<-s.done
	return nil
}

func (s *TrustedDevicesCleanupSystem) runCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case <-s.stopCleanup:
			s.logger.Debug("trusted devices cleanup loop stopped")
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := s.repo.DeleteExpiredTrustedDevices(ctx); err != nil {
				s.logger.Error("trusted devices expired cleanup failed", "error", err)
			}
			cancel()
		}
	}
}
