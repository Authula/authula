package plugins_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/internal/plugins"
)

type testService struct {
	Name string
}

func TestNewServiceRegistry(t *testing.T) {
	t.Parallel()

	registry := plugins.NewServiceRegistry()

	require.NotNil(t, registry)
	assert.Nil(t, registry.Get("missing"))
}

func TestDefaultServiceRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		service any
	}{
		{name: "string service", service: "plain-value"},
		{name: "struct pointer service", service: &testService{Name: "svc"}},
		{name: "nil service", service: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := plugins.NewServiceRegistry()

			registry.Register("service", tt.service)

			assert.Equal(t, tt.service, registry.Get("service"))
		})
	}
}

func TestDefaultServiceRegistry_RegisterOverwrites(t *testing.T) {
	t.Parallel()

	registry := plugins.NewServiceRegistry()
	first := &testService{Name: "first"}
	second := &testService{Name: "second"}

	registry.Register("service", first)
	registry.Register("service", second)

	assert.Same(t, second, registry.Get("service"))
}

func TestDefaultServiceRegistry_GetMissingService(t *testing.T) {
	t.Parallel()

	registry := plugins.NewServiceRegistry()
	registry.Register("service", &testService{Name: "svc"})

	assert.Nil(t, registry.Get("other"))
}

func TestDefaultServiceRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const services = 100

	registry := plugins.NewServiceRegistry()
	registry.Register("shared", &testService{Name: "shared"})

	var wg sync.WaitGroup
	wg.Add(services * 2)
	for i := range services {
		go func() {
			defer wg.Done()
			registry.Register(fmt.Sprintf("service-%d", i), &testService{Name: fmt.Sprintf("service-%d", i)})
		}()
		go func() {
			defer wg.Done()
			// Reads race with writes on purpose; the registry must stay consistent.
			registry.Get("shared")
		}()
	}
	wg.Wait()

	for i := range services {
		service, ok := registry.Get(fmt.Sprintf("service-%d", i)).(*testService)
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("service-%d", i), service.Name)
	}
	shared, ok := registry.Get("shared").(*testService)
	require.True(t, ok)
	assert.Equal(t, "shared", shared.Name)
}
