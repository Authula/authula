package services

import (
	"context"
	"fmt"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type blacklistService struct {
	storage models.SecondaryStorage
	logger  models.Logger
}

// NewBlacklistService creates a new blacklist service
func NewBlacklistService(storage models.SecondaryStorage, logger models.Logger) BlacklistService {
	return &blacklistService{
		storage: storage,
		logger:  logger,
	}
}

func (s *blacklistService) BlacklistToken(ctx context.Context, jti string, expiresAt time.Time) error {
	if jti == "" {
		return fmt.Errorf("jti cannot be empty")
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}

	key := s.blacklistKey(jti)

	if err := s.storage.Set(ctx, key, "1", &ttl); err != nil {
		s.logger.Error("failed to blacklist token", "jti", jti, "error", err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

func (s *blacklistService) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}

	key := s.blacklistKey(jti)

	value, err := s.storage.Get(ctx, key)
	if err != nil {
		s.logger.Error("failed to check blacklist", "jti", jti, "error", err)
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	// Key not found means not blacklisted
	if value == nil {
		return false, nil
	}

	return true, nil
}

func (s *blacklistService) BlacklistAllSessionTokens(ctx context.Context, sessionID string, expiresAt time.Time) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}

	key := s.sessionBlacklistKey(sessionID)

	if err := s.storage.Set(ctx, key, "1", &ttl); err != nil {
		s.logger.Error("failed to blacklist session tokens", "session_id", sessionID, "error", err)
		return fmt.Errorf("failed to blacklist session tokens: %w", err)
	}

	return nil
}

func (s *blacklistService) CleanupExpired(ctx context.Context) error {
	// With storage TTL, cleanup happens automatically
	// This method is a no-op for cache-based implementation
	// If using database storage, implement cleanup logic here
	return nil
}

func (s *blacklistService) blacklistKey(jti string) string {
	return fmt.Sprintf("jwt:blacklist:token:%s", jti)
}

func (s *blacklistService) sessionBlacklistKey(sessionID string) string {
	return fmt.Sprintf("jwt:blacklist:session:%s", sessionID)
}
