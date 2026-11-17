package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/organizations/repositories"
	plugintests "github.com/Authula/authula/plugins/organizations/tests"
)

func TestBunOrganizationMemberRepository_WithUserLookups(t *testing.T) {
	t.Parallel()

	t.Run("get by id hydrates the user", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "admin")

		repo := repositories.NewBunOrganizationMemberRepository(db)
		member, err := repo.GetByIDWithUser(context.Background(), "mem-1")
		require.NoError(t, err)
		require.NotNil(t, member)
		require.Equal(t, "mem-1", member.ID)
		require.Equal(t, "org-1", member.OrganizationID)
		require.Equal(t, "admin", member.Role)
		require.Equal(t, "user-2", member.User.ID)
		require.Equal(t, "Test User", member.User.Name)
		require.Equal(t, "user-2@example.com", member.User.Email)
		require.True(t, member.User.EmailVerified)
	})

	t.Run("get by id returns nil when the member is missing", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		repo := repositories.NewBunOrganizationMemberRepository(db)

		member, err := repo.GetByIDWithUser(context.Background(), "missing")
		require.NoError(t, err)
		require.Nil(t, member)
	})

	t.Run("get by id returns database errors", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		require.NoError(t, db.Close())

		repo := repositories.NewBunOrganizationMemberRepository(db)
		member, err := repo.GetByIDWithUser(context.Background(), "mem-1")
		require.Error(t, err)
		require.Nil(t, member)
	})

	t.Run("get by organization and user hydrates the user", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")

		repo := repositories.NewBunOrganizationMemberRepository(db)
		member, err := repo.GetByOrganizationIDAndUserIDWithUser(context.Background(), "org-1", "user-2")
		require.NoError(t, err)
		require.NotNil(t, member)
		require.Equal(t, "mem-1", member.ID)
		require.Equal(t, "user-2", member.User.ID)
		require.Equal(t, "user-2@example.com", member.User.Email)
	})

	t.Run("get by organization and user returns nil when the member is missing", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		repo := repositories.NewBunOrganizationMemberRepository(db)

		member, err := repo.GetByOrganizationIDAndUserIDWithUser(context.Background(), "org-1", "user-2")
		require.NoError(t, err)
		require.Nil(t, member)
	})

	t.Run("get by organization and user returns database errors", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		require.NoError(t, db.Close())

		repo := repositories.NewBunOrganizationMemberRepository(db)
		member, err := repo.GetByOrganizationIDAndUserIDWithUser(context.Background(), "org-1", "user-2")
		require.Error(t, err)
		require.Nil(t, member)
	})
}

func TestBunOrganizationTeamMemberRepository_GetByIDWithMemberAndUser(t *testing.T) {
	t.Parallel()

	t.Run("hydrates the member and user", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
		plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
		plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
		plugintests.SeedOrganizationTeamMember(t, db, "tm-1", "team-1", "mem-1")

		repo := repositories.NewBunOrganizationTeamMemberRepository(db)
		teamMember, err := repo.GetByIDWithMemberAndUser(context.Background(), "tm-1")
		require.NoError(t, err)
		require.NotNil(t, teamMember)
		require.Equal(t, "tm-1", teamMember.ID)
		require.Equal(t, "team-1", teamMember.TeamID)
		require.Equal(t, "mem-1", teamMember.Member.ID)
		require.Equal(t, "org-1", teamMember.Member.OrganizationID)
		require.Equal(t, "member", teamMember.Member.Role)
		require.Equal(t, "user-2", teamMember.Member.User.ID)
		require.Equal(t, "user-2@example.com", teamMember.Member.User.Email)
	})

	t.Run("returns nil when the team member is missing", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		repo := repositories.NewBunOrganizationTeamMemberRepository(db)

		teamMember, err := repo.GetByIDWithMemberAndUser(context.Background(), "missing")
		require.NoError(t, err)
		require.Nil(t, teamMember)
	})

	t.Run("returns database errors", func(t *testing.T) {
		t.Parallel()

		db := plugintests.SetupRepoDB(t)
		require.NoError(t, db.Close())

		repo := repositories.NewBunOrganizationTeamMemberRepository(db)
		teamMember, err := repo.GetByIDWithMemberAndUser(context.Background(), "tm-1")
		require.Error(t, err)
		require.Nil(t, teamMember)
	})
}
