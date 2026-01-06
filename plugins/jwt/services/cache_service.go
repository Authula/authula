package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/repositories"
)

const jwksCacheKey = "jwks:public:set"

type cacheService struct {
	repo             repositories.JWKSRepository
	secondaryStorage models.SecondaryStorage
	logger           models.Logger
	cacheTTL         time.Duration
}

// NewCacheService creates a new cache service
func NewCacheService(repo repositories.JWKSRepository, secondaryStorage models.SecondaryStorage, logger models.Logger, cacheTTL time.Duration) CacheService {
	return &cacheService{
		repo:             repo,
		secondaryStorage: secondaryStorage,
		logger:           logger,
		cacheTTL:         cacheTTL,
	}
}

// GetCachedJWKS retrieves JWKS from cache if available and not expired
func (s *cacheService) GetCachedJWKS(ctx context.Context) (jwk.Set, error) {
	if s.secondaryStorage == nil {
		return nil, errors.New("secondary storage not available")
	}

	value, err := s.secondaryStorage.Get(ctx, jwksCacheKey)
	if err != nil {
		return nil, fmt.Errorf("cache miss: %w", err)
	}

	valueStr, ok := value.(string)
	if !ok || valueStr == "" {
		return nil, errors.New("cached JWKS is empty or invalid type")
	}

	set, err := jwk.Parse([]byte(valueStr))
	if err != nil {
		s.logger.Warn("failed to parse cached JWKS", "error", err)
		return nil, fmt.Errorf("failed to parse cached JWKS: %w", err)
	}

	return set, nil
}

// FetchJWKSFromDatabase loads all non-expired public keys from the database
func (s *cacheService) FetchJWKSFromDatabase(ctx context.Context) (jwk.Set, error) {
	jwksKeys, err := s.repo.GetJWKSKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch web keys: %w", err)
	}

	set := jwk.NewSet()
	for _, wk := range jwksKeys {
		pubKey, err := jwk.ParseKey([]byte(wk.PublicKey), jwk.WithPEM(true))
		if err != nil {
			s.logger.Warn("failed to parse public key", "id", wk.ID, "error", err)
			continue
		}

		_ = pubKey.Set(jwk.KeyIDKey, wk.ID)

		_ = set.AddKey(pubKey)
	}

	if set.Len() == 0 {
		return nil, errors.New("no valid keys found")
	}

	return set, nil
}

// CacheJWKS stores the JWKS in the cache with the configured TTL
func (s *cacheService) CacheJWKS(ctx context.Context, set jwk.Set) error {
	if s.secondaryStorage == nil {
		s.logger.Debug("secondary storage not available, skipping cache")
		return nil
	}

	data, err := json.Marshal(set)
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	if err := s.secondaryStorage.Set(ctx, jwksCacheKey, string(data), &s.cacheTTL); err != nil {
		return fmt.Errorf("failed to cache JWKS: %w", err)
	}

	s.logger.Debug("cached JWKS", "ttl", s.cacheTTL)
	return nil
}

// InvalidateCache removes the cached JWKS immediately and fetches fresh from DB
func (s *cacheService) InvalidateCache(ctx context.Context) error {
	if s.secondaryStorage == nil {
		return nil
	}

	if err := s.secondaryStorage.Delete(ctx, jwksCacheKey); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	set, err := s.FetchJWKSFromDatabase(ctx)
	if err != nil {
		s.logger.Warn("failed to fetch JWKS from database for cache", "error", err)
		return nil // Don't fail, cache will be populated on next access
	}

	return s.CacheJWKS(ctx, set)
}

// GetJWKSWithFallback retrieves JWKS from cache with database fallback
func (s *cacheService) GetJWKSWithFallback(ctx context.Context) (jwk.Set, error) {
	set, err := s.GetCachedJWKS(ctx)
	if err == nil {
		s.logger.Debug("retrieved JWKS from cache")
		return set, nil
	}

	s.logger.Debug("cache miss, fetching from database", "error", err)

	set, err = s.FetchJWKSFromDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from database: %w", err)
	}

	if err := s.CacheJWKS(ctx, set); err != nil {
		s.logger.Warn("failed to cache JWKS", "error", err)
		// Don't fail; still return the set
	}

	return set, nil
}
