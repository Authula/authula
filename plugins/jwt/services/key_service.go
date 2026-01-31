package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	coreservices "github.com/GoBetterAuth/go-better-auth/v2/services"
	"github.com/google/uuid"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
)

type keyService struct {
	repo         repositories.JWKSRepository
	logger       models.Logger
	secret       string
	algorithm    types.JWTAlgorithm
	tokenService coreservices.TokenService
}

// NewKeyService creates a new key service
func NewKeyService(repo repositories.JWKSRepository, logger models.Logger, tokenService coreservices.TokenService, secret string, algorithm types.JWTAlgorithm) KeyService {
	return &keyService{
		repo:         repo,
		logger:       logger,
		secret:       secret,
		algorithm:    algorithm,
		tokenService: tokenService,
	}
}

// GenerateKeysIfMissing generates the initial key pair if none exist
func (s *keyService) GenerateKeysIfMissing(ctx context.Context) error {
	keys, err := s.repo.GetJWKSKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) > 0 {
		return nil
	}

	return s.generateAndStoreKey(ctx)
}

// GetActiveKey retrieves the currently active (non-expired) key
func (s *keyService) GetActiveKey(ctx context.Context) (*types.JWKS, error) {
	keys, err := s.repo.GetJWKSKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, errors.New("no active key found")
	}

	// Return the most recent key
	var activeKey *types.JWKS
	for _, key := range keys {
		if activeKey == nil || key.CreatedAt.After(activeKey.CreatedAt) {
			activeKey = key
		}
	}

	return activeKey, nil
}

// IsKeyRotationDue returns true if the active key's age exceeds the rotation interval
func (s *keyService) IsKeyRotationDue(ctx context.Context, rotationInterval time.Duration) bool {
	key, err := s.GetActiveKey(ctx)
	if err != nil {
		return false
	}

	return time.Since(key.CreatedAt) > rotationInterval
}

// RotateKeysIfNeeded rotates keys if they're past the rotation interval
// gracePeriod specifies how long old keys remain valid after rotation
func (s *keyService) RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, gracePeriod time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error) {
	if !s.IsKeyRotationDue(ctx, rotationInterval) {
		return false, nil
	}

	now := time.Now()
	expirationTime := now.Add(gracePeriod)

	keys, err := s.repo.GetJWKSKeys(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get keys for rotation: %w", err)
	}

	for _, key := range keys {
		if err := s.repo.MarkKeyExpired(ctx, key.ID, expirationTime); err != nil {
			return false, fmt.Errorf("failed to set expiration on old key: %w", err)
		}
	}

	if err := s.generateAndStoreKey(ctx); err != nil {
		return false, fmt.Errorf("failed to generate new key: %w", err)
	}

	if invalidateCacheFunc != nil {
		if err := invalidateCacheFunc(ctx); err != nil {
			s.logger.Warn("failed to invalidate cache after key rotation", "error", err)
		}
	}

	s.logger.Info("key rotation completed", "grace_period", gracePeriod, "old_keys_expire_at", expirationTime)
	return true, nil
}

// generateKey returns a newly generated private/public key pair for the given algorithm
func generateKey(alg types.JWTAlgorithm) (priv any, pub any, err error) {
	switch alg {
	case types.JWTAlgRS256, types.JWTAlgPS256:
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, err
		}
		pub = &priv.(*rsa.PrivateKey).PublicKey
		return

	case types.JWTAlgES256:
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		pub = &priv.(*ecdsa.PrivateKey).PublicKey
		return

	case types.JWTAlgES512:
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		pub = &priv.(*ecdsa.PrivateKey).PublicKey
		return

	case types.JWTAlgEdDSA:
		var seed [32]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return nil, nil, fmt.Errorf("failed to read random seed: %w", err)
		}
		priv = ed25519.NewKeyFromSeed(seed[:])
		pub = priv.(ed25519.PrivateKey).Public()
		return

	case types.JWTAlgECDHES:
		// ECDH-ES uses EC P-256 keys for key agreement (future JWE)
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		pub = &priv.(*ecdsa.PrivateKey).PublicKey
		return

	default:
		return nil, nil, fmt.Errorf("unsupported algorithm: %s", alg)
	}
}

// generateAndStoreKey generates a key pair and stores it in the database
func (s *keyService) generateAndStoreKey(ctx context.Context) error {
	privKey, pubKey, err := generateKey(s.algorithm)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	privKeyPEM, err := privateKeyToPEM(privKey)
	if err != nil {
		return fmt.Errorf("failed to convert private key to PEM: %w", err)
	}

	pubKeyPEM, err := publicKeyToPEM(pubKey)
	if err != nil {
		return fmt.Errorf("failed to convert public key to PEM: %w", err)
	}

	encryptedPrivKey, err := s.tokenService.Encrypt(string(privKeyPEM))
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}

	jwksKey := types.JWKS{
		ID:         uuid.New().String(),
		PublicKey:  string(pubKeyPEM),
		PrivateKey: encryptedPrivKey,
		CreatedAt:  time.Now(),
		ExpiresAt:  nil,
	}

	if err := s.repo.StoreJWKSKey(ctx, &jwksKey); err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}

	s.logger.Info("generated and stored key", "id", jwksKey.ID, "algorithm", s.algorithm.String())
	return nil
}

// privateKeyToPEM converts a private key to PEM format
func privateKeyToPEM(privKey any) ([]byte, error) {
	var keyBytes []byte
	var keyType string

	switch pk := privKey.(type) {
	case *rsa.PrivateKey:
		keyBytes, _ = x509.MarshalPKCS8PrivateKey(pk)
		keyType = "PRIVATE KEY"
	case *ecdsa.PrivateKey:
		keyBytes, _ = x509.MarshalPKCS8PrivateKey(pk)
		keyType = "PRIVATE KEY"
	case ed25519.PrivateKey:
		keyBytes, _ = x509.MarshalPKCS8PrivateKey(pk)
		keyType = "PRIVATE KEY"
	default:
		return nil, errors.New("unsupported private key type")
	}

	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}

	return pem.EncodeToMemory(block), nil
}

// publicKeyToPEM converts a public key to PEM format
func publicKeyToPEM(pubKey any) ([]byte, error) {
	var keyBytes []byte
	var keyType string

	switch pk := pubKey.(type) {
	case *rsa.PublicKey:
		keyBytes, _ = x509.MarshalPKIXPublicKey(pk)
		keyType = "PUBLIC KEY"
	case *ecdsa.PublicKey:
		keyBytes, _ = x509.MarshalPKIXPublicKey(pk)
		keyType = "PUBLIC KEY"
	case ed25519.PublicKey:
		keyBytes, _ = x509.MarshalPKIXPublicKey(pk)
		keyType = "PUBLIC KEY"
	default:
		return nil, errors.New("unsupported public key type")
	}

	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}

	return pem.EncodeToMemory(block), nil
}
