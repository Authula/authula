package models

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubServiceRegistry struct {
	services map[string]any
}

func (r *stubServiceRegistry) Register(name string, service any) {
	if r.services == nil {
		r.services = make(map[string]any)
	}
	r.services[name] = service
}

func (r *stubServiceRegistry) Get(name string) any {
	if r == nil {
		return nil
	}
	return r.services[name]
}

type stubRegisteredService struct {
	Name string
}

func TestServiceRegistryContextRoundTrip(t *testing.T) {
	t.Parallel()

	registry := &stubServiceRegistry{}
	ctx := NewContextWithServiceRegistry(context.Background(), registry)

	require.Same(t, registry, GetServiceRegistry(ctx))
}

func TestGetServiceRegistryMissingValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "empty context", ctx: context.Background()},
		{
			name: "value of the wrong type",
			ctx:  context.WithValue(context.Background(), ContextServiceRegistry, "not-a-registry"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Nil(t, GetServiceRegistry(tt.ctx))
		})
	}
}

func TestGetServiceFromContext(t *testing.T) {
	t.Parallel()

	service := &stubRegisteredService{Name: "users"}

	tests := []struct {
		name      string
		ctx       context.Context
		wantOK    bool
		wantValue *stubRegisteredService
	}{
		{
			name:   "returns false without a registry",
			ctx:    context.Background(),
			wantOK: false,
		},
		{
			name: "returns the registered service",
			ctx: NewContextWithServiceRegistry(context.Background(), &stubServiceRegistry{
				services: map[string]any{ServiceUser.String(): service},
			}),
			wantOK:    true,
			wantValue: service,
		},
		{
			name: "returns false for an unknown service",
			ctx: NewContextWithServiceRegistry(context.Background(), &stubServiceRegistry{
				services: map[string]any{},
			}),
			wantOK: false,
		},
		{
			name: "returns false when the service has the wrong type",
			ctx: NewContextWithServiceRegistry(context.Background(), &stubServiceRegistry{
				services: map[string]any{ServiceUser.String(): "not-a-service"},
			}),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ok, got := GetServiceFromContext[*stubRegisteredService](tt.ctx, ServiceUser)

			require.Equal(t, tt.wantOK, ok)
			require.Same(t, tt.wantValue, got)
		})
	}
}

func TestServiceRegistryRegisterAndGet(t *testing.T) {
	t.Parallel()

	registry := &stubServiceRegistry{}
	service := &stubRegisteredService{Name: "sessions"}

	registry.Register(ServiceSession.String(), service)

	require.Same(t, service, registry.Get(ServiceSession.String()))
}
