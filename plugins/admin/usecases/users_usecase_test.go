package usecases_test

import (
	"context"
	"testing"

	"github.com/uptrace/bun"

	coreinternalrepos "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	coreinternalservices "github.com/GoBetterAuth/go-better-auth/v2/internal/services"
	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	migrationsmodule "github.com/GoBetterAuth/go-better-auth/v2/migrations"
	adminplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin"
	admintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

type usersUsecaseLogger struct{}

func (l usersUsecaseLogger) Debug(msg string, args ...any) {}
func (l usersUsecaseLogger) Info(msg string, args ...any)  {}
func (l usersUsecaseLogger) Warn(msg string, args ...any)  {}
func (l usersUsecaseLogger) Error(msg string, args ...any) {}

func newUsersUsecaseDB(t *testing.T) *bun.DB {
	t.Helper()
	ctx := context.Background()

	db := internaltests.NewSQLiteIntegrationDB(t)

	migrator, err := migrationsmodule.NewMigrator(db, usersUsecaseLogger{})
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	coreSet, err := migrationsmodule.CoreMigrationSet("sqlite")
	if err != nil {
		t.Fatalf("failed to load core migration set: %v", err)
	}

	plugin := adminplugin.New(admintypes.AdminPluginConfig{})
	adminSet := migrationsmodule.MigrationSet{
		PluginID:   plugin.Metadata().ID,
		DependsOn:  plugin.DependsOn(),
		Migrations: plugin.Migrations("sqlite"),
	}

	if err := migrator.Migrate(ctx, []migrationsmodule.MigrationSet{coreSet, adminSet}); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func seedUsecaseUser(t *testing.T, db bun.IDB, id, name, email string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO users (id, name, email, email_verified) VALUES (?, ?, ?, ?)`, id, name, email, false)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func buildUsersUsecase(db bun.IDB) usecases.UsersUseCase {
	repo := coreinternalrepos.NewBunUserRepository(db)
	service := coreinternalservices.NewUserService(repo, nil)
	return usecases.NewUsersUseCase(service)
}

func TestUsersUseCase_GetAllUsersCursorPagination(t *testing.T) {
	db := newUsersUsecaseDB(t)
	uc := buildUsersUsecase(db)
	seedUsecaseUser(t, db, "u-001", "A", "a@example.com")
	seedUsecaseUser(t, db, "u-002", "B", "b@example.com")
	seedUsecaseUser(t, db, "u-003", "C", "c@example.com")

	first, err := uc.GetAll(context.Background(), nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(first.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(first.Users))
	}
	if first.NextCursor == nil || *first.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}

	second, err := uc.GetAll(context.Background(), first.NextCursor, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(second.Users) != 1 {
		t.Fatalf("expected 1 user on second page, got %d", len(second.Users))
	}
}

func TestUsersUseCase_GetUpdateDeleteUser(t *testing.T) {
	db := newUsersUsecaseDB(t)
	uc := buildUsersUsecase(db)
	seedUsecaseUser(t, db, "u-101", "Original", "original@example.com")

	user, err := uc.GetByID(context.Background(), "u-101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || user.ID != "u-101" {
		t.Fatalf("expected user u-101")
	}

	newName := "Updated"
	updated, err := uc.Update(context.Background(), "u-101", admintypes.UpdateUserRequest{Name: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("expected updated name %s, got %s", newName, updated.Name)
	}

	if err := uc.Delete(context.Background(), "u-101"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	deleted, err := uc.GetByID(context.Background(), "u-101")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected user to be deleted")
	}
}

func TestUsersUseCase_CreateUser(t *testing.T) {
	db := newUsersUsecaseDB(t)
	uc := buildUsersUsecase(db)

	created, err := uc.Create(context.Background(), admintypes.CreateUserRequest{
		Name:  "Created User",
		Email: "created@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if created == nil || created.Email != "created@example.com" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	_, err = uc.Create(context.Background(), admintypes.CreateUserRequest{
		Name:  "Created User",
		Email: "created@example.com",
	})
	if err == nil {
		t.Fatal("expected duplicate create to fail")
	}
}
