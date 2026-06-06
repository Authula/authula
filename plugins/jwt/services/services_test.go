package services

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/mock"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	jwttests "github.com/Authula/authula/plugins/jwt/tests"
	"github.com/Authula/authula/plugins/jwt/types"
)

type serviceTestFixture struct {
	logger           models.Logger
	sessionSvc       *jwttests.MockSessionService
	coreTokenSvc     *jwttests.MockTokenServiceCore
	keySvc           *jwttests.MockKeyService
	cacheSvc         *jwttests.MockCacheService
	blacklistSvc     *jwttests.MockBlacklistService
	refreshTokenRepo *jwttests.MockRefreshTokenRepository
	activeKey        *types.JWKS
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
}

func newServiceTestFixture(t *testing.T) *serviceTestFixture {
	t.Helper()

	pubPEM, privPEM, err := generateEd25519KeyPair()
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	activeKey := &types.JWKS{
		ID:         uuid.New().String(),
		PublicKey:  string(pubPEM),
		PrivateKey: string(privPEM),
		CreatedAt:  time.Now(),
	}

	return &serviceTestFixture{
		logger:           &internaltests.MockLogger{},
		sessionSvc:       &jwttests.MockSessionService{},
		coreTokenSvc:     &jwttests.MockTokenServiceCore{},
		keySvc:           &jwttests.MockKeyService{},
		cacheSvc:         &jwttests.MockCacheService{},
		blacklistSvc:     &jwttests.MockBlacklistService{},
		refreshTokenRepo: &jwttests.MockRefreshTokenRepository{},
		activeKey:        activeKey,
		expiresIn:        15 * time.Minute,
		refreshExpiresIn: 7 * 24 * time.Hour,
	}
}

func (f *serviceTestFixture) newJWTService() *tokenService {
	return &tokenService{
		logger:           f.logger,
		coreTokenService: f.coreTokenSvc,
		keyService:       f.keySvc,
		cacheService:     f.cacheSvc,
		blacklistService: f.blacklistSvc,
		sessionService:   f.sessionSvc,
		expiresIn:        f.expiresIn,
		refreshExpiresIn: f.refreshExpiresIn,
	}
}

func (f *serviceTestFixture) signTestToken(t *testing.T, claims map[string]any) string {
	t.Helper()

	privKey, err := jwk.ParseKey([]byte(f.activeKey.PrivateKey), jwk.WithPEM(true))
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	if err := privKey.Set(jwk.KeyIDKey, f.activeKey.ID); err != nil {
		t.Fatalf("failed to set key ID: %v", err)
	}

	token := jwt.New()
	for k, v := range claims {
		if err := token.Set(k, v); err != nil {
			t.Fatalf("failed to set claim %s: %v", k, err)
		}
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.EdDSA(), privKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return string(signed)
}

func (f *serviceTestFixture) setupKeyServiceMock() {
	f.keySvc.On("GetActiveKey", mock.Anything).Return(f.activeKey, nil).Maybe()

	pubKey, err := jwk.ParseKey([]byte(f.activeKey.PublicKey), jwk.WithPEM(true))
	if err != nil {
		panic(err)
	}

	if err := pubKey.Set(jwk.KeyIDKey, f.activeKey.ID); err != nil {
		panic(err)
	}
	if err := pubKey.Set(jwk.AlgorithmKey, "EdDSA"); err != nil {
		panic(err)
	}

	set := jwk.NewSet()
	if err := set.AddKey(pubKey); err != nil {
		panic(err)
	}

	data, err := json.Marshal(set)
	if err != nil {
		panic(err)
	}
	parsedSet, err := jwk.Parse(data)
	if err != nil {
		panic(err)
	}

	f.cacheSvc.On("GetJWKSWithFallback", mock.Anything).Return(parsedSet, nil).Maybe()
}

func generateEd25519KeyPair() (pubPEM, privPEM []byte, err error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	privPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})

	pubBytes, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		return nil, nil, err
	}

	pubPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	return pubPEM, privPEM, nil
}
