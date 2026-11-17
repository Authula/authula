package models

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubPluginRegistry struct {
	PluginRegistry
}

func TestPluginRegistryContextRoundTrip(t *testing.T) {
	t.Parallel()

	registry := &stubPluginRegistry{}
	ctx := NewContextWithPluginRegistry(context.Background(), registry)

	got := GetPluginRegistryFromContext(ctx)

	require.Same(t, registry, got)
}

func TestGetPluginRegistryFromContextMissingValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "empty context", ctx: context.Background()},
		{
			name: "value of the wrong type",
			ctx:  context.WithValue(context.Background(), ContextPluginRegistry, "not-a-registry"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Nil(t, GetPluginRegistryFromContext(tt.ctx))
		})
	}
}
