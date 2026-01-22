package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/types"
)

// GenerateTokens creates access and refresh JWT tokens tied to a session
func (p *JWTPlugin) GenerateTokens(secret string, userID string, sessionID string) (*types.TokenPair, error) {
	if sessionID == "" {
		return nil, errors.New("session id is required to generate tokens")
	}

	jwksKey, err := p.keyService.GetActiveKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	privateKeyPEM, err := p.tokenService.Decrypt(jwksKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	privKey, err := jwk.ParseKey([]byte(privateKeyPEM), jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Set the Key ID (kid) on the key so it's included in the JWT header
	if err := privKey.Set(jwk.KeyIDKey, jwksKey.ID); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}

	keyAlgorithm := p.detectAlgorithmFromKey(privKey)

	now := time.Now()
	jti := uuid.New().String()

	accessClaims := jwt.New()
	accessClaims.Set(jwt.SubjectKey, userID)
	accessClaims.Set(jwt.IssuedAtKey, now)
	accessClaims.Set(jwt.ExpirationKey, now.Add(p.pluginConfig.ExpiresIn))
	accessClaims.Set(jwt.JwtIDKey, jti)
	accessClaims.Set("user_id", userID)
	accessClaims.Set("session_id", sessionID)
	accessClaims.Set("type", "access")

	accessTokenBytes, err := jwt.Sign(accessClaims, jwt.WithKey(keyAlgorithm, privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := jwt.New()
	refreshClaims.Set(jwt.SubjectKey, userID)
	refreshClaims.Set(jwt.IssuedAtKey, now)
	refreshClaims.Set(jwt.ExpirationKey, now.Add(p.pluginConfig.RefreshExpiresIn))
	refreshClaims.Set(jwt.JwtIDKey, jti+"_refresh")
	refreshClaims.Set("user_id", userID)
	refreshClaims.Set("session_id", sessionID)
	refreshClaims.Set("type", "refresh")

	refreshTokenBytes, err := jwt.Sign(refreshClaims, jwt.WithKey(keyAlgorithm, privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &types.TokenPair{
		AccessToken:  string(accessTokenBytes),
		RefreshToken: string(refreshTokenBytes),
		ExpiresIn:    p.pluginConfig.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates a JWT token and ensures the referenced session is still active
func (p *JWTPlugin) ValidateToken(token string) (userID string, err error) {
	jwkSet, err := p.cacheService.GetJWKSWithFallback(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get JWKS: %w", err)
	}

	parsedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(jwkSet), jwt.WithValidate(true))
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	jti, ok := parsedToken.JwtID()
	if ok && jti != "" && p.blacklistService != nil {
		isBlacklisted, err := p.blacklistService.IsBlacklisted(context.Background(), jti)
		if err != nil {
			p.Logger.Error("failed to check token blacklist", "jti", jti, "error", err)
			// Don't fail validation on blacklist check error, but log it
		} else if isBlacklisted {
			return "", errors.New("token has been revoked")
		}
	}

	var tokenType string
	if err := parsedToken.Get("type", &tokenType); err != nil {
		return "", errors.New("missing token type claim")
	}

	if tokenType != "access" {
		return "", errors.New("invalid token type")
	}

	var extractedUserID string
	if err := parsedToken.Get("user_id", &extractedUserID); err != nil {
		return "", errors.New("missing user_id claim")
	}

	if extractedUserID == "" {
		return "", errors.New("missing user_id claim")
	}

	var sessionID string
	if err := parsedToken.Get("session_id", &sessionID); err != nil {
		return "", errors.New("missing session_id claim")
	}

	if sessionID == "" {
		return "", errors.New("missing session_id claim")
	}

	if p.blacklistService != nil {
		isBlacklisted, err := p.blacklistService.IsBlacklisted(context.Background(), "session:"+sessionID)
		if err != nil {
			p.Logger.Error("failed to check session blacklist", "session_id", sessionID, "error", err)
		} else if isBlacklisted {
			return "", errors.New("session has been revoked")
		}
	}

	if err := p.ensureSessionActive(context.Background(), extractedUserID, sessionID); err != nil {
		return "", err
	}

	return extractedUserID, nil
}

// ValidateRefreshToken validates a refresh token and ensures the session is active
func (p *JWTPlugin) ValidateRefreshToken(token string) (userID string, sessionID string, err error) {
	jwkSet, err := p.cacheService.GetJWKSWithFallback(context.Background())
	if err != nil {
		return "", "", fmt.Errorf("failed to get JWKS: %w", err)
	}

	parsedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(jwkSet), jwt.WithValidate(true))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse token: %w", err)
	}

	jti, ok := parsedToken.JwtID()
	if ok && jti != "" && p.blacklistService != nil {
		isBlacklisted, err := p.blacklistService.IsBlacklisted(context.Background(), jti)
		if err != nil {
			p.Logger.Error("failed to check refresh token blacklist", "jti", jti, "error", err)
		} else if isBlacklisted {
			return "", "", errors.New("refresh token has been revoked")
		}
	}

	var tokenType string
	if err := parsedToken.Get("type", &tokenType); err != nil {
		return "", "", errors.New("missing token type claim")
	}

	if tokenType != "refresh" {
		return "", "", errors.New("invalid token type")
	}

	var extractedUserID string
	if err := parsedToken.Get("user_id", &extractedUserID); err != nil {
		return "", "", errors.New("missing user_id claim")
	}

	if extractedUserID == "" {
		return "", "", errors.New("missing user_id claim")
	}

	var extractedSessionID string
	if err := parsedToken.Get("session_id", &extractedSessionID); err != nil {
		return "", "", errors.New("missing session_id claim")
	}

	if extractedSessionID == "" {
		return "", "", errors.New("missing session_id claim")
	}

	if p.blacklistService != nil {
		isBlacklisted, err := p.blacklistService.IsBlacklisted(context.Background(), "session:"+extractedSessionID)
		if err != nil {
			p.Logger.Error("failed to check session blacklist for refresh token", "session_id", extractedSessionID, "error", err)
		} else if isBlacklisted {
			return "", "", errors.New("session has been revoked")
		}
	}

	if err := p.ensureSessionActive(context.Background(), extractedUserID, extractedSessionID); err != nil {
		return "", "", err
	}

	return extractedUserID, extractedSessionID, nil
}

// ensureSessionActive verifies that the session still exists and belongs to the user
func (p *JWTPlugin) ensureSessionActive(ctx context.Context, userID, sessionID string) error {
	session, err := p.sessionService.GetByID(ctx, sessionID)
	if err != nil {
		if session == nil {
			return errors.New("session not found")
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.UserID != userID {
		return errors.New("session does not belong to user")
	}

	if session.ExpiresAt.Before(time.Now().UTC()) {
		return errors.New("session has expired")
	}

	return nil
}

// detectAlgorithmFromKey detects the algorithm from a JWK key and validates it
func (p *JWTPlugin) detectAlgorithmFromKey(k jwk.Key) jwa.SignatureAlgorithm {
	// Try to get algorithm from key
	if alg, ok := k.Algorithm(); ok {
		if sigAlg, ok := alg.(jwa.SignatureAlgorithm); ok {
			return sigAlg
		}
	}

	// Fallback: determine from key type string
	keyType := k.KeyType().String()
	var detectedAlg jwa.SignatureAlgorithm
	switch keyType {
	case "OKP":
		detectedAlg = jwa.EdDSA()
	case "RSA":
		detectedAlg = jwa.RS256()
	case "EC":
		detectedAlg = jwa.ES256()
	case "oct":
		detectedAlg = jwa.HS256()
	default:
		detectedAlg = jwa.EdDSA() // Default to EdDSA
	}

	// Validate that detected algorithm is compatible with configured algorithm
	configAlg := strings.ToLower(p.pluginConfig.Algorithm.String())
	detectedStr := strings.ToLower(detectedAlg.String())
	if !strings.Contains(configAlg, detectedStr) && !strings.Contains(detectedStr, configAlg) {
		p.Logger.Warn("detected algorithm differs from config",
			"configured", configAlg,
			"detected", detectedStr,
			"key_type", keyType)
	}

	return detectedAlg
}
