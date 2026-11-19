package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/services"
)

type fakeJWTService struct{}

func (fakeJWTService) ValidateToken(context.Context, string) (*models.Actor, error) {
	return nil, nil
}

func (fakeJWTService) GetActiveKey(context.Context, string, string) (*models.SigningKey, error) {
	return nil, models.ErrSigningKeyNotFound
}

func (fakeJWTService) GetKeyByID(context.Context, string) (*models.SigningKey, error) {
	return nil, models.ErrSigningKeyNotFound
}

func (fakeJWTService) GetPublicKeys(context.Context, string) ([]models.SigningKey, error) {
	return nil, nil
}

func (fakeJWTService) GenerateKey(context.Context, string, string) error { return nil }
func (fakeJWTService) RotateKey(context.Context, string, string) error   { return nil }
func (fakeJWTService) RotateKeyWithGrace(context.Context, string, string, time.Duration) error {
	return nil
}

var _ services.JWTService = fakeJWTService{}

type fakeServiceRegistry struct {
	services map[string]any
}

func (r *fakeServiceRegistry) Register(name string, service any) {
	if r.services == nil {
		r.services = make(map[string]any)
	}
	r.services[name] = service
}

func (r *fakeServiceRegistry) Get(name string) any {
	return r.services[name]
}

func TestJWTServiceContractResolvesFromRegistry(t *testing.T) {
	t.Parallel()

	registry := new(fakeServiceRegistry)
	registry.Register(models.ServiceJWT.String(), fakeJWTService{})

	resolved, ok := registry.Get(models.ServiceJWT.String()).(services.JWTService)

	require.True(t, ok)
	require.NotNil(t, resolved)
}
