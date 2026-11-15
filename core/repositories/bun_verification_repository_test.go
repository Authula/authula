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

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/models"
)

const (
	failVerificationDeleteTrigger = `CREATE TRIGGER fail_verification_delete
		BEFORE DELETE ON verifications
		BEGIN
			SELECT RAISE(ABORT, 'verification delete blocked');
		END;`

	deleteVerificationAfterInsertTrigger = `CREATE TRIGGER delete_verification_after_insert
		AFTER INSERT ON verifications
		BEGIN
			DELETE FROM verifications WHERE id = NEW.id;
		END;`
)

// newTestVerificationDB creates an in-memory SQLite database with the
// verification table.
func newTestVerificationDB(t *testing.T) bun.IDB {
	t.Helper()

	sqldb, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)

	// In-memory SQLite databases are per connection. A single connection keeps
	// every query, including transactions, on the same database.
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	_, err = db.NewCreateTable().Model(&models.Verification{}).Exec(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func newTestVerification(id, userID string, vType models.VerificationType, expiresAt time.Time) *models.Verification {
	return &models.Verification{
		ID:         id,
		UserID:     &userID,
		Identifier: "user@example.com",
		Token:      "token-" + id,
		Type:       vType,
		ExpiresAt:  expiresAt,
	}
}

func createTestVerifications(t *testing.T, ctx context.Context, repo VerificationRepository, verifications ...*models.Verification) {
	t.Helper()

	for _, verification := range verifications {
		// Clone so parallel subtests never mutate the same fixture model.
		clone := *verification
		_, err := repo.Create(ctx, &clone)
		require.NoError(t, err)
	}
}

func blockVerificationDeletes(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()

	execTestSQL(t, ctx, db, failVerificationDeleteTrigger)
}

func deleteVerificationAfterInsert(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()

	execTestSQL(t, ctx, db, deleteVerificationAfterInsertTrigger)
}

func verificationIDs(t *testing.T, ctx context.Context, db bun.IDB) []string {
	t.Helper()

	var verifications []*models.Verification
	require.NoError(t, db.NewSelect().Model(&verifications).Scan(ctx))

	ids := make([]string, 0, len(verifications))
	for _, verification := range verifications {
		ids = append(ids, verification.ID)
	}
	return ids
}

func TestBunVerificationRepository_Getters(t *testing.T) {
	t.Parallel()

	type getter func(VerificationRepository, context.Context, string) (*models.Verification, error)

	const (
		wantedID = "verification-wanted"
		otherID  = "verification-other"
	)

	now := time.Now().UTC()

	tests := []struct {
		name          string
		get           getter
		key           string
		cancelContext bool
		wantID        string
		wantErr       error
	}{
		{
			name:   "GetByID returns the matching verification",
			get:    VerificationRepository.GetByID,
			key:    wantedID,
			wantID: wantedID,
		},
		{
			name:    "GetByID returns ErrVerificationNotFound for an unknown id",
			get:     VerificationRepository.GetByID,
			key:     "missing",
			wantErr: coreerrors.ErrVerificationNotFound,
		},
		{
			name:          "GetByID returns a query error",
			get:           VerificationRepository.GetByID,
			key:           wantedID,
			cancelContext: true,
			wantErr:       context.Canceled,
		},
		{
			name:   "GetByToken returns the matching verification",
			get:    VerificationRepository.GetByToken,
			key:    "token-" + wantedID,
			wantID: wantedID,
		},
		{
			name:    "GetByToken returns ErrVerificationNotFound for an unknown token",
			get:     VerificationRepository.GetByToken,
			key:     "missing",
			wantErr: coreerrors.ErrVerificationNotFound,
		},
		{
			name:          "GetByToken returns a query error",
			get:           VerificationRepository.GetByToken,
			key:           "token-" + wantedID,
			cancelContext: true,
			wantErr:       context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestVerificationDB(t)
			repo := NewBunVerificationRepository(db)

			createTestVerifications(t, ctx, repo,
				newTestVerification(wantedID, "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
				newTestVerification(otherID, "user-2", models.TypePasswordResetRequest, now.Add(2*time.Hour)),
			)

			if tt.cancelContext {
				ctx = cancelledContext()
			}

			got, err := tt.get(repo, ctx, tt.key)

			if tt.wantErr != nil {
				require.Nil(t, got)
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.wantID, got.ID)
		})
	}
}

func TestBunVerificationRepository_Create(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	newVerification := func(id string) *models.Verification {
		return newTestVerification(id, "user-1", models.TypeEmailVerification, now.Add(time.Hour))
	}

	tests := []struct {
		name             string
		seed             func() []*models.Verification
		verification     func() *models.Verification
		prepare          func(*testing.T, context.Context, bun.IDB)
		cancelContext    bool
		wantErr          bool
		wantErrIs        error
		wantPersisted    bool
		wantRemainingIDs []string
	}{
		{
			name:             "creates and reloads the verification",
			verification:     func() *models.Verification { return newVerification("verification-1") },
			wantPersisted:    true,
			wantRemainingIDs: []string{"verification-1"},
		},
		{
			name: "returns an error when the insert fails",
			seed: func() []*models.Verification {
				return []*models.Verification{newVerification("verification-duplicate")}
			},
			verification:     func() *models.Verification { return newVerification("verification-duplicate") },
			wantErr:          true,
			wantRemainingIDs: []string{"verification-duplicate"},
		},
		{
			name:             "returns an error when the reload fails after insert",
			verification:     func() *models.Verification { return newVerification("verification-reload") },
			prepare:          deleteVerificationAfterInsert,
			wantErrIs:        sql.ErrNoRows,
			wantRemainingIDs: []string{},
		},
		{
			name:             "returns an error when the context is cancelled",
			verification:     func() *models.Verification { return newVerification("verification-cancelled") },
			cancelContext:    true,
			wantErrIs:        context.Canceled,
			wantRemainingIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestVerificationDB(t)
			repo := NewBunVerificationRepository(db)

			if tt.seed != nil {
				createTestVerifications(t, ctx, repo, tt.seed()...)
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}
			if tt.cancelContext {
				ctx = cancelledContext()
			}

			verification := tt.verification()
			got, err := repo.Create(ctx, verification)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			} else if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Same(t, verification, got)

			verifyCtx := context.Background()
			if tt.wantPersisted {
				reloaded, err := repo.GetByID(verifyCtx, verification.ID)
				require.NoError(t, err)
				require.Equal(t, verification.ID, reloaded.ID)
				require.Equal(t, verification.UserID, reloaded.UserID)
				require.Equal(t, verification.Identifier, reloaded.Identifier)
				require.Equal(t, verification.Token, reloaded.Token)
				require.Equal(t, verification.Type, reloaded.Type)
				require.WithinDuration(t, verification.ExpiresAt, reloaded.ExpiresAt, time.Millisecond)
			}

			require.ElementsMatch(t, tt.wantRemainingIDs, verificationIDs(t, verifyCtx, db))
		})
	}
}

func TestBunVerificationRepository_Delete(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	newSeed := func() []*models.Verification {
		return []*models.Verification{
			newTestVerification("verification-1", "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
			newTestVerification("verification-2", "user-1", models.TypePasswordResetRequest, now.Add(time.Hour)),
		}
	}

	tests := []struct {
		name             string
		seed             func() []*models.Verification
		id               string
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name:             "deletes the requested verification",
			seed:             newSeed,
			id:               "verification-1",
			wantRemainingIDs: []string{"verification-2"},
		},
		{
			name:             "is a no-op when the verification does not exist",
			seed:             newSeed,
			id:               "missing",
			wantRemainingIDs: []string{"verification-1", "verification-2"},
		},
		{
			name:             "returns an error when the delete fails",
			seed:             newSeed,
			id:               "verification-1",
			prepare:          blockVerificationDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"verification-1", "verification-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestVerificationDB(t)
			repo := NewBunVerificationRepository(db)

			createTestVerifications(t, ctx, repo, tt.seed()...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.Delete(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, verificationIDs(t, ctx, db))
		})
	}
}

func TestBunVerificationRepository_DeleteByUserIDAndType(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	newSeed := func() []*models.Verification {
		return []*models.Verification{
			newTestVerification("user-1-email-a", "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
			newTestVerification("user-1-email-b", "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
			newTestVerification("user-1-password", "user-1", models.TypePasswordResetRequest, now.Add(time.Hour)),
			newTestVerification("user-2-email", "user-2", models.TypeEmailVerification, now.Add(time.Hour)),
		}
	}

	tests := []struct {
		name             string
		seed             func() []*models.Verification
		userID           string
		vType            models.VerificationType
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name:             "deletes only verifications for the user and type",
			seed:             newSeed,
			userID:           "user-1",
			vType:            models.TypeEmailVerification,
			wantRemainingIDs: []string{"user-1-password", "user-2-email"},
		},
		{
			name:             "is a no-op when no verification matches",
			seed:             newSeed,
			userID:           "user-3",
			vType:            models.TypeEmailVerification,
			wantRemainingIDs: []string{"user-1-email-a", "user-1-email-b", "user-1-password", "user-2-email"},
		},
		{
			name:             "returns an error when the delete fails",
			seed:             newSeed,
			userID:           "user-1",
			vType:            models.TypeEmailVerification,
			prepare:          blockVerificationDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"user-1-email-a", "user-1-email-b", "user-1-password", "user-2-email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestVerificationDB(t)
			repo := NewBunVerificationRepository(db)

			createTestVerifications(t, ctx, repo, tt.seed()...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.DeleteByUserIDAndType(ctx, tt.userID, tt.vType)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, verificationIDs(t, ctx, db))
		})
	}
}

func TestBunVerificationRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name             string
		verifications    []*models.Verification
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name: "deletes only expired verifications",
			verifications: []*models.Verification{
				newTestVerification("expired-1", "user-1", models.TypeEmailVerification, now.Add(-2*time.Hour)),
				newTestVerification("expired-2", "user-2", models.TypePasswordResetRequest, now.Add(-30*time.Minute)),
				newTestVerification("active-1", "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
				newTestVerification("active-2", "user-3", models.TypeMagicLinkSignInRequest, now.Add(2*time.Hour)),
			},
			wantRemainingIDs: []string{"active-1", "active-2"},
		},
		{
			name: "keeps all verifications when none are expired",
			verifications: []*models.Verification{
				newTestVerification("active-1", "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
				newTestVerification("active-2", "user-2", models.TypePasswordResetRequest, now.Add(2*time.Hour)),
			},
			wantRemainingIDs: []string{"active-1", "active-2"},
		},
		{
			name: "deletes all verifications when all are expired",
			verifications: []*models.Verification{
				newTestVerification("expired-1", "user-1", models.TypeEmailVerification, now.Add(-3*time.Hour)),
				newTestVerification("expired-2", "user-2", models.TypePasswordResetRequest, now.Add(-time.Hour)),
				newTestVerification("expired-3", "user-3", models.TypeMagicLinkSignInRequest, now.Add(-5*time.Minute)),
			},
			wantRemainingIDs: []string{},
		},
		{
			name: "returns an error when the delete fails",
			verifications: []*models.Verification{
				newTestVerification("expired-1", "user-1", models.TypeEmailVerification, now.Add(-2*time.Hour)),
				newTestVerification("active-1", "user-1", models.TypeEmailVerification, now.Add(time.Hour)),
			},
			prepare:          blockVerificationDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"expired-1", "active-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestVerificationDB(t)
			repo := NewBunVerificationRepository(db)

			createTestVerifications(t, ctx, repo, tt.verifications...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.DeleteExpired(ctx)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, verificationIDs(t, ctx, db))
		})
	}
}

func TestBunVerificationRepository_WithTx(t *testing.T) {
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
			db := newTestVerificationDB(t)
			repo := NewBunVerificationRepository(db)

			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			txRepo := repo.WithTx(tx)
			require.NotNil(t, txRepo)

			verification := newTestVerification(
				"verification-tx",
				"user-1",
				models.TypeEmailVerification,
				time.Now().UTC().Add(time.Hour),
			)
			created, err := txRepo.Create(ctx, verification)
			require.NoError(t, err)
			require.Equal(t, verification.ID, created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			got, err := repo.GetByID(ctx, verification.ID)
			if tt.commit {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, verification.ID, got.ID)
				return
			}

			require.Nil(t, got)
			require.ErrorIs(t, err, coreerrors.ErrVerificationNotFound)
		})
	}
}
