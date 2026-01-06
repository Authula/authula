package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"

	"github.com/GoBetterAuth/go-better-auth/models"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/repositories"
	"github.com/GoBetterAuth/go-better-auth/plugins/jwt/types"
)

type keyService struct {
	repo      repositories.JWKSRepository
	logger    models.Logger
	secret    string
	algorithm string
}

// NewKeyService creates a new key service
func NewKeyService(repo repositories.JWKSRepository, logger models.Logger, secret, algorithm string) KeyService {
	return &keyService{
		repo:      repo,
		logger:    logger,
		secret:    secret,
		algorithm: strings.ToLower(algorithm),
	}
}

// GenerateKeysIfMissing generates the initial key pair if none exist
func (s *keyService) GenerateKeysIfMissing(ctx context.Context) error {
	keys, err := s.repo.GetJWKSKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) > 0 {
		s.logger.Debug("keys already exist, skipping generation")
		return nil
	}

	s.logger.Info("generating initial key pair", "algorithm", s.algorithm)
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
		s.logger.Debug("failed to check key rotation", "error", err)
		return false
	}

	return time.Since(key.CreatedAt) > rotationInterval
}

// RotateKeysIfNeeded rotates keys if they're past the rotation interval
func (s *keyService) RotateKeysIfNeeded(ctx context.Context, rotationInterval time.Duration, invalidateCacheFunc func(context.Context) error) (bool, error) {
	if !s.IsKeyRotationDue(ctx, rotationInterval) {
		return false, nil
	}

	s.logger.Info("rotating keys due to age", "algorithm", s.algorithm)

	// Mark old keys as expired
	keys, err := s.repo.GetJWKSKeys(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get keys for rotation: %w", err)
	}

	now := time.Now()
	for _, key := range keys {
		if err := s.repo.MarkKeyExpired(ctx, key.ID, now); err != nil {
			return false, fmt.Errorf("failed to expire old key: %w", err)
		}
	}

	if err := s.generateAndStoreKey(ctx); err != nil {
		return false, fmt.Errorf("failed to generate new key: %w", err)
	}

	if invalidateCacheFunc != nil {
		if err := invalidateCacheFunc(ctx); err != nil {
			s.logger.Warn("failed to invalidate cache after key rotation", "error", err)
			// Don't fail rotation; cache will be refreshed on next access
		}
	}

	return true, nil
}

// generateAndStoreKey generates a key pair and stores it in the database
func (s *keyService) generateAndStoreKey(ctx context.Context) error {
	var pubKey, privKey any
	var err error

	switch s.algorithm {
	case "rsa2048", "rs2048":
		privKey, err = rsa.GenerateKey(rand.Reader, 2048)
	case "rsa4096", "rs4096":
		privKey, err = rsa.GenerateKey(rand.Reader, 4096)
	case "rsa256", "rs256":
		privKey, err = rsa.GenerateKey(rand.Reader, 2048) // RS256 uses 2048 minimum
	case "es256":
		privKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "es384":
		privKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "es512":
		privKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		// Default to Ed25519
		var seed [32]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return fmt.Errorf("failed to read random seed: %w", err)
		}
		privKey = ed25519.NewKeyFromSeed(seed[:])
	}

	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	switch pk := privKey.(type) {
	case *rsa.PrivateKey:
		pubKey = &pk.PublicKey
	case *ecdsa.PrivateKey:
		pubKey = &pk.PublicKey
	case ed25519.PrivateKey:
		pubKey = pk.Public()
	default:
		return errors.New("unsupported key type")
	}

	privKeyPEM, err := privateKeyToPEM(privKey)
	if err != nil {
		return fmt.Errorf("failed to convert private key to PEM: %w", err)
	}

	pubKeyPEM, err := publicKeyToPEM(pubKey)
	if err != nil {
		return fmt.Errorf("failed to convert public key to PEM: %w", err)
	}

	encryptedPrivKey, err := s.encryptPrivateKey(privKeyPEM)
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

	s.logger.Info("generated and stored key", "id", jwksKey.ID, "algorithm", s.algorithm)
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

// encryptPrivateKey encrypts a private key using PBKDF2 + AES-256-GCM
// Returns base64-encoded [salt_16][nonce_12][ciphertext][tag_16]
func (s *keyService) encryptPrivateKey(keyData []byte) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(s.secret), salt, 100000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate random nonce
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, keyData, nil)

	// Combine: [salt][nonce][ciphertext]
	result := append(salt, nonce...)
	result = append(result, ciphertext...)

	return base64.StdEncoding.EncodeToString(result), nil
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
