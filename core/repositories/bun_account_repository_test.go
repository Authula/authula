package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/models"
)

const (
	failAccountDeleteTrigger = `CREATE TRIGGER fail_account_delete
		BEFORE DELETE ON accounts
		BEGIN
			SELECT RAISE(ABORT, 'account delete blocked');
		END;`

	failAccountUpdateTrigger = `CREATE TRIGGER fail_account_update
		BEFORE UPDATE ON accounts
		BEGIN
			SELECT RAISE(ABORT, 'account update blocked');
		END;`

	deleteAccountAfterInsertTrigger = `CREATE TRIGGER delete_account_after_insert
		AFTER INSERT ON accounts
		BEGIN
			DELETE FROM accounts WHERE id = NEW.id;
		END;`
)

// newTestAccountDB creates an in-memory SQLite database with the accounts table.
func newTestAccountDB(t *testing.T) bun.IDB {
	t.Helper()

	sqldb, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)

	// In-memory SQLite databases are per connection. A single connection keeps
	// every query, including transactions, on the same database.
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	_, err = db.NewCreateTable().Model(&models.Account{}).Exec(context.Background())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func accountFixtureTime() time.Time {
	return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
}

// newTestAccountFixtures returns accounts where user-1 owns two providers and
// user-2 owns one.
func newTestAccountFixtures() []*models.Account {
	createdAt := accountFixtureTime()

	emailAccount := newTestAccount("account-1", "user-1", models.AuthProviderEmail.String(), "alice@example.com")
	emailAccount.Password = new("password-hash")
	emailAccount.Scope = new("openid profile")
	emailAccount.CreatedAt = createdAt
	emailAccount.UpdatedAt = createdAt

	googleAccount := newTestAccount("account-2", "user-1", models.AuthProviderGoogle.String(), "google-sub-1")
	googleAccount.AccessToken = new("access-token-2")
	googleAccount.RefreshToken = new("refresh-token-2")
	googleAccount.IDToken = new("id-token-2")
	googleAccount.AccessTokenExpiresAt = new(createdAt.Add(time.Hour))
	googleAccount.RefreshTokenExpiresAt = new(createdAt.Add(24 * time.Hour))
	googleAccount.Scope = new("email")
	googleAccount.CreatedAt = createdAt
	googleAccount.UpdatedAt = createdAt

	githubAccount := newTestAccount("account-3", "user-2", models.AuthProviderGitHub.String(), "gh-1")
	githubAccount.CreatedAt = createdAt
	githubAccount.UpdatedAt = createdAt

	return []*models.Account{emailAccount, googleAccount, githubAccount}
}

func newTestAccount(id, userID, providerID, accountID string) *models.Account {
	return &models.Account{
		ID:         id,
		UserID:     userID,
		ProviderID: providerID,
		AccountID:  accountID,
	}
}

func createTestAccounts(t *testing.T, ctx context.Context, repo AccountRepository, accounts ...*models.Account) {
	t.Helper()

	for _, account := range accounts {
		// Clone so parallel subtests never mutate the same fixture model.
		clone := *account
		_, err := repo.Create(ctx, &clone)
		require.NoError(t, err)
	}
}

func execAccountTestSQL(t *testing.T, ctx context.Context, db bun.IDB, query string) {
	t.Helper()

	_, err := db.ExecContext(ctx, query)
	require.NoError(t, err)
}

func blockAccountDeletes(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()
	execAccountTestSQL(t, ctx, db, failAccountDeleteTrigger)
}

func blockAccountUpdates(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()
	execAccountTestSQL(t, ctx, db, failAccountUpdateTrigger)
}

func deleteAccountAfterInsert(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()
	execAccountTestSQL(t, ctx, db, deleteAccountAfterInsertTrigger)
}

func accountRowIDs(t *testing.T, ctx context.Context, db bun.IDB, userID *string) []string {
	t.Helper()

	var accounts []*models.Account
	query := db.NewSelect().Model(&accounts)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	require.NoError(t, query.Scan(ctx))

	ids := make([]string, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	return ids
}

func cancelledAccountContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func requireAccountFields(t *testing.T, want, got *models.Account) {
	t.Helper()

	require.NotNil(t, got)
	require.Equal(t, want.ID, got.ID)
	require.Equal(t, want.UserID, got.UserID)
	require.Equal(t, want.AccountID, got.AccountID)
	require.Equal(t, want.ProviderID, got.ProviderID)
	require.Equal(t, want.AccessToken, got.AccessToken)
	require.Equal(t, want.RefreshToken, got.RefreshToken)
	require.Equal(t, want.IDToken, got.IDToken)
	require.Equal(t, want.Scope, got.Scope)
	require.Equal(t, want.Password, got.Password)
	require.WithinDuration(t, want.CreatedAt, got.CreatedAt, time.Millisecond)
	require.WithinDuration(t, want.UpdatedAt, got.UpdatedAt, time.Millisecond)

	if want.AccessTokenExpiresAt == nil {
		require.Nil(t, got.AccessTokenExpiresAt)
	} else {
		require.NotNil(t, got.AccessTokenExpiresAt)
		require.WithinDuration(t, *want.AccessTokenExpiresAt, *got.AccessTokenExpiresAt, time.Millisecond)
	}
	if want.RefreshTokenExpiresAt == nil {
		require.Nil(t, got.RefreshTokenExpiresAt)
	} else {
		require.NotNil(t, got.RefreshTokenExpiresAt)
		require.WithinDuration(t, *want.RefreshTokenExpiresAt, *got.RefreshTokenExpiresAt, time.Millisecond)
	}
}

func TestBunAccountRepository_Getters(t *testing.T) {
	t.Parallel()

	type getter func(context.Context, AccountRepository) (*models.Account, error)

	tests := []struct {
		name          string
		get           getter
		cancelContext bool
		wantID        string
		wantErrIs     error
	}{
		{
			name: "GetByID returns the matching account",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByID(ctx, "account-2")
			},
			wantID: "account-2",
		},
		{
			name: "GetByID returns nil for an unknown id",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByID(ctx, "missing")
			},
		},
		{
			name: "GetByID returns a query error",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByID(ctx, "account-1")
			},
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
		{
			name: "GetByUserID returns the matching account",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserID(ctx, "user-2")
			},
			wantID: "account-3",
		},
		{
			name: "GetByUserID returns nil when the user has no accounts",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserID(ctx, "user-without-accounts")
			},
		},
		{
			name: "GetByUserID returns a query error",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserID(ctx, "user-2")
			},
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
		{
			name: "GetByUserIDAndProvider returns the matching account",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserIDAndProvider(ctx, "user-1", models.AuthProviderEmail.String())
			},
			wantID: "account-1",
		},
		{
			name: "GetByUserIDAndProvider returns nil when the user has no accounts",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserIDAndProvider(ctx, "user-without-accounts", models.AuthProviderEmail.String())
			},
		},
		{
			name: "GetByUserIDAndProvider returns nil on a provider mismatch",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserIDAndProvider(ctx, "user-1", models.AuthProviderGitHub.String())
			},
		},
		{
			name: "GetByUserIDAndProvider returns a query error",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByUserIDAndProvider(ctx, "user-1", models.AuthProviderEmail.String())
			},
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
		{
			name: "GetByProviderAndAccountID returns the matching account",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByProviderAndAccountID(ctx, models.AuthProviderGitHub.String(), "gh-1")
			},
			wantID: "account-3",
		},
		{
			name: "GetByProviderAndAccountID returns nil for an unknown account id",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByProviderAndAccountID(ctx, models.AuthProviderGitHub.String(), "missing")
			},
		},
		{
			name: "GetByProviderAndAccountID returns a query error",
			get: func(ctx context.Context, repo AccountRepository) (*models.Account, error) {
				return repo.GetByProviderAndAccountID(ctx, models.AuthProviderEmail.String(), "alice@example.com")
			},
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)
			createTestAccounts(t, ctx, repo, newTestAccountFixtures()...)

			if tt.cancelContext {
				ctx = cancelledAccountContext()
			}

			got, err := tt.get(ctx, repo)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				return
			}

			require.NoError(t, err)
			if tt.wantID == "" {
				require.Nil(t, got)
				return
			}

			for _, want := range newTestAccountFixtures() {
				if want.ID == tt.wantID {
					requireAccountFields(t, want, got)
					return
				}
			}
			t.Fatalf("fixture account %q not found", tt.wantID)
		})
	}
}

func TestBunAccountRepository_GetAllByUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        string
		cancelContext bool
		wantErrIs     error
		wantIDs       []string
	}{
		{
			name:    "returns every account for the user",
			userID:  "user-1",
			wantIDs: []string{"account-1", "account-2"},
		},
		{
			name:    "returns an empty slice when the user has no accounts",
			userID:  "user-without-accounts",
			wantIDs: []string{},
		},
		{
			name:          "returns a query error",
			userID:        "user-1",
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)
			createTestAccounts(t, ctx, repo, newTestAccountFixtures()...)

			if tt.cancelContext {
				ctx = cancelledAccountContext()
			}

			got, err := repo.GetAllByUserID(ctx, tt.userID)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			ids := make([]string, 0, len(got))
			for _, account := range got {
				ids = append(ids, account.ID)
			}
			require.ElementsMatch(t, tt.wantIDs, ids)
		})
	}
}

func TestBunAccountRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		seed             func() []*models.Account
		account          func() *models.Account
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantErrIs        error
		wantAccountID    string
		wantRemainingIDs []string
	}{
		{
			name: "persists and reloads the account",
			account: func() *models.Account {
				account := newTestAccount("account-new", "user-1", models.AuthProviderGoogle.String(), "Google-Sub-New")
				account.AccessToken = new("access-token-new")
				account.CreatedAt = accountFixtureTime()
				account.UpdatedAt = accountFixtureTime()
				return account
			},
			wantAccountID:    "Google-Sub-New",
			wantRemainingIDs: []string{"account-new"},
		},
		{
			name: "normalizes an email provider account id",
			account: func() *models.Account {
				return newTestAccount("account-email", "user-1", models.AuthProviderEmail.String(), "  NewUser@Example.COM  ")
			},
			wantAccountID:    "newuser@example.com",
			wantRemainingIDs: []string{"account-email"},
		},
		{
			name: "returns an error when the insert fails",
			seed: func() []*models.Account {
				return []*models.Account{
					newTestAccount("account-duplicate", "user-1", models.AuthProviderEmail.String(), "duplicate@example.com"),
				}
			},
			account: func() *models.Account {
				return newTestAccount("account-duplicate", "user-1", models.AuthProviderEmail.String(), "duplicate@example.com")
			},
			wantErr:          true,
			wantRemainingIDs: []string{"account-duplicate"},
		},
		{
			name: "returns an error when the reload fails after insert",
			account: func() *models.Account {
				return newTestAccount("account-reload", "user-1", models.AuthProviderEmail.String(), "reload@example.com")
			},
			prepare:          deleteAccountAfterInsert,
			wantErrIs:        sql.ErrNoRows,
			wantRemainingIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)

			if tt.seed != nil {
				createTestAccounts(t, ctx, repo, tt.seed()...)
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			account := tt.account()
			got, err := repo.Create(ctx, account)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Nil(t, got)
			} else if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Same(t, account, got)
			}

			if tt.wantAccountID != "" {
				reloaded, err := repo.GetByID(ctx, account.ID)
				require.NoError(t, err)
				require.Equal(t, tt.wantAccountID, reloaded.AccountID)
			}

			require.ElementsMatch(t, tt.wantRemainingIDs, accountRowIDs(t, ctx, db, nil))
		})
	}
}

func TestBunAccountRepository_Update(t *testing.T) {
	t.Parallel()

	newSeed := func() *models.Account {
		return newTestAccount("account-1", "user-1", models.AuthProviderEmail.String(), "old@example.com")
	}
	newUpdate := func() *models.Account {
		account := newTestAccount("account-1", "user-1", models.AuthProviderEmail.String(), "  New@Example.COM  ")
		account.Scope = new("openid")
		account.Password = new("new-password-hash")
		return account
	}

	tests := []struct {
		name          string
		seed          func() *models.Account
		account       func() *models.Account
		prepare       func(*testing.T, context.Context, bun.IDB)
		wantErr       bool
		wantErrIs     error
		wantAccountID string
		wantScope     *string
	}{
		{
			name:          "normalizes and reloads the updated account",
			seed:          newSeed,
			account:       newUpdate,
			wantAccountID: "new@example.com",
			wantScope:     new("openid"),
		},
		{
			name: "does not normalize a non-email provider account id",
			seed: func() *models.Account {
				return newTestAccount("account-1", "user-1", models.AuthProviderGoogle.String(), "old-sub")
			},
			account: func() *models.Account {
				return newTestAccount("account-1", "user-1", models.AuthProviderGoogle.String(), "New-Sub")
			},
			wantAccountID: "New-Sub",
		},
		{
			name:          "returns an error when the update fails",
			seed:          newSeed,
			account:       newUpdate,
			prepare:       blockAccountUpdates,
			wantErr:       true,
			wantAccountID: "old@example.com",
		},
		{
			name: "returns an error when the account does not exist",
			account: func() *models.Account {
				return newTestAccount("account-missing", "user-1", models.AuthProviderEmail.String(), "missing@example.com")
			},
			wantErrIs: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)

			if tt.seed != nil {
				createTestAccounts(t, ctx, repo, tt.seed())
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			account := tt.account()
			got, err := repo.Update(ctx, account)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Nil(t, got)
			} else if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Same(t, account, got)
			}

			if tt.wantAccountID != "" {
				reloaded, err := repo.GetByID(ctx, account.ID)
				require.NoError(t, err)
				require.Equal(t, tt.wantAccountID, reloaded.AccountID)
				require.Equal(t, tt.wantScope, reloaded.Scope)
			}
		})
	}
}

func TestBunAccountRepository_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		id               string
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name:             "deletes the requested account",
			id:               "account-1",
			wantRemainingIDs: []string{"account-2", "account-3"},
		},
		{
			name:             "is a no-op when the account does not exist",
			id:               "missing",
			wantRemainingIDs: []string{"account-1", "account-2", "account-3"},
		},
		{
			name:             "returns an error when the delete fails",
			id:               "account-1",
			prepare:          blockAccountDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"account-1", "account-2", "account-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)
			createTestAccounts(t, ctx, repo, newTestAccountFixtures()...)

			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.Delete(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, accountRowIDs(t, ctx, db, nil))
		})
	}
}

func TestBunAccountRepository_UpdateFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        string
		fields        map[string]any
		prepare       func(*testing.T, context.Context, bun.IDB)
		wantErr       bool
		wantScopes    map[string]*string
		wantPasswords map[string]*string
		wantUntouched string
	}{
		{
			name:   "updates every account owned by the user",
			userID: "user-1",
			fields: map[string]any{"scope": "read write", "password": "hashed"},
			wantScopes: map[string]*string{
				"account-1": new("read write"),
				"account-2": new("read write"),
			},
			wantPasswords: map[string]*string{
				"account-1": new("hashed"),
				"account-2": new("hashed"),
			},
			wantUntouched: "account-3",
		},
		{
			name:   "is a no-op when no account matches the user",
			userID: "user-without-accounts",
			fields: map[string]any{"scope": "ignored"},
			wantScopes: map[string]*string{
				"account-1": new("openid profile"),
				"account-2": new("email"),
			},
			wantUntouched: "account-3",
		},
		{
			name:    "returns an error when the update fails",
			userID:  "user-1",
			fields:  map[string]any{"scope": "blocked"},
			prepare: blockAccountUpdates,
			wantErr: true,
			wantScopes: map[string]*string{
				"account-1": new("openid profile"),
				"account-2": new("email"),
			},
			wantUntouched: "account-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)
			createTestAccounts(t, ctx, repo, newTestAccountFixtures()...)

			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.UpdateFields(ctx, tt.userID, tt.fields)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			for id, wantScope := range tt.wantScopes {
				got, err := repo.GetByID(ctx, id)
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, wantScope, got.Scope)

				if tt.wantPasswords != nil {
					require.Equal(t, tt.wantPasswords[id], got.Password)
				}
			}

			untouched, err := repo.GetByID(ctx, tt.wantUntouched)
			require.NoError(t, err)
			require.NotNil(t, untouched)
			require.Nil(t, untouched.Scope)
			require.Nil(t, untouched.Password)
		})
	}
}

func TestBunAccountRepository_WithTx(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		commit bool
	}{
		{name: "commits writes through the transaction", commit: true},
		{name: "rolls back writes through the transaction", commit: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestAccountDB(t)
			repo := NewBunAccountRepository(db)

			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			txRepo := repo.WithTx(tx)
			require.NotNil(t, txRepo)

			account := newTestAccount("account-tx", "user-1", models.AuthProviderEmail.String(), "tx@example.com")
			created, err := txRepo.Create(ctx, account)
			require.NoError(t, err)
			require.Equal(t, account.ID, created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			got, err := repo.GetByID(ctx, account.ID)
			require.NoError(t, err)

			if tt.commit {
				require.NotNil(t, got)
				require.Equal(t, account.ID, got.ID)
				return
			}

			require.Nil(t, got)
		})
	}
}
