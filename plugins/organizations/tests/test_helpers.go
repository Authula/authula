package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	orgpluginmigrations "github.com/Authula/authula/plugins/organizations/migrationset"
)

func SetupRepoDB(t *testing.T) *bun.DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db := bun.NewDB(sqlDB, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	migrator, err := migrations.NewMigrator(db, &internaltests.MockLogger{})
	require.NoError(t, err)

	coreSet, err := migrations.CoreMigrationSet("sqlite")
	require.NoError(t, err)

	orgSet := migrations.MigrationSet{
		PluginID:   models.PluginOrganizations.String(),
		DependsOn:  []string{migrations.CorePluginID},
		Migrations: orgpluginmigrations.ForProvider("sqlite"),
	}

	err = migrator.Migrate(ctx, []migrations.MigrationSet{coreSet, orgSet})
	require.NoError(t, err)

	SeedUser(t, db, "user-1")
	SeedUser(t, db, "user-2")

	return db
}

func SeedUser(t *testing.T, db bun.IDB, userID string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `INSERT INTO users (id, name, email, email_verified, metadata) VALUES (?, ?, ?, ?, ?)`, userID, "Test User", userID+"@example.com", true, `{}`)
	require.NoError(t, err)
}

func SeedOrganization(t *testing.T, db bun.IDB, organizationID, ownerID, name, slug string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `INSERT INTO organizations (id, owner_id, name, slug, metadata) VALUES (?, ?, ?, ?, ?)`, organizationID, ownerID, name, slug, `{}`)
	require.NoError(t, err)
}

func SeedOrganizationMember(t *testing.T, db bun.IDB, memberID, organizationID, userID, role string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `INSERT INTO organization_members (id, organization_id, user_id, role) VALUES (?, ?, ?, ?)`, memberID, organizationID, userID, role)
	require.NoError(t, err)
}

func SeedOrganizationTeam(t *testing.T, db bun.IDB, teamID, organizationID, name, slug string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `INSERT INTO organization_teams (id, organization_id, name, slug, metadata) VALUES (?, ?, ?, ?, ?)`, teamID, organizationID, name, slug, `{}`)
	require.NoError(t, err)
}

func SeedOrganizationInvitation(t *testing.T, db bun.IDB, invitationID, email, inviterID, organizationID, role, status string, expiresAt time.Time) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `INSERT INTO organization_invitations (id, email, inviter_id, organization_id, role, status, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, invitationID, email, inviterID, organizationID, role, status, expiresAt.UTC())
	require.NoError(t, err)
}

func SeedOrganizationTeamMember(t *testing.T, db bun.IDB, id, teamID, memberID string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `INSERT INTO organization_team_members (id, team_id, member_id) VALUES (?, ?, ?)`, id, teamID, memberID)
	require.NoError(t, err)
}
