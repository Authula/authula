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

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/internal/sqlitedsn"
	"github.com/Authula/authula/models"
)

func newTestVerificationDB(t *testing.T) bun.IDB {
	t.Helper()

	sqldb, err := sql.Open("sqlite", sqlitedsn.Build(":memory:"))
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.NewCreateTable().Model(&models.Verification{}).Exec(context.Background()); err != nil {
		t.Fatalf("failed to create Verification table: %v", err)
	}

	return db
}

func TestBunVerificationRepository_GetByToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantFound bool
	}{
		{name: "returns the verification for a known token", token: "hashed-known", wantFound: true},
		{name: "reports not found for an unknown token", token: "hashed-missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := assert.New(t)
			must := require.New(t)

			repo := NewBunVerificationRepository(newTestVerificationDB(t))
			ctx := context.Background()

			userID := "user1"
			verification := &models.Verification{
				ID:         "verification-1",
				UserID:     &userID,
				Identifier: "user@example.com",
				Token:      "hashed-known",
				Type:       models.TypeEmailVerification,
				ExpiresAt:  time.Now().UTC().Add(time.Hour),
			}
			_, err := repo.Create(ctx, verification)
			must.NoError(err)

			got, err := repo.GetByToken(ctx, tt.token)

			if tt.wantFound {
				must.NoError(err)
				must.NotNil(got)
				is.Equal("verification-1", got.ID)
				is.Equal("hashed-known", got.Token)
				return
			}

			is.Nil(got)
			is.Error(err, "a missing verification must not be reported as (nil, nil)")
			is.ErrorIs(err, coreerrors.ErrNotFound)
		})
	}
}
