package repositories_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/plugins/admin/repositories"
)

func TestNewAdminRepositories_WiresRepositories(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	repos := repositories.NewAdminRepositories(db)
	require.NotNil(t, repos)

	assert.NotNil(t, repos.UserStateRepository())
	assert.NotNil(t, repos.SessionStateRepository())
	assert.NotNil(t, repos.ImpersonationRepository())

	_, ok := repos.UserStateRepository().(*repositories.BunUserStateRepository)
	assert.True(t, ok)
	_, ok = repos.SessionStateRepository().(*repositories.BunSessionStateRepository)
	assert.True(t, ok)
	_, ok = repos.ImpersonationRepository().(*repositories.BunImpersonationRepository)
	assert.True(t, ok)
}
