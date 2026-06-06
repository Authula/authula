package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Authula/authula/models"
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

const (
	jwtBlacklistTokenPrefix   = "jwt:blacklist:token:"
	jwtBlacklistSessionPrefix = "jwt:blacklist:session:"
)

func (s *blacklistService) CleanupExpired(ctx context.Context) error {
	prefixes := []string{jwtBlacklistTokenPrefix, jwtBlacklistSessionPrefix}
	for _, prefix := range prefixes {
		keys, err := s.storage.Scan(ctx, prefix)
		if err != nil {
			s.logger.Error("failed to scan blacklist keys", "prefix", prefix, "error", err)
			return fmt.Errorf("failed to scan blacklist keys: %w", err)
		}
		for _, key := range keys {
			ttl, err := s.storage.TTL(ctx, key)
			if err != nil {
				s.logger.Error("failed to check TTL for key", "key", key, "error", err)
				continue
			}
			if ttl == nil || *ttl <= 0 {
				if err := s.storage.Delete(ctx, key); err != nil {
					s.logger.Error("failed to delete expired blacklist entry", "key", key, "error", err)
				}
			}
		}
	}
	return nil
}

func (s *blacklistService) blacklistKey(jti string) string {
	return fmt.Sprintf("%s%s", jwtBlacklistTokenPrefix, jti)
}

func (s *blacklistService) sessionBlacklistKey(sessionID string) string {
	return fmt.Sprintf("%s%s", jwtBlacklistSessionPrefix, sessionID)
}
