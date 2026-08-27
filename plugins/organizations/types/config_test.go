package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/plugins/organizations/types"
)

func TestOrganizationsPluginConfig_ApplyDefaultsMaxPageLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		limit    *int
		expected int
	}{
		{name: "an unset maximum falls back to the default", limit: nil, expected: pagination.DefaultMaxLimit},
		{name: "a zero maximum falls back to the default", limit: new(0), expected: pagination.DefaultMaxLimit},
		{name: "a negative maximum falls back to the default", limit: new(-10), expected: pagination.DefaultMaxLimit},
		{name: "a configured maximum is left alone", limit: new(250), expected: 250},
		{name: "a maximum below the default page size is left alone", limit: new(5), expected: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := &types.OrganizationsPluginConfig{MaxPageLimit: tt.limit}
			config.ApplyDefaults()

			require.NotNil(t, config.MaxPageLimit)
			require.Equal(t, tt.expected, *config.MaxPageLimit)
		})
	}
}

func TestOrganizationsPluginConfig_ApplyDefaultsLeavesOtherDefaultsIntact(t *testing.T) {
	t.Parallel()

	config := &types.OrganizationsPluginConfig{}
	config.ApplyDefaults()

	require.Equal(t, 100, *config.MembersLimit)
	require.Equal(t, 100, *config.InvitationsLimit)
	require.Equal(t, pagination.DefaultMaxLimit, *config.MaxPageLimit)
	require.Equal(t, 24*time.Hour, config.InvitationExpiresIn)
}
