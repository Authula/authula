package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/uptrace/bun"

	coreinternalrepos "github.com/GoBetterAuth/go-better-auth/v2/internal/repositories"
	coreinternalservices "github.com/GoBetterAuth/go-better-auth/v2/internal/services"
	internaltests "github.com/GoBetterAuth/go-better-auth/v2/internal/tests"
	migrationsmodule "github.com/GoBetterAuth/go-better-auth/v2/migrations"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/repositories"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/services"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/usecases"
)

func newTestAPI(db bun.IDB) *API {
	repos := repositories.NewAdminRepositories(db)
	adminServices := services.NewAdminServices(repos)
	coreUserRepo := coreinternalrepos.NewBunUserRepository(db)
	coreUserService := coreinternalservices.NewUserService(coreUserRepo, nil)
	useCases := usecases.NewAdminUseCases(adminServices, coreUserService, types.AdminPluginConfig{}, nil, nil, 0)
	return NewAPI(useCases)
}

type adminTestLogger struct{}

func (l adminTestLogger) Debug(msg string, args ...any) {}
func (l adminTestLogger) Info(msg string, args ...any)  {}
func (l adminTestLogger) Warn(msg string, args ...any)  {}
func (l adminTestLogger) Error(msg string, args ...any) {}

func newAdminTestDB(t *testing.T) *bun.DB {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)

	ctx := context.Background()
	migrator, err := migrationsmodule.NewMigrator(db, adminTestLogger{})
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	coreSet, err := migrationsmodule.CoreMigrationSet("sqlite")
	if err != nil {
		t.Fatalf("failed to load core migration set: %v", err)
	}

	adminSet := migrationsmodule.MigrationSet{
		PluginID:   models.PluginAdmin.String(),
		DependsOn:  []string{migrationsmodule.CorePluginID},
		Migrations: []migrationsmodule.Migration{adminSQLiteInitial()},
	}

	if err := migrator.Migrate(ctx, []migrationsmodule.MigrationSet{coreSet, adminSet}); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func seedUser(t *testing.T, db bun.IDB, id, email string) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, email_verified) VALUES (?, ?, ?, ?)`, id, "Test User", email, false)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedSession(t *testing.T, db bun.IDB, id, userID string) {
	t.Helper()
	ctx := context.Background()
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	_, err := db.ExecContext(ctx, `INSERT INTO sessions (id, user_id, token, expires_at) VALUES (?, ?, ?, ?)`, id, userID, "token-"+id, expiresAt)
	if err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
}

func newReqCtx(method string, path string, metadata map[string]any, userID *string) *models.RequestContext {
	req := httptest.NewRequest(method, path, nil)
	return &models.RequestContext{
		Request: req,
		Route: &models.Route{
			Method:   method,
			Path:     path,
			Metadata: metadata,
		},
		Values: make(map[string]any),
		UserID: userID,
	}
}

func TestAdminRBACHook_Unauthenticated(t *testing.T) {
	db := newAdminTestDB(t)
	plugin := &AdminPlugin{Api: newTestAPI(db)}

	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/permissions", map[string]any{
		"permissions": []string{"admin.read"},
	}, nil)

	err := plugin.requireRBAC(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reqCtx.Handled {
		t.Fatalf("expected request to be handled")
	}
	if reqCtx.ResponseStatus != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, reqCtx.ResponseStatus)
	}
}

func TestAdminRBACHook_ForbiddenWithoutPermissions(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "user-no-perm", "noperm@example.com")

	plugin := &AdminPlugin{Api: newTestAPI(db)}

	uid := "user-no-perm"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/permissions", map[string]any{
		"permissions": []string{"admin.read"},
	}, &uid)

	err := plugin.requireRBAC(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reqCtx.Handled {
		t.Fatalf("expected request to be handled")
	}
	if reqCtx.ResponseStatus != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, reqCtx.ResponseStatus)
	}
}

func TestAdminRBACHook_AllowsWithRequiredPermission(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "admin-user", "admin@example.com")

	api := newTestAPI(db)
	plugin := &AdminPlugin{Api: api}

	ctx := context.Background()
	permission, err := api.CreatePermission(ctx, types.CreatePermissionRequest{Key: "admin.read"})
	if err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	role, err := api.CreateRole(ctx, types.CreateRoleRequest{Name: "ops"})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	if err := api.ReplaceRolePermissions(ctx, role.ID, []string{permission.ID}, nil); err != nil {
		t.Fatalf("failed to replace role permissions: %v", err)
	}

	if err := api.ReplaceUserRoles(ctx, "admin-user", []string{role.ID}, nil); err != nil {
		t.Fatalf("failed to assign user roles: %v", err)
	}

	uid := "admin-user"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/permissions", map[string]any{
		"permissions": []string{"admin.read"},
	}, &uid)

	err = plugin.requireRBAC(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCtx.Handled {
		t.Fatalf("expected request not to be handled when permissions are granted")
	}
	if reqCtx.ResponseReady {
		t.Fatalf("expected no response to be set when request is allowed")
	}
}

func TestAdminRBACHook_AllowsWhenAnyPermissionMatches(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "admin-any-perm", "admin-any-perm@example.com")

	api := newTestAPI(db)
	plugin := &AdminPlugin{Api: api}

	ctx := context.Background()
	permission, err := api.CreatePermission(ctx, types.CreatePermissionRequest{Key: "admin.read"})
	if err != nil {
		t.Fatalf("failed to create permission: %v", err)
	}

	role, err := api.CreateRole(ctx, types.CreateRoleRequest{Name: "ops-any"})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	if err := api.ReplaceRolePermissions(ctx, role.ID, []string{permission.ID}, nil); err != nil {
		t.Fatalf("failed to replace role permissions: %v", err)
	}

	if err := api.ReplaceUserRoles(ctx, "admin-any-perm", []string{role.ID}, nil); err != nil {
		t.Fatalf("failed to assign user roles: %v", err)
	}

	uid := "admin-any-perm"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/permissions", map[string]any{
		"permissions": []string{"missing.permission", "admin.read", "also.missing"},
	}, &uid)

	err = plugin.requireRBAC(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCtx.Handled {
		t.Fatalf("expected request not to be handled when one permission matches")
	}
}

func TestAdminRBACHook_DeniesWhenPermissionsMetadataMissing(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "admin-missing-meta", "admin-missing-meta@example.com")

	plugin := &AdminPlugin{Api: newTestAPI(db)}
	uid := "admin-missing-meta"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/permissions", map[string]any{
		"plugins": []string{"admin.rbac"},
	}, &uid)

	err := plugin.requireRBAC(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reqCtx.Handled {
		t.Fatalf("expected request to be handled when route permissions are not configured")
	}
	if reqCtx.ResponseStatus != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, reqCtx.ResponseStatus)
	}
}

func TestAdminRBACHook_DeniesWhenPermissionsMetadataOnlyWhitespace(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "admin-whitespace-meta", "admin-whitespace-meta@example.com")

	plugin := &AdminPlugin{Api: newTestAPI(db)}
	uid := "admin-whitespace-meta"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/permissions", map[string]any{
		"permissions": []string{"", "   ", "\t", "\n"},
	}, &uid)

	err := plugin.requireRBAC(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reqCtx.Handled {
		t.Fatalf("expected request to be handled when route permissions are blank/whitespace")
	}
	if reqCtx.ResponseStatus != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, reqCtx.ResponseStatus)
	}
}

func TestAdminStateHook_BannedUserGets403(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "banned-user", "banned@example.com")

	api := newTestAPI(db)
	plugin := &AdminPlugin{Api: api}

	_, err := api.UpsertUserState(context.Background(), "banned-user", types.UpsertUserStateRequest{IsBanned: true}, nil)
	if err != nil {
		t.Fatalf("failed to set user state: %v", err)
	}

	uid := "banned-user"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/roles", nil, &uid)

	err = plugin.enforceAdminState(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reqCtx.Handled {
		t.Fatalf("expected request to be handled")
	}
	if reqCtx.ResponseStatus != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, reqCtx.ResponseStatus)
	}
}

func TestAdminStateHook_RevokedSessionGets401(t *testing.T) {
	db := newAdminTestDB(t)
	seedUser(t, db, "session-user", "session-user@example.com")
	seedSession(t, db, "session-1", "session-user")

	api := newTestAPI(db)
	plugin := &AdminPlugin{Api: api}

	_, err := api.UpsertSessionState(context.Background(), "session-1", types.UpsertSessionStateRequest{Revoke: true}, nil)
	if err != nil {
		t.Fatalf("failed to set session state: %v", err)
	}

	uid := "session-user"
	reqCtx := newReqCtx(http.MethodGet, "/auth/admin/roles", nil, &uid)
	reqCtx.Values[models.ContextSessionID.String()] = "session-1"

	err = plugin.enforceAdminState(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reqCtx.Handled {
		t.Fatalf("expected request to be handled")
	}
	if reqCtx.ResponseStatus != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, reqCtx.ResponseStatus)
	}
}

func TestAdminStateHook_UnauthenticatedNoOp(t *testing.T) {
	db := newAdminTestDB(t)
	plugin := &AdminPlugin{Api: newTestAPI(db)}

	reqCtx := newReqCtx(http.MethodGet, "/auth/public/ping", nil, nil)
	reqCtx.Values[models.ContextSessionID.String()] = "session-should-not-be-evaluated"

	err := plugin.enforceAdminState(reqCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reqCtx.Handled {
		t.Fatalf("expected unauthenticated request to pass through without enforcement")
	}
	if reqCtx.ResponseReady {
		t.Fatalf("expected no response to be set for unauthenticated request")
	}
}
