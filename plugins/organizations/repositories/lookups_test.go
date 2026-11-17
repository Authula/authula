package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/repositories"
	plugintests "github.com/Authula/authula/plugins/organizations/tests"
)

func TestBunOrganizationMemberRepository_WithUserLookups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationMemberRepository, context.Context)
	}{
		{
			name: "get by id hydrates the user",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "admin")
			},
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				member, err := repo.GetByIDWithUser(ctx, "mem-1")
				require.NoError(t, err)
				require.NotNil(t, member)
				require.Equal(t, "mem-1", member.ID)
				require.Equal(t, "org-1", member.OrganizationID)
				require.Equal(t, "admin", member.Role)
				require.Equal(t, "user-2", member.User.ID)
				require.Equal(t, "Test User", member.User.Name)
				require.Equal(t, "user-2@example.com", member.User.Email)
				require.True(t, member.User.EmailVerified)
			},
		},
		{
			name: "get by id returns nil when the member is missing",
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				member, err := repo.GetByIDWithUser(ctx, "missing")
				require.NoError(t, err)
				require.Nil(t, member)
			},
		},
		{
			name:    "get by id returns database errors",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				member, err := repo.GetByIDWithUser(ctx, "mem-1")
				require.Error(t, err)
				require.Nil(t, member)
			},
		},
		{
			name: "get by organization and user hydrates the user",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
			},
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				member, err := repo.GetByOrganizationIDAndUserIDWithUser(ctx, "org-1", "user-2")
				require.NoError(t, err)
				require.NotNil(t, member)
				require.Equal(t, "mem-1", member.ID)
				require.Equal(t, "user-2", member.User.ID)
				require.Equal(t, "user-2@example.com", member.User.Email)
			},
		},
		{
			name: "get by organization and user returns nil when the member is missing",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
			},
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				member, err := repo.GetByOrganizationIDAndUserIDWithUser(ctx, "org-1", "user-2")
				require.NoError(t, err)
				require.Nil(t, member)
			},
		},
		{
			name:    "get by organization and user returns database errors",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationMemberRepository, ctx context.Context) {
				t.Helper()
				member, err := repo.GetByOrganizationIDAndUserIDWithUser(ctx, "org-1", "user-2")
				require.Error(t, err)
				require.Nil(t, member)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationMemberRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}

func TestBunOrganizationTeamMemberRepository_GetByIDWithMemberAndUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seed    func(t *testing.T, db *bun.DB)
		closeDB bool
		run     func(*testing.T, repositories.OrganizationTeamMemberRepository, context.Context)
	}{
		{
			name: "hydrates the member and user",
			seed: func(t *testing.T, db *bun.DB) {
				t.Helper()
				plugintests.SeedOrganization(t, db, "org-1", "user-1", "Acme Inc", "acme-inc")
				plugintests.SeedOrganizationMember(t, db, "mem-1", "org-1", "user-2", "member")
				plugintests.SeedOrganizationTeam(t, db, "team-1", "org-1", "Platform", "platform")
				plugintests.SeedOrganizationTeamMember(t, db, "tm-1", "team-1", "mem-1")
			},
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				teamMember, err := repo.GetByIDWithMemberAndUser(ctx, "tm-1")
				require.NoError(t, err)
				require.NotNil(t, teamMember)
				require.Equal(t, "tm-1", teamMember.ID)
				require.Equal(t, "team-1", teamMember.TeamID)
				require.Equal(t, "mem-1", teamMember.Member.ID)
				require.Equal(t, "org-1", teamMember.Member.OrganizationID)
				require.Equal(t, "member", teamMember.Member.Role)
				require.Equal(t, "user-2", teamMember.Member.User.ID)
				require.Equal(t, "user-2@example.com", teamMember.Member.User.Email)
			},
		},
		{
			name: "returns nil when the team member is missing",
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				teamMember, err := repo.GetByIDWithMemberAndUser(ctx, "missing")
				require.NoError(t, err)
				require.Nil(t, teamMember)
			},
		},
		{
			name:    "returns database errors",
			closeDB: true,
			run: func(t *testing.T, repo repositories.OrganizationTeamMemberRepository, ctx context.Context) {
				t.Helper()
				teamMember, err := repo.GetByIDWithMemberAndUser(ctx, "tm-1")
				require.Error(t, err)
				require.Nil(t, teamMember)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := plugintests.SetupRepoDB(t)
			if tt.seed != nil {
				tt.seed(t, db)
			}
			if tt.closeDB {
				require.NoError(t, db.Close())
			}

			repo := repositories.NewBunOrganizationTeamMemberRepository(db)
			tt.run(t, repo, context.Background())
		})
	}
}
