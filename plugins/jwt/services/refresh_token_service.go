package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/constants"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/events"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/types"
	coreservices "github.com/GoBetterAuth/go-better-auth/services"
)

type refreshTokenService struct {
	logger           models.Logger
	eventBus         models.EventBus
	sessionService   coreservices.SessionService
	jwtService       JwtService
	storage          RefreshTokenRepository
	gracePeriod      time.Duration
	refreshExpiresIn time.Duration
}

// NewRefreshTokenService creates a new refresh token service
func NewRefreshTokenService(
	logger models.Logger,
	eventBus models.EventBus,
	sessionService coreservices.SessionService,
	jwtService JwtService,
	storage RefreshTokenRepository,
	gracePeriod time.Duration,
	refreshExpiresIn time.Duration,
) RefreshTokenService {
	return &refreshTokenService{
		logger:           logger,
		eventBus:         eventBus,
		sessionService:   sessionService,
		jwtService:       jwtService,
		storage:          storage,
		gracePeriod:      gracePeriod,
		refreshExpiresIn: refreshExpiresIn,
	}
}

func (s *refreshTokenService) RefreshTokens(ctx context.Context, refreshToken string) (*RefreshTokenResponse, error) {
	return s.RefreshTokensWithMetadata(ctx, refreshToken, events.AuditMetadata{})
}

// RefreshTokensWithMetadata refreshes tokens with optional audit metadata for event logging
func (s *refreshTokenService) RefreshTokensWithMetadata(ctx context.Context, refreshToken string, auditMeta events.AuditMetadata) (*RefreshTokenResponse, error) {
	// Hash the incoming refresh token
	tokenHash := HashRefreshToken(refreshToken)

	// Check if token exists in database and is not revoked
	record, err := s.storage.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		s.logger.Error("refresh token not found in database", "error", err)
		return nil, fmt.Errorf("invalid refresh token")
	}
	if record == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check if token is revoked - THREE-TIER REUSE ATTACK DETECTION
	if record.IsRevoked {
		now := time.Now()
		revokedAt := record.RevokedAt
		if revokedAt == nil {
			// Should not happen, but handle gracefully
			s.logger.Error("token marked revoked but has no revoked_at timestamp", "session_id", record.SessionID)
			return nil, fmt.Errorf("invalid refresh token")
		}

		gracePeriodMs := s.gracePeriod.Milliseconds()

		// Tier 1: First reuse within grace period - RECOVERY
		// This handles legitimate concurrent requests or quick retries
		if record.LastReuseAttempt == nil {
			deltaMs := now.Sub(*revokedAt).Milliseconds()
			if deltaMs <= gracePeriodMs {
				s.logger.Warn("[AUTH_REUSE_RECOVERY] Refresh token reuse detected within grace period. Recovering.",
					"session_id", record.SessionID,
					"delta_ms", deltaMs,
					"grace_period_ms", gracePeriodMs,
				)

				// Update LastReuseAttempt to mark this reuse
				if err := s.storage.SetLastReuseAttempt(ctx, tokenHash); err != nil {
					s.logger.Error("failed to set last reuse attempt", "error", err)
				}

				// Emit recovery event
				s.emitTokenReuseRecoveredEvent(record.SessionID, tokenHash, deltaMs, gracePeriodMs, auditMeta)

				// Continue with normal token rotation - user stays logged in
				return s.completeTokenRotation(ctx, tokenHash, record)
			}
		}

		// Tier 2: Repeated reuse within grace period - THROTTLE
		// Multiple reuses within short window suggests possible attack or bug
		deltaMs := now.Sub(*revokedAt).Milliseconds()
		if deltaMs <= gracePeriodMs {
			s.logger.Warn("[AUTH_REUSE_THROTTLED] Repeated token reuse within grace period. Rejecting to prevent spam.",
				"session_id", record.SessionID,
				"delta_ms", deltaMs,
				"grace_period_ms", gracePeriodMs,
			)

			// Emit throttled event
			s.emitTokenReuseThrottledEvent(record.SessionID, tokenHash, deltaMs, gracePeriodMs, auditMeta)

			// Reject without killing session (may be legitimate concurrent retry)
			return nil, fmt.Errorf("invalid refresh token")
		}

		// Tier 3: Reuse after grace period - REJECT (but don't revoke session)
		// This could be a legitimate delayed retry (e.g., network issues, offline for a while)
		// We reject the token but don't assume it's an attack requiring session revocation
		s.logger.Debug("[AUTH_REUSE_DELAYED] Refresh token reuse detected after grace period. Rejecting token.",
			"session_id", record.SessionID,
			"delta_ms", deltaMs,
			"grace_period_ms", gracePeriodMs,
		)

		return nil, fmt.Errorf("invalid refresh token")
	}

	// Token not revoked - normal rotation flow
	return s.completeTokenRotation(ctx, tokenHash, record)
}

// completeTokenRotation handles the token rotation after validation passes
func (s *refreshTokenService) completeTokenRotation(ctx context.Context, tokenHash string, record *types.RefreshToken) (*RefreshTokenResponse, error) {
	// Check if token is expired
	if time.Now().After(record.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// Verify session still exists and is valid
	session, err := s.sessionService.GetByID(ctx, record.SessionID)
	if err != nil {
		s.logger.Error("session lookup failed", "session_id", record.SessionID, "error", err)
		return nil, fmt.Errorf("session expired or invalid")
	}

	if session == nil {
		s.logger.Error("session not found", "session_id", record.SessionID)
		return nil, fmt.Errorf("session expired or invalid")
	}

	// STEP 1: Revoke the old refresh token (rotation)
	if err := s.storage.RevokeRefreshToken(ctx, tokenHash); err != nil {
		s.logger.Error("failed to revoke old refresh token", "error", err)
		return nil, fmt.Errorf("failed to rotate token")
	}

	// STEP 2: Generate new token pair
	tokenPair, err := s.jwtService.GenerateTokens(ctx, session.UserID, record.SessionID)
	if err != nil {
		s.logger.Error("failed to generate new tokens", "user_id", session.UserID, "session_id", record.SessionID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens")
	}

	// STEP 3: Store new refresh token in database
	newTokenHash := HashRefreshToken(tokenPair.RefreshToken)
	expiresAt := time.Now().Add(s.refreshExpiresIn)

	newRecord := &types.RefreshToken{
		ID:        uuid.New().String(),
		SessionID: record.SessionID,
		TokenHash: newTokenHash,
		ExpiresAt: expiresAt,
		IsRevoked: false,
		CreatedAt: time.Now(),
	}

	if err := s.storage.StoreRefreshToken(ctx, newRecord); err != nil {
		s.logger.Error("failed to store new refresh token", "error", err)
		// Token was already revoked, this is critical
		return nil, fmt.Errorf("failed to rotate token")
	}

	return &RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

// Event emission methods - all fail-open to prioritize availability
func (s *refreshTokenService) emitTokenReuseRecoveredEvent(sessionID, tokenHash string, deltaMs, gracePeriodMs int64, meta events.AuditMetadata) {
	if s.eventBus == nil {
		return
	}

	event := &events.TokenReuseRecoveredEvent{
		Type:              constants.EventTokenReuseRecovered,
		SessionID:         sessionID,
		TokenHash:         tokenHash,
		DeltaMs:           deltaMs,
		GracePeriodConfig: fmt.Sprintf("%dms", gracePeriodMs),
		Metadata:          meta,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(event)
	eventMsg := models.Event{
		Type:      constants.EventTokenReuseRecovered,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	// Publish event asynchronously - don't block on event bus
	util.PublishEventAsync(s.eventBus, s.logger, eventMsg)
}

func (s *refreshTokenService) emitTokenReuseThrottledEvent(sessionID, tokenHash string, deltaMs, gracePeriodMs int64, meta events.AuditMetadata) {
	if s.eventBus == nil {
		return
	}

	event := &events.TokenReuseThrottledEvent{
		Type:              constants.EventTokenReuseThrottled,
		SessionID:         sessionID,
		TokenHash:         tokenHash,
		DeltaMs:           deltaMs,
		GracePeriodConfig: fmt.Sprintf("%dms", gracePeriodMs),
		AttemptCount:      2, // Second attempt within grace period
		Metadata:          meta,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(event)
	eventMsg := models.Event{
		Type:      constants.EventTokenReuseThrottled,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	// Publish event asynchronously - don't block on event bus
	util.PublishEventAsync(s.eventBus, s.logger, eventMsg)
}

func (s *refreshTokenService) emitTokenReuseMaliciousEvent(sessionID, tokenHash string, deltaMs, gracePeriodMs int64, meta events.AuditMetadata) {
	if s.eventBus == nil {
		return
	}

	event := &events.TokenReuseMaliciousEvent{
		Type:              constants.EventTokenReuseMalicious,
		SessionID:         sessionID,
		TokenHash:         tokenHash,
		DeltaMs:           deltaMs,
		GracePeriodConfig: fmt.Sprintf("%dms", gracePeriodMs),
		Metadata:          meta,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(event)
	eventMsg := models.Event{
		Type:      constants.EventTokenReuseMalicious,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	// Publish event asynchronously - don't block on event bus
	util.PublishEventAsync(s.eventBus, s.logger, eventMsg)
}

// StoreInitialRefreshToken stores the first refresh token when user logs in
func (s *refreshTokenService) StoreInitialRefreshToken(ctx context.Context, refreshToken, sessionID string, expiresAt time.Time) error {
	tokenHash := HashRefreshToken(refreshToken)

	record := &types.RefreshToken{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		IsRevoked: false,
		CreatedAt: time.Now(),
	}

	return s.storage.StoreRefreshToken(ctx, record)
}

// HashRefreshToken creates a SHA256 hash of a refresh token
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
