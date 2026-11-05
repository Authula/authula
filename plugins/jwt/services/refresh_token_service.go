package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/constants"
	"github.com/Authula/authula/plugins/jwt/events"
	"github.com/Authula/authula/plugins/jwt/types"
	coreservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type refreshTokenService struct {
	logger           models.Logger
	eventBus         models.EventBus
	sessionService   coreservices.SessionService
	jwtService       TokenService
	storage          RefreshTokenRepository
	blacklistService BlacklistService
	gracePeriod      time.Duration
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
}

func NewRefreshTokenService(
	logger models.Logger,
	eventBus models.EventBus,
	sessionService coreservices.SessionService,
	jwtService TokenService,
	storage RefreshTokenRepository,
	blacklistService BlacklistService,
	gracePeriod time.Duration,
	expiresIn time.Duration,
	refreshExpiresIn time.Duration,
) RefreshTokenService {
	return &refreshTokenService{
		logger:           logger,
		eventBus:         eventBus,
		sessionService:   sessionService,
		jwtService:       jwtService,
		storage:          storage,
		blacklistService: blacklistService,
		gracePeriod:      gracePeriod,
		expiresIn:        expiresIn,
		refreshExpiresIn: refreshExpiresIn,
	}
}

func (s *refreshTokenService) RefreshTokens(ctx context.Context, refreshToken string) (*types.RefreshTokenResponse, error) {
	return s.RefreshTokensWithMetadata(ctx, refreshToken, events.AuditMetadata{})
}

// RefreshTokensWithMetadata refreshes tokens with optional audit metadata for event logging.
func (s *refreshTokenService) RefreshTokensWithMetadata(ctx context.Context, refreshToken string, auditMeta events.AuditMetadata) (*types.RefreshTokenResponse, error) {
	claims, err := s.jwtService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		s.logger.Debug("refresh token validation failed", "error", err)
		return nil, constants.ErrInvalidRefreshToken
	}

	tokenHash := HashRefreshToken(refreshToken)

	record, err := s.storage.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		s.logger.Error("refresh token lookup failed", "error", err)
		return nil, constants.ErrInvalidRefreshToken
	}
	if record == nil {
		return nil, constants.ErrInvalidRefreshToken
	}

	claimSessionID, _ := claims["session_id"].(string)
	if claimSessionID == "" || claimSessionID != record.SessionID {
		return nil, constants.ErrInvalidRefreshToken
	}

	// A revoked token is never exchanged for new credentials. Reuse within the
	// grace window is rejected without side effects; reuse after it is treated as
	// an attack and invalidates the whole session.
	if record.IsRevoked {
		return nil, s.handleRevokedToken(ctx, tokenHash, record, auditMeta)
	}

	if !time.Now().Before(record.ExpiresAt) {
		return nil, constants.ErrInvalidRefreshToken
	}

	session, err := s.sessionService.GetByID(ctx, record.SessionID)
	if err != nil {
		s.logger.Error("session lookup failed", "session_id", record.SessionID, "error", err)
		return nil, constants.ErrInvalidRefreshToken
	}
	if session == nil || !session.ExpiresAt.After(time.Now()) {
		s.logger.Warn("refresh token belongs to a missing or expired session", "session_id", record.SessionID)
		return nil, constants.ErrInvalidRefreshToken
	}

	claimUserID, _ := claims["user_id"].(string)
	if claimUserID == "" || claimUserID != session.UserID {
		return nil, constants.ErrInvalidRefreshToken
	}

	extraClaims := s.extraClaims(claims)

	tokenPair, err := s.jwtService.GenerateUserToken(ctx, session.UserID, record.SessionID, extraClaims)
	if err != nil {
		s.logger.Error("failed to generate new tokens", "user_id", session.UserID, "session_id", record.SessionID, "error", err)
		return nil, errors.New("failed to generate tokens")
	}

	newRecord := &types.RefreshToken{
		ID:        uuid.New().String(),
		SessionID: record.SessionID,
		TokenHash: HashRefreshToken(tokenPair.RefreshToken),
		ExpiresAt: time.Now().Add(s.refreshExpiresIn),
		IsRevoked: false,
		CreatedAt: time.Now(),
	}

	rotated, err := s.storage.RotateRefreshToken(ctx, tokenHash, newRecord)
	if err != nil {
		s.logger.Error("failed to rotate refresh token", "session_id", record.SessionID, "error", err)
		return nil, constants.ErrInvalidRefreshToken
	}
	if !rotated {
		return nil, s.handleRotationConflict(ctx, tokenHash, auditMeta)
	}

	return &types.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.expiresIn.Seconds()),
	}, nil
}

// handleRotationConflict runs when another request rotated the token first. The
// token is re-read and handled as reuse, never as a second rotation.
func (s *refreshTokenService) handleRotationConflict(ctx context.Context, tokenHash string, auditMeta events.AuditMetadata) error {
	latest, err := s.storage.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		s.logger.Error("refresh token re-read failed", "error", err)
		return constants.ErrInvalidRefreshToken
	}
	if latest == nil {
		return constants.ErrInvalidRefreshToken
	}
	if latest.IsRevoked {
		return s.handleRevokedToken(ctx, tokenHash, latest, auditMeta)
	}
	return constants.ErrInvalidRefreshToken
}

// handleRevokedToken rejects a revoked refresh token. Reuse inside the grace
// period is throttled; reuse after it revokes the session family.
func (s *refreshTokenService) handleRevokedToken(ctx context.Context, tokenHash string, record *types.RefreshToken, auditMeta events.AuditMetadata) error {
	if record.RevokedAt == nil {
		s.logger.Error("token marked revoked but has no revoked_at timestamp", "session_id", record.SessionID)
		return constants.ErrInvalidRefreshToken
	}

	delta := time.Since(*record.RevokedAt)
	gracePeriodMs := s.gracePeriod.Milliseconds()

	if delta <= s.gracePeriod {
		deltaMs := delta.Milliseconds()
		s.logger.Warn("[AUTH_REUSE_THROTTLED] Refresh token reuse within the grace period rejected",
			"session_id", record.SessionID,
			"delta_ms", deltaMs,
			"grace_period_ms", gracePeriodMs,
		)
		s.emitTokenReuseThrottledEvent(record.SessionID, tokenHash, deltaMs, gracePeriodMs, auditMeta)
		return constants.ErrInvalidRefreshToken
	}

	deltaMs := delta.Milliseconds()
	s.logger.Warn("[AUTH_REUSE_MALICIOUS] Refresh token reuse after the grace period invalidates the session",
		"session_id", record.SessionID,
		"delta_ms", deltaMs,
		"grace_period_ms", gracePeriodMs,
	)

	s.emitTokenReuseMaliciousEvent(record.SessionID, tokenHash, deltaMs, gracePeriodMs, auditMeta)
	s.invalidateSessionFamily(ctx, record.SessionID)

	return constants.ErrInvalidRefreshToken
}

// invalidateSessionFamily revokes every refresh token and the session itself.
func (s *refreshTokenService) invalidateSessionFamily(ctx context.Context, sessionID string) {
	if err := s.storage.RevokeAllSessionTokens(ctx, sessionID); err != nil {
		s.logger.Error("failed to revoke session refresh tokens", "session_id", sessionID, "error", err)
	}

	if err := s.blacklistService.BlacklistAllSessionTokens(ctx, sessionID, time.Now().Add(s.refreshExpiresIn)); err != nil {
		s.logger.Error("failed to blacklist session tokens", "session_id", sessionID, "error", err)
	}

	if err := s.sessionService.Delete(ctx, sessionID); err != nil {
		s.logger.Error("failed to delete invalidated session", "session_id", sessionID, "error", err)
	}
}

func (s *refreshTokenService) extraClaims(claims map[string]any) map[string]any {
	extraClaims := make(map[string]any)
	for k, v := range claims {
		switch k {
		case "iss", "aud", "sub", "user_id", "session_id", "iat", "exp", "nbf", "jti",
			"token_type", "actor_type", "organization_id", "scopes", "client_id":
			continue
		default:
			extraClaims[k] = v
		}
	}
	if len(extraClaims) == 0 {
		return nil
	}
	return extraClaims
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
		AttemptCount:      2,
		Metadata:          meta,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}

	eventMsg := models.Event{
		Type:      constants.EventTokenReuseThrottled,
		Timestamp: time.Now().UTC(),
		Payload:   util.ToMap(event),
	}

	util.PublishEventAsync(s.eventBus, s.logger, eventMsg)
}

func (s *refreshTokenService) emitTokenReuseMaliciousEvent(sessionID, tokenHash string, deltaMs, gracePeriodMs int64, meta events.AuditMetadata) {
	event := &events.TokenReuseMaliciousEvent{
		Type:              constants.EventTokenReuseMalicious,
		SessionID:         sessionID,
		TokenHash:         tokenHash,
		DeltaMs:           deltaMs,
		GracePeriodConfig: fmt.Sprintf("%dms", gracePeriodMs),
		Metadata:          meta,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}

	eventMsg := models.Event{
		Type:      constants.EventTokenReuseMalicious,
		Timestamp: time.Now().UTC(),
		Payload:   util.ToMap(event),
	}

	util.PublishEventAsync(s.eventBus, s.logger, eventMsg)
}

// StoreInitialRefreshToken stores the first refresh token when user logs in.
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

// HashRefreshToken creates a SHA256 hash of a refresh token.
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
