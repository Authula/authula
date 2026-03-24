package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOrganizationsPluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	defaultConfig := &OrganizationsPluginConfig{}
	defaultConfig.ApplyDefaults()
	require.NotNil(t, defaultConfig.MembersLimit)
	require.Equal(t, 100, *defaultConfig.MembersLimit)
	require.NotNil(t, defaultConfig.InvitationsLimit)
	require.Equal(t, 100, *defaultConfig.InvitationsLimit)
	require.Equal(t, 24*time.Hour, defaultConfig.InvitationExpiresIn)

	zeroLimit := 0
	explicitConfig := &OrganizationsPluginConfig{MembersLimit: &zeroLimit}
	explicitConfig.ApplyDefaults()
	require.NotNil(t, explicitConfig.MembersLimit)
	require.Equal(t, 0, *explicitConfig.MembersLimit)
}
