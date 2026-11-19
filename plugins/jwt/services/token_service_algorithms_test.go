package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
)

func algorithmTestKey(t *testing.T, algorithm string) *models.SigningKey {
	t.Helper()

	var privateKey any
	switch algorithm {
	case models.SigningKeyAlgorithmRS256, models.SigningKeyAlgorithmPS256:
		key, err := rsa.GenerateKey(rand.Reader, models.DefaultRSAKeySize)
		require.NoError(t, err)
		privateKey = key
	case models.SigningKeyAlgorithmES256:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		privateKey = key
	default:
		t.Fatalf("unsupported test algorithm %s", algorithm)
	}

	privateBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	var publicKey any
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		publicKey = &key.PublicKey
	case *ecdsa.PrivateKey:
		publicKey = &key.PublicKey
	}
	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	require.NoError(t, err)

	return &models.SigningKey{
		ID:         "id-" + algorithm,
		KeyID:      "kid-" + algorithm,
		Algorithm:  algorithm,
		KeyPurpose: models.SigningKeyPurposeJWTPlugin,
		PublicKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes})),
		IsActive:   true,
	}
}

func TestTokenService_SignsWithConfiguredAlgorithm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		algorithm string
		jwa       jwa.SignatureAlgorithm
	}{
		{algorithm: models.SigningKeyAlgorithmRS256, jwa: jwa.RS256()},
		{algorithm: models.SigningKeyAlgorithmES256, jwa: jwa.ES256()},
		{algorithm: models.SigningKeyAlgorithmPS256, jwa: jwa.PS256()},
	}

	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			t.Parallel()

			key := algorithmTestKey(t, tt.algorithm)
			keySvc := new(jwttests.MockKeyService)
			keySvc.On("GetActiveKey", mock.Anything).Return(key, nil)
			coreTokenSvc := new(jwttests.MockTokenServiceCore)
			coreTokenSvc.On("Decrypt", key.PrivateKey).Return(key.PrivateKey, nil)

			svc := &tokenService{
				logger:            &internaltests.MockLogger{},
				coreTokenService:  coreTokenSvc,
				keyService:        keySvc,
				signingKeyService: new(jwttests.MockSigningKeyService),
				expiresIn:         15 * time.Minute,
				refreshExpiresIn:  7 * 24 * time.Hour,
				issuer:            testIssuer,
				audience:          testAudience,
			}

			pair, err := svc.GenerateUserToken(context.Background(), "user-1", "session-1", nil)
			require.NoError(t, err)

			publicKey, err := jwk.ParseKey([]byte(key.PublicKey), jwk.WithPEM(true))
			require.NoError(t, err)
			require.NoError(t, publicKey.Set(jwk.KeyIDKey, key.KeyID))

			parsed, err := jwt.Parse([]byte(pair.AccessToken), jwt.WithKey(tt.jwa, publicKey), jwt.WithValidate(false))
			require.NoError(t, err)
			assertSubject, _ := parsed.Subject()
			require.Equal(t, "user-1", assertSubject)

			keySvc.AssertExpectations(t)
			coreTokenSvc.AssertExpectations(t)
		})
	}
}
