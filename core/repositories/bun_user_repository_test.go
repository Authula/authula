package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/models"
)

const (
	failUserDeleteTrigger = `CREATE TRIGGER fail_user_delete
		BEFORE DELETE ON users
		BEGIN
			SELECT RAISE(ABORT, 'user delete blocked');
		END;`

	failUserUpdateTrigger = `CREATE TRIGGER fail_user_update
		BEFORE UPDATE ON users
		BEGIN
			SELECT RAISE(ABORT, 'user update blocked');
		END;`

	deleteUserAfterInsertTrigger = `CREATE TRIGGER delete_user_after_insert
		AFTER INSERT ON users
		BEGIN
			DELETE FROM users WHERE id = NEW.id;
		END;`
)

// userFixtureTime keeps fixture timestamps deterministic.
var userFixtureTime = time.Date(2024, time.January, 1, 12, 0, 0, 0, time.UTC)

// newTestUserDB creates an in-memory SQLite database with the users table.
func newTestUserDB(t *testing.T) bun.IDB {
	t.Helper()

	sqldb, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)

	// In-memory SQLite databases are per connection. A single connection keeps
	// every query, including transactions, on the same database.
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	_, err = db.NewCreateTable().Model(&models.User{}).Exec(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func newTestUser(id, email string) *models.User {
	return &models.User{
		ID:            id,
		Name:          "User " + id,
		Email:         email,
		EmailVerified: true,
		CreatedAt:     userFixtureTime,
		UpdatedAt:     userFixtureTime,
	}
}

func createTestUsers(t *testing.T, ctx context.Context, repo UserRepository, users ...*models.User) {
	t.Helper()

	for _, user := range users {
		// Clone so parallel subtests never mutate the same fixture model.
		clone := *user
		_, err := repo.Create(ctx, &clone)
		require.NoError(t, err)
	}
}

func blockUserDeletes(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()

	execTestSQL(t, ctx, db, failUserDeleteTrigger)
}

func blockUserUpdates(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()

	execTestSQL(t, ctx, db, failUserUpdateTrigger)
}

func deleteUserAfterInsert(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()

	execTestSQL(t, ctx, db, deleteUserAfterInsertTrigger)
}

func userIDs(t *testing.T, ctx context.Context, db bun.IDB) []string {
	t.Helper()

	var users []*models.User
	require.NoError(t, db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx))

	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func TestBunUserRepository_GetAll(t *testing.T) {
	t.Parallel()

	threeUsers := func() []*models.User {
		return []*models.User{
			newTestUser("user-1", "user1@example.com"),
			newTestUser("user-2", "user2@example.com"),
			newTestUser("user-3", "user3@example.com"),
		}
	}

	tests := []struct {
		name          string
		seed          func() []*models.User
		cursor        *string
		limit         int
		cancelContext bool
		wantIDs       []string
		wantNext      *string
		wantErrIs     error
	}{
		{
			name:     "returns the first page with a next cursor",
			seed:     threeUsers,
			limit:    1,
			wantIDs:  []string{"user-1"},
			wantNext: new("user-1"),
		},
		{
			name:     "continues after the cursor",
			seed:     threeUsers,
			cursor:   new("user-1"),
			limit:    1,
			wantIDs:  []string{"user-2"},
			wantNext: new("user-2"),
		},
		{
			name:     "returns the final page with a nil cursor",
			seed:     threeUsers,
			cursor:   new("user-2"),
			limit:    1,
			wantIDs:  []string{"user-3"},
			wantNext: nil,
		},
		{
			name:     "returns a full page with a next cursor when more rows remain",
			seed:     threeUsers,
			limit:    2,
			wantIDs:  []string{"user-1", "user-2"},
			wantNext: new("user-2"),
		},
		{
			name:     "returns the last partial page with a nil cursor",
			seed:     threeUsers,
			cursor:   new("user-1"),
			limit:    2,
			wantIDs:  []string{"user-2", "user-3"},
			wantNext: nil,
		},
		{
			name:     "trims whitespace around the cursor",
			seed:     threeUsers,
			cursor:   new("  user-1  "),
			limit:    1,
			wantIDs:  []string{"user-2"},
			wantNext: new("user-2"),
		},
		{
			name:     "ignores a blank cursor",
			seed:     threeUsers,
			cursor:   new("   "),
			limit:    2,
			wantIDs:  []string{"user-1", "user-2"},
			wantNext: new("user-2"),
		},
		{
			name:    "returns an empty slice and nil cursor for an empty table",
			limit:   2,
			wantIDs: []string{},
		},
		{
			name:    "returns an empty slice when the cursor is past the last row",
			seed:    threeUsers,
			cursor:  new("user-3"),
			limit:   2,
			wantIDs: []string{},
		},
		{
			name:          "returns an error when the query fails",
			seed:          threeUsers,
			limit:         2,
			cancelContext: true,
			wantIDs:       []string{},
			wantErrIs:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestUserDB(t)
			repo := NewBunUserRepository(db)

			if tt.seed != nil {
				createTestUsers(t, ctx, repo, tt.seed()...)
			}

			callCtx := ctx
			if tt.cancelContext {
				callCtx = cancelledContext()
			}

			got, next, err := repo.GetAll(callCtx, tt.cursor, tt.limit)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Empty(t, got)
				assert.Nil(t, next)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantIDs, userIDsFrom(got))
			if tt.wantNext == nil {
				assert.Nil(t, next)
			} else {
				require.NotNil(t, next)
				assert.Equal(t, *tt.wantNext, *next)
			}
		})
	}
}

// userIDsFrom extracts IDs in result order without re-querying the database.
func userIDsFrom(users []models.User) []string {
	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}

func TestBunUserRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		seed          func() *models.User
		user          func() *models.User
		prepare       func(*testing.T, context.Context, bun.IDB)
		wantErr       bool
		wantErrIs     error
		wantEmail     string
		wantRemaining []string
	}{
		{
			name:          "creates and reloads the user",
			user:          func() *models.User { return newTestUser("user-1", "user1@example.com") },
			wantEmail:     "user1@example.com",
			wantRemaining: []string{"user-1"},
		},
		{
			name:          "normalizes the email on insert",
			user:          func() *models.User { return newTestUser("user-1", "  User1@Example.COM  ") },
			wantEmail:     "user1@example.com",
			wantRemaining: []string{"user-1"},
		},
		{
			name:          "returns an error when the insert fails",
			seed:          func() *models.User { return newTestUser("user-1", "user1@example.com") },
			user:          func() *models.User { return newTestUser("user-1", "other@example.com") },
			wantErr:       true,
			wantRemaining: []string{"user-1"},
		},
		{
			name:          "returns an error when the reload fails after insert",
			user:          func() *models.User { return newTestUser("user-reload", "reload@example.com") },
			prepare:       deleteUserAfterInsert,
			wantErrIs:     sql.ErrNoRows,
			wantRemaining: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestUserDB(t)
			repo := NewBunUserRepository(db)

			if tt.seed != nil {
				createTestUsers(t, ctx, repo, tt.seed())
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			user := tt.user()
			got, err := repo.Create(ctx, user)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			} else if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Same(t, user, got)

			if tt.wantEmail != "" {
				assert.Equal(t, tt.wantEmail, got.Email)
			}
			assert.Equal(t, tt.wantRemaining, userIDs(t, ctx, db))
		})
	}
}

func TestBunUserRepository_Getters(t *testing.T) {
	t.Parallel()

	type getter func(UserRepository, context.Context, string) (*models.User, error)

	tests := []struct {
		name          string
		get           getter
		key           string
		cancelContext bool
		wantID        string
		wantErrIs     error
	}{
		{
			name:   "GetByID returns the matching user",
			get:    UserRepository.GetByID,
			key:    "user-1",
			wantID: "user-1",
		},
		{
			name:   "GetByID returns nil for an unknown id",
			get:    UserRepository.GetByID,
			key:    "missing",
			wantID: "",
		},
		{
			name:          "GetByID returns a query error",
			get:           UserRepository.GetByID,
			key:           "user-1",
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
		{
			name:   "GetByEmail returns the matching user",
			get:    UserRepository.GetByEmail,
			key:    "user1@example.com",
			wantID: "user-1",
		},
		{
			name:   "GetByEmail normalizes the lookup input",
			get:    UserRepository.GetByEmail,
			key:    "  USER1@EXAMPLE.COM  ",
			wantID: "user-1",
		},
		{
			name:   "GetByEmail returns nil for an unknown email",
			get:    UserRepository.GetByEmail,
			key:    "missing@example.com",
			wantID: "",
		},
		{
			name:          "GetByEmail returns a query error",
			get:           UserRepository.GetByEmail,
			key:           "user1@example.com",
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestUserDB(t)
			repo := NewBunUserRepository(db)

			createTestUsers(t, ctx, repo,
				newTestUser("user-1", "user1@example.com"),
				newTestUser("user-2", "user2@example.com"),
			)

			if tt.cancelContext {
				ctx = cancelledContext()
			}

			got, err := tt.get(repo, ctx, tt.key)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				return
			}

			require.NoError(t, err)
			if tt.wantID == "" {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.ID)
		})
	}
}

func TestBunUserRepository_Update(t *testing.T) {
	t.Parallel()

	newSeed := func() *models.User {
		return newTestUser("user-1", "user1@example.com")
	}

	tests := []struct {
		name          string
		seed          func() *models.User
		user          func() *models.User
		prepare       func(*testing.T, context.Context, bun.IDB)
		wantErr       bool
		wantErrIs     error
		wantEmail     string
		wantRemaining []string
	}{
		{
			name:          "updates and reloads the user",
			seed:          newSeed,
			user:          func() *models.User { return newTestUser("user-1", "changed@example.com") },
			wantEmail:     "changed@example.com",
			wantRemaining: []string{"user-1"},
		},
		{
			name:          "normalizes the email on update",
			seed:          newSeed,
			user:          func() *models.User { return newTestUser("user-1", "  Changed@Example.COM  ") },
			wantEmail:     "changed@example.com",
			wantRemaining: []string{"user-1"},
		},
		{
			name:          "returns an error when the update fails",
			seed:          newSeed,
			user:          func() *models.User { return newTestUser("user-1", "changed@example.com") },
			prepare:       blockUserUpdates,
			wantErr:       true,
			wantRemaining: []string{"user-1"},
		},
		{
			name:          "returns an error when the user does not exist",
			user:          func() *models.User { return newTestUser("user-missing", "missing@example.com") },
			wantErrIs:     sql.ErrNoRows,
			wantRemaining: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestUserDB(t)
			repo := NewBunUserRepository(db)

			if tt.seed != nil {
				createTestUsers(t, ctx, repo, tt.seed())
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			user := tt.user()
			got, err := repo.Update(ctx, user)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			} else if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Same(t, user, got)

			if tt.wantEmail != "" {
				assert.Equal(t, tt.wantEmail, got.Email)

				reloaded, err := repo.GetByID(context.Background(), user.ID)
				require.NoError(t, err)
				require.NotNil(t, reloaded)
				assert.Equal(t, tt.wantEmail, reloaded.Email)
			}
			assert.Equal(t, tt.wantRemaining, userIDs(t, ctx, db))
		})
	}
}

func TestBunUserRepository_Delete(t *testing.T) {
	t.Parallel()

	newSeed := func() []*models.User {
		return []*models.User{
			newTestUser("user-1", "user1@example.com"),
			newTestUser("user-2", "user2@example.com"),
		}
	}

	tests := []struct {
		name          string
		seed          func() []*models.User
		id            string
		prepare       func(*testing.T, context.Context, bun.IDB)
		wantErr       bool
		wantRemaining []string
	}{
		{
			name:          "deletes the requested user",
			seed:          newSeed,
			id:            "user-1",
			wantRemaining: []string{"user-2"},
		},
		{
			name:          "is a no-op when the user does not exist",
			seed:          newSeed,
			id:            "missing",
			wantRemaining: []string{"user-1", "user-2"},
		},
		{
			name:          "returns an error when the delete fails",
			seed:          newSeed,
			id:            "user-1",
			prepare:       blockUserDeletes,
			wantErr:       true,
			wantRemaining: []string{"user-1", "user-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestUserDB(t)
			repo := NewBunUserRepository(db)

			if tt.seed != nil {
				createTestUsers(t, ctx, repo, tt.seed()...)
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.Delete(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRemaining, userIDs(t, ctx, db))
		})
	}
}
