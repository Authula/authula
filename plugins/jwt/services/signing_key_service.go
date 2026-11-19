package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/jwt/repositories"
)

const (
	defaultGracePeriod = time.Hour
	defaultClockSkew   = 5 * time.Minute
)

// Encryptor encrypts private key material at rest. The core token service
// implements this interface.
type Encryptor interface {
	Encrypt(value string) (string, error)
}

// SigningKeyConfig controls rotation behaviour. MaxTokenLifetime is optional
// and, when set, lets the service keep old public keys long enough for the
// longest-lived token issued by any client instead of relying only on the
// global grace period.
type SigningKeyConfig struct {
	GracePeriod      time.Duration
	ClockSkew        time.Duration
	MaxTokenLifetime func(ctx context.Context) (time.Duration, error)
}

type signingKeyService struct {
	repo      repositories.SigningKeyRepository
	encryptor Encryptor
	logger    models.Logger
	config    SigningKeyConfig
}

var _ SigningKeyService = (*signingKeyService)(nil)

// NewSigningKeyService creates the JWT plugin's signing key service.
func NewSigningKeyService(repo repositories.SigningKeyRepository, encryptor Encryptor, logger models.Logger, config SigningKeyConfig) SigningKeyService {
	if config.GracePeriod <= 0 {
		config.GracePeriod = defaultGracePeriod
	}
	if config.ClockSkew <= 0 {
		config.ClockSkew = defaultClockSkew
	}
	return &signingKeyService{
		repo:      repo,
		encryptor: encryptor,
		logger:    logger,
		config:    config,
	}
}

func (s *signingKeyService) GetActiveKey(ctx context.Context, purpose, algorithm string) (*models.SigningKey, error) {
	if err := models.ValidateSigningKeyPurpose(purpose); err != nil {
		return nil, err
	}
	if err := models.ValidateSigningKeyAlgorithm(algorithm); err != nil {
		return nil, err
	}
	return s.repo.GetActiveKey(ctx, purpose, algorithm)
}

func (s *signingKeyService) GetKeyByID(ctx context.Context, kid string) (*models.SigningKey, error) {
	if kid == "" {
		return nil, fmt.Errorf("%w: empty key id", models.ErrSigningKeyNotFound)
	}
	return s.repo.GetKeyByID(ctx, kid)
}

func (s *signingKeyService) GetPublicKeys(ctx context.Context, purpose string) ([]models.SigningKey, error) {
	if err := models.ValidateSigningKeyPurpose(purpose); err != nil {
		return nil, err
	}
	keys, err := s.repo.GetPublicKeys(ctx, purpose, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	result := make([]models.SigningKey, 0, len(keys))
	for _, key := range keys {
		publicKey := *key
		publicKey.PrivateKey = ""
		result = append(result, publicKey)
	}
	return result, nil
}

// GenerateKey creates an active key only when no active key exists for the
// purpose and algorithm. If a key exists for another algorithm, it is retired
// with the normal rotation grace period and remains published until it expires.
func (s *signingKeyService) GenerateKey(ctx context.Context, purpose, algorithm string) error {
	if err := models.ValidateSigningKeyPurpose(purpose); err != nil {
		return err
	}
	if err := models.ValidateSigningKeyAlgorithm(algorithm); err != nil {
		return err
	}

	if _, err := s.repo.GetActiveKey(ctx, purpose, algorithm); err == nil {
		return nil
	} else if !errors.Is(err, models.ErrSigningKeyNotFound) {
		return fmt.Errorf("failed to inspect active signing key: %w", err)
	}

	key, err := s.buildKey(purpose, algorithm)
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(s.effectiveGracePeriod(ctx, s.config.GracePeriod))
	if err := s.repo.Rotate(ctx, purpose, key, expiresAt); err != nil {
		// A concurrent caller may have won the race. If a usable key now
		// exists, generation succeeded from the caller's perspective.
		if _, getErr := s.repo.GetActiveKey(ctx, purpose, algorithm); getErr == nil {
			return nil
		}
		return fmt.Errorf("failed to generate signing key: %w", err)
	}

	s.logger.Info("generated jwt signing key", "purpose", purpose, "algorithm", algorithm, "kid", key.KeyID)
	return nil
}

func (s *signingKeyService) RotateKey(ctx context.Context, purpose, algorithm string) error {
	return s.RotateKeyWithGrace(ctx, purpose, algorithm, s.config.GracePeriod)
}

func (s *signingKeyService) RotateKeyWithGrace(ctx context.Context, purpose, algorithm string, gracePeriod time.Duration) error {
	if err := models.ValidateSigningKeyPurpose(purpose); err != nil {
		return err
	}
	if err := models.ValidateSigningKeyAlgorithm(algorithm); err != nil {
		return err
	}

	key, err := s.buildKey(purpose, algorithm)
	if err != nil {
		return err
	}

	grace := s.effectiveGracePeriod(ctx, gracePeriod)
	expiresAt := time.Now().UTC().Add(grace)
	if err := s.repo.Rotate(ctx, purpose, key, expiresAt); err != nil {
		return fmt.Errorf("failed to rotate signing key: %w", err)
	}

	s.logger.Info("rotated jwt signing key", "purpose", purpose, "algorithm", algorithm, "kid", key.KeyID, "old_keys_expire_at", expiresAt)
	return nil
}

func (s *signingKeyService) buildKey(purpose, algorithm string) (*models.SigningKey, error) {
	privatePEM, publicPEM, err := generateKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	encrypted, err := s.encryptor.Encrypt(privatePEM)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	return &models.SigningKey{
		ID:         uuid.NewString(),
		KeyID:      uuid.NewString(),
		Algorithm:  algorithm,
		KeyPurpose: purpose,
		PublicKey:  publicPEM,
		PrivateKey: encrypted,
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// effectiveGracePeriod returns the longer of the requested grace period and the
// longest token lifetime known to the deployment, plus clock skew. This keeps
// old public keys available until every token signed with them has expired.
func (s *signingKeyService) effectiveGracePeriod(ctx context.Context, requested time.Duration) time.Duration {
	if requested <= 0 {
		requested = s.config.GracePeriod
	}
	if requested <= 0 {
		requested = defaultGracePeriod
	}

	if s.config.MaxTokenLifetime != nil {
		maxLifetime, err := s.config.MaxTokenLifetime(ctx)
		if err != nil {
			s.logger.Warn("failed to determine maximum token lifetime for key rotation", "error", err)
		} else if maxLifetime > requested {
			s.logger.Warn("extending key rotation grace period to cover token lifetime",
				"configured_grace_period", requested,
				"max_token_lifetime", maxLifetime,
			)
			requested = maxLifetime
		}
	}

	return requested + s.config.ClockSkew
}

// generateKeyPair returns a PEM-encoded PKCS#8 private key and a PEM-encoded
// PKIX public key for the requested algorithm.
func generateKeyPair(algorithm string) (privatePEM string, publicPEM string, err error) {
	if err := models.ValidateSigningKeyAlgorithm(algorithm); err != nil {
		return "", "", err
	}

	var privateKey any
	switch algorithm {
	case models.SigningKeyAlgorithmRS256, models.SigningKeyAlgorithmPS256:
		privateKey, err = rsa.GenerateKey(rand.Reader, models.DefaultRSAKeySize)
	case models.SigningKeyAlgorithmES256:
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	default:
		return "", "", fmt.Errorf("%w: %q", models.ErrUnsupportedSigningAlgorithm, algorithm)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to generate %s key: %w", algorithm, err)
	}

	privateBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	var publicKey any
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		publicKey = &key.PublicKey
	case *ecdsa.PrivateKey:
		publicKey = &key.PublicKey
	default:
		return "", "", fmt.Errorf("%w: unsupported key type %T", models.ErrUnsupportedSigningAlgorithm, privateKey)
	}

	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	privatePEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes}))
	publicPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
	return privatePEM, publicPEM, nil
}
