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
	failSessionDeleteTrigger = `CREATE TRIGGER fail_session_delete
		BEFORE DELETE ON sessions
		BEGIN
			SELECT RAISE(ABORT, 'session delete blocked');
		END;`

	failSessionUpdateTrigger = `CREATE TRIGGER fail_session_update
		BEFORE UPDATE ON sessions
		BEGIN
			SELECT RAISE(ABORT, 'session update blocked');
		END;`

	deleteSessionAfterInsertTrigger = `CREATE TRIGGER delete_session_after_insert
		AFTER INSERT ON sessions
		BEGIN
			DELETE FROM sessions WHERE id = NEW.id;
		END;`
)

// newTestSessionDB creates an in-memory SQLite database with the session table.
func newTestSessionDB(t *testing.T) bun.IDB {
	t.Helper()

	sqldb, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)

	// In-memory SQLite databases are per connection. A single connection keeps
	// every query, including transactions, on the same database.
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	_, err = db.NewCreateTable().Model(&models.Session{}).Exec(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func newTestSession(id, userID string, createdAt, expiresAt time.Time) *models.Session {
	return &models.Session{
		ID:        id,
		UserID:    userID,
		Token:     "token-" + id,
		ExpiresAt: expiresAt,
		CreatedAt: createdAt,
	}
}

func createTestSessions(t *testing.T, ctx context.Context, repo SessionRepository, sessions ...*models.Session) {
	t.Helper()

	for _, session := range sessions {
		// Clone so parallel subtests never mutate the same fixture model.
		clone := *session
		_, err := repo.Create(ctx, &clone)
		require.NoError(t, err)
	}
}

func execTestSQL(t *testing.T, ctx context.Context, db bun.IDB, query string) {
	t.Helper()

	_, err := db.ExecContext(ctx, query)
	require.NoError(t, err)
}

func blockSessionDeletes(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()
	execTestSQL(t, ctx, db, failSessionDeleteTrigger)
}

func blockSessionUpdates(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()
	execTestSQL(t, ctx, db, failSessionUpdateTrigger)
}

func deleteSessionAfterInsert(t *testing.T, ctx context.Context, db bun.IDB) {
	t.Helper()
	execTestSQL(t, ctx, db, deleteSessionAfterInsertTrigger)
}

func sessionIDs(t *testing.T, ctx context.Context, db bun.IDB, userID *string) []string {
	t.Helper()

	var sessions []*models.Session
	query := db.NewSelect().Model(&sessions)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	require.NoError(t, query.Scan(ctx))

	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ID)
	}
	return ids
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestBunSessionRepository_Getters(t *testing.T) {
	t.Parallel()

	type getter func(SessionRepository, context.Context, string) (*models.Session, error)

	const (
		newerID    = "session-newer"
		newerToken = "token-" + newerID
	)

	tests := []struct {
		name          string
		get           getter
		key           string
		cancelContext bool
		wantID        string
		wantErr       error
	}{
		{
			name:   "GetByID returns the matching session",
			get:    SessionRepository.GetByID,
			key:    newerID,
			wantID: newerID,
		},
		{
			name:    "GetByID returns ErrSessionNotFound for an unknown id",
			get:     SessionRepository.GetByID,
			key:     "missing",
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:          "GetByID returns a query error",
			get:           SessionRepository.GetByID,
			key:           newerID,
			cancelContext: true,
			wantErr:       context.Canceled,
		},
		{
			name:   "GetByToken returns the matching session",
			get:    SessionRepository.GetByToken,
			key:    newerToken,
			wantID: newerID,
		},
		{
			name:    "GetByToken returns ErrSessionNotFound for an unknown token",
			get:     SessionRepository.GetByToken,
			key:     "missing",
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:          "GetByToken returns a query error",
			get:           SessionRepository.GetByToken,
			key:           newerToken,
			cancelContext: true,
			wantErr:       context.Canceled,
		},
		{
			name:   "GetByUserID returns the newest session",
			get:    SessionRepository.GetByUserID,
			key:    "user-1",
			wantID: newerID,
		},
		{
			name:    "GetByUserID returns ErrSessionNotFound when the user has no sessions",
			get:     SessionRepository.GetByUserID,
			key:     "user-without-sessions",
			wantErr: coreerrors.ErrSessionNotFound,
		},
		{
			name:          "GetByUserID returns a query error",
			get:           SessionRepository.GetByUserID,
			key:           "user-1",
			cancelContext: true,
			wantErr:       context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			now := time.Now().UTC()
			createTestSessions(t, ctx, repo,
				newTestSession("session-older", "user-1", now.Add(-time.Hour), now.Add(time.Hour)),
				newTestSession(newerID, "user-1", now, now.Add(time.Hour)),
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

func TestBunSessionRepository_Create(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	newSession := func(id string) *models.Session {
		session := newTestSession(id, "user-1", now, now.Add(time.Hour))
		session.IPAddress = new("127.0.0.1")
		session.UserAgent = new("test-agent")
		return session
	}

	tests := []struct {
		name             string
		seed             func() []*models.Session
		session          func() *models.Session
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantErrIs        error
		wantPersisted    bool
		wantRemainingIDs []string
	}{
		{
			name:             "creates and reloads the session",
			session:          func() *models.Session { return newSession("session-1") },
			wantPersisted:    true,
			wantRemainingIDs: []string{"session-1"},
		},
		{
			name: "returns an error when the insert fails",
			seed: func() []*models.Session {
				return []*models.Session{newSession("session-duplicate")}
			},
			session:          func() *models.Session { return newSession("session-duplicate") },
			wantErr:          true,
			wantRemainingIDs: []string{"session-duplicate"},
		},
		{
			name:             "returns an error when the reload fails after insert",
			session:          func() *models.Session { return newSession("session-reload") },
			prepare:          deleteSessionAfterInsert,
			wantErrIs:        sql.ErrNoRows,
			wantRemainingIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			if tt.seed != nil {
				createTestSessions(t, ctx, repo, tt.seed()...)
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			session := tt.session()
			got, err := repo.Create(ctx, session)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			} else if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Same(t, session, got)

			if tt.wantPersisted {
				reloaded, err := repo.GetByID(ctx, session.ID)
				require.NoError(t, err)
				require.Equal(t, session.ID, reloaded.ID)
				require.Equal(t, session.UserID, reloaded.UserID)
				require.Equal(t, session.Token, reloaded.Token)
				require.WithinDuration(t, session.ExpiresAt, reloaded.ExpiresAt, time.Millisecond)
				require.Equal(t, session.IPAddress, reloaded.IPAddress)
				require.Equal(t, session.UserAgent, reloaded.UserAgent)
			}

			require.ElementsMatch(t, tt.wantRemainingIDs, sessionIDs(t, ctx, db, nil))
		})
	}
}

func TestBunSessionRepository_Update(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	oldExpiry := now.Add(time.Hour)
	newExpiry := now.Add(2 * time.Hour)

	newSeed := func() *models.Session {
		session := newTestSession("session-1", "user-1", now.Add(-time.Hour), oldExpiry)
		session.IPAddress = new("127.0.0.1")
		session.UserAgent = new("old-agent")
		return session
	}
	newUpdate := func() *models.Session {
		session := newTestSession("session-1", "user-1", now.Add(-time.Hour), newExpiry)
		session.Token = "token-updated"
		session.IPAddress = new("10.0.0.1")
		session.UserAgent = new("new-agent")
		return session
	}
	newMissingUpdate := func() *models.Session {
		return newTestSession("session-missing", "user-1", now, newExpiry)
	}

	tests := []struct {
		name          string
		seed          func() *models.Session
		session       func() *models.Session
		prepare       func(*testing.T, context.Context, bun.IDB)
		wantErr       bool
		wantErrIs     error
		wantReload    bool
		wantToken     string
		wantUserAgent *string
	}{
		{
			name:          "updates and reloads the session",
			seed:          newSeed,
			session:       newUpdate,
			wantReload:    true,
			wantToken:     "token-updated",
			wantUserAgent: new("new-agent"),
		},
		{
			name:          "returns an error when the update fails",
			seed:          newSeed,
			session:       newUpdate,
			prepare:       blockSessionUpdates,
			wantErr:       true,
			wantReload:    true,
			wantToken:     "token-session-1",
			wantUserAgent: new("old-agent"),
		},
		{
			name:      "returns an error when the session does not exist",
			session:   newMissingUpdate,
			wantErrIs: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			if tt.seed != nil {
				createTestSessions(t, ctx, repo, tt.seed())
			}
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			session := tt.session()
			got, err := repo.Update(ctx, session)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			} else if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Same(t, session, got)

			if tt.wantReload {
				reloaded, err := repo.GetByID(ctx, session.ID)
				require.NoError(t, err)
				require.Equal(t, tt.wantToken, reloaded.Token)
				require.Equal(t, tt.wantUserAgent, reloaded.UserAgent)
			}
		})
	}
}

func TestBunSessionRepository_Delete(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name             string
		seed             func() []*models.Session
		id               string
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name: "deletes the requested session",
			seed: func() []*models.Session {
				return []*models.Session{
					newTestSession("session-1", "user-1", now, now.Add(time.Hour)),
					newTestSession("session-2", "user-1", now, now.Add(time.Hour)),
				}
			},
			id:               "session-1",
			wantRemainingIDs: []string{"session-2"},
		},
		{
			name: "is a no-op when the session does not exist",
			seed: func() []*models.Session {
				return []*models.Session{newTestSession("session-1", "user-1", now, now.Add(time.Hour))}
			},
			id:               "missing",
			wantRemainingIDs: []string{"session-1"},
		},
		{
			name: "returns an error when the delete fails",
			seed: func() []*models.Session {
				return []*models.Session{
					newTestSession("session-1", "user-1", now, now.Add(time.Hour)),
					newTestSession("session-2", "user-1", now, now.Add(time.Hour)),
				}
			},
			id:               "session-1",
			prepare:          blockSessionDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"session-1", "session-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			createTestSessions(t, ctx, repo, tt.seed()...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.Delete(ctx, tt.id)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, sessionIDs(t, ctx, db, nil))
		})
	}
}

func TestBunSessionRepository_DeleteByUserID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	newSeed := func() []*models.Session {
		return []*models.Session{
			newTestSession("user-1-a", "user-1", now, now.Add(time.Hour)),
			newTestSession("user-1-b", "user-1", now, now.Add(time.Hour)),
			newTestSession("user-2-a", "user-2", now, now.Add(time.Hour)),
		}
	}

	tests := []struct {
		name             string
		seed             func() []*models.Session
		userID           string
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name:             "deletes all sessions for the user",
			seed:             newSeed,
			userID:           "user-1",
			wantRemainingIDs: []string{"user-2-a"},
		},
		{
			name:             "is a no-op when the user has no sessions",
			seed:             newSeed,
			userID:           "user-3",
			wantRemainingIDs: []string{"user-1-a", "user-1-b", "user-2-a"},
		},
		{
			name:             "returns an error when the delete fails",
			seed:             newSeed,
			userID:           "user-1",
			prepare:          blockSessionDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"user-1-a", "user-1-b", "user-2-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			createTestSessions(t, ctx, repo, tt.seed()...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.DeleteByUserID(ctx, tt.userID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, sessionIDs(t, ctx, db, nil))
		})
	}
}

func TestBunSessionRepository_DeleteExpired(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name             string
		sessions         []*models.Session
		prepare          func(*testing.T, context.Context, bun.IDB)
		wantErr          bool
		wantRemainingIDs []string
	}{
		{
			name: "deletes only expired sessions",
			sessions: []*models.Session{
				newTestSession("expired-1", "user-1", now.Add(-2*time.Hour), now.Add(-time.Hour)),
				newTestSession("expired-2", "user-2", now.Add(-2*time.Hour), now.Add(-30*time.Minute)),
				newTestSession("active-1", "user-1", now.Add(-time.Hour), now.Add(time.Hour)),
				newTestSession("active-2", "user-3", now.Add(-time.Hour), now.Add(2*time.Hour)),
			},
			wantRemainingIDs: []string{"active-1", "active-2"},
		},
		{
			name: "keeps all sessions when none are expired",
			sessions: []*models.Session{
				newTestSession("active-1", "user-1", now.Add(-time.Hour), now.Add(time.Hour)),
				newTestSession("active-2", "user-2", now.Add(-time.Hour), now.Add(2*time.Hour)),
			},
			wantRemainingIDs: []string{"active-1", "active-2"},
		},
		{
			name: "deletes all sessions when all are expired",
			sessions: []*models.Session{
				newTestSession("expired-1", "user-1", now.Add(-3*time.Hour), now.Add(-time.Hour)),
				newTestSession("expired-2", "user-2", now.Add(-2*time.Hour), now.Add(-30*time.Minute)),
				newTestSession("expired-3", "user-3", now.Add(-time.Hour), now.Add(-5*time.Minute)),
			},
			wantRemainingIDs: []string{},
		},
		{
			name: "returns an error when the delete fails",
			sessions: []*models.Session{
				newTestSession("expired-1", "user-1", now.Add(-2*time.Hour), now.Add(-time.Hour)),
				newTestSession("active-1", "user-1", now.Add(-time.Hour), now.Add(time.Hour)),
			},
			prepare:          blockSessionDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"expired-1", "active-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			createTestSessions(t, ctx, repo, tt.sessions...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			err := repo.DeleteExpired(ctx)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, sessionIDs(t, ctx, db, nil))
		})
	}
}

func TestBunSessionRepository_DeleteOldestByUserID(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	sessionAt := func(id, userID string, offset time.Duration) *models.Session {
		return newTestSession(id, userID, now.Add(offset), now.Add(time.Hour))
	}

	tests := []struct {
		name             string
		sessions         []*models.Session
		userID           string
		keep             int
		otherUserID      string
		prepare          func(*testing.T, context.Context, bun.IDB)
		cancelContext    bool
		wantErr          bool
		wantRemainingIDs []string
		wantOtherUserIDs []string
	}{
		{
			name: "deletes oldest sessions and keeps newest",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
				sessionAt("sess-3", "user-1", 3*time.Minute),
				sessionAt("sess-4", "user-1", 4*time.Minute),
				sessionAt("sess-5", "user-1", 5*time.Minute),
			},
			userID:           "user-1",
			keep:             2,
			wantRemainingIDs: []string{"sess-4", "sess-5"},
		},
		{
			name: "keeps all sessions when below the limit",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
			},
			userID:           "user-1",
			keep:             5,
			wantRemainingIDs: []string{"sess-1", "sess-2"},
		},
		{
			name: "deletes all sessions when keep is zero",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
				sessionAt("sess-3", "user-1", 3*time.Minute),
			},
			userID:           "user-1",
			keep:             0,
			wantRemainingIDs: []string{},
		},
		{
			name: "deletes all sessions when keep is negative",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
			},
			userID:           "user-1",
			keep:             -1,
			wantRemainingIDs: []string{},
		},
		{
			name: "is a no-op when the user has no sessions",
			sessions: []*models.Session{
				sessionAt("other-1", "user-2", 1*time.Minute),
			},
			userID:           "user-1",
			keep:             5,
			otherUserID:      "user-2",
			wantRemainingIDs: []string{},
			wantOtherUserIDs: []string{"other-1"},
		},
		{
			name: "only deletes the requested user's sessions",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
				sessionAt("sess-3", "user-1", 3*time.Minute),
				sessionAt("other-1", "user-2", 4*time.Minute),
				sessionAt("other-2", "user-2", 5*time.Minute),
				sessionAt("other-3", "user-2", 6*time.Minute),
			},
			userID:           "user-1",
			keep:             1,
			otherUserID:      "user-2",
			wantRemainingIDs: []string{"sess-3"},
			wantOtherUserIDs: []string{"other-1", "other-2", "other-3"},
		},
		{
			name: "returns an error when the session query fails",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
			},
			userID:           "user-1",
			keep:             1,
			cancelContext:    true,
			wantErr:          true,
			wantRemainingIDs: []string{"sess-1", "sess-2"},
		},
		{
			name: "returns an error when the delete fails",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
				sessionAt("sess-3", "user-1", 3*time.Minute),
			},
			userID:           "user-1",
			keep:             1,
			prepare:          blockSessionDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"sess-1", "sess-2", "sess-3"},
		},
		{
			name: "returns an error when deleting all fails",
			sessions: []*models.Session{
				sessionAt("sess-1", "user-1", 1*time.Minute),
				sessionAt("sess-2", "user-1", 2*time.Minute),
			},
			userID:           "user-1",
			keep:             0,
			prepare:          blockSessionDeletes,
			wantErr:          true,
			wantRemainingIDs: []string{"sess-1", "sess-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			createTestSessions(t, ctx, repo, tt.sessions...)
			if tt.prepare != nil {
				tt.prepare(t, ctx, db)
			}

			callCtx := ctx
			if tt.cancelContext {
				callCtx = cancelledContext()
			}

			err := repo.DeleteOldestByUserID(callCtx, tt.userID, tt.keep)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.ElementsMatch(t, tt.wantRemainingIDs, sessionIDs(t, ctx, db, &tt.userID))

			if tt.otherUserID != "" {
				require.ElementsMatch(t, tt.wantOtherUserIDs, sessionIDs(t, ctx, db, &tt.otherUserID))
			}
		})
	}
}

func TestBunSessionRepository_GetDistinctUserIDs(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	newSessions := func() []*models.Session {
		return []*models.Session{
			newTestSession("sess-1", "user-1", now, now.Add(time.Hour)),
			newTestSession("sess-2", "user-1", now, now.Add(time.Hour)),
			newTestSession("sess-3", "user-2", now, now.Add(time.Hour)),
			newTestSession("sess-4", "user-3", now, now.Add(time.Hour)),
			newTestSession("sess-5", "user-3", now, now.Add(time.Hour)),
		}
	}

	tests := []struct {
		name          string
		sessions      func() []*models.Session
		cancelContext bool
		wantUserIDs   []string
		wantErrIs     error
	}{
		{
			name:        "returns each user id once",
			sessions:    newSessions,
			wantUserIDs: []string{"user-1", "user-2", "user-3"},
		},
		{
			name:        "returns an empty list when there are no sessions",
			wantUserIDs: []string{},
		},
		{
			name:          "returns an error when the query fails",
			sessions:      newSessions,
			cancelContext: true,
			wantErrIs:     context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			if tt.sessions != nil {
				createTestSessions(t, ctx, repo, tt.sessions()...)
			}
			if tt.cancelContext {
				ctx = cancelledContext()
			}

			got, err := repo.GetDistinctUserIDs(ctx)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Empty(t, got)
				return
			}

			require.NoError(t, err)
			require.ElementsMatch(t, tt.wantUserIDs, got)
		})
	}
}

func TestBunSessionRepository_WithTx(t *testing.T) {
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
			db := newTestSessionDB(t)
			repo := NewBunSessionRepository(db)

			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			txRepo := repo.WithTx(tx)
			require.NotNil(t, txRepo)

			now := time.Now().UTC()
			session := newTestSession("session-tx", "user-1", now, now.Add(time.Hour))
			created, err := txRepo.Create(ctx, session)
			require.NoError(t, err)
			require.Equal(t, session.ID, created.ID)

			if tt.commit {
				require.NoError(t, tx.Commit())
			} else {
				require.NoError(t, tx.Rollback())
			}

			got, err := repo.GetByID(ctx, session.ID)
			if tt.commit {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, session.ID, got.ID)
				return
			}

			require.Nil(t, got)
			require.ErrorIs(t, err, coreerrors.ErrSessionNotFound)
		})
	}
}
