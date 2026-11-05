package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	corerepos "github.com/Authula/authula/core/repositories"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
)

func newUserRepositoryTestDB(t *testing.T) *bun.DB {
	t.Helper()

	db := internaltests.NewSQLiteIntegrationDB(t)
	_, err := db.NewCreateTable().Model((*models.User)(nil)).IfNotExists().Exec(context.Background())
	require.NoError(t, err)
	return db
}

func newAccountRepositoryTestDB(t *testing.T) *bun.DB {
	t.Helper()

	db := newUserRepositoryTestDB(t)
	_, err := db.NewCreateTable().Model((*models.Account)(nil)).IfNotExists().Exec(context.Background())
	require.NoError(t, err)

	_, err = corerepos.NewBunUserRepository(db).Create(context.Background(), &models.User{
		ID:    "user-1",
		Name:  "User",
		Email: "user@example.com",
	})
	require.NoError(t, err)
	return db
}

func TestBunUserRepository_NormalizesEmails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(context.Context, *bun.DB, corerepos.UserRepository) (string, error)
		want string
	}{
		{
			name: "create stores a normalized email",
			run: func(ctx context.Context, db *bun.DB, repo corerepos.UserRepository) (string, error) {
				_, err := repo.Create(ctx, &models.User{
					ID:    "user-1",
					Name:  "User",
					Email: "  User@Example.COM  ",
				})
				if err != nil {
					return "", err
				}

				raw := new(models.User)
				if err := db.NewSelect().Model(raw).Where("id = ?", "user-1").Scan(ctx); err != nil {
					return "", err
				}
				return raw.Email, nil
			},
			want: "user@example.com",
		},
		{
			name: "get by email normalizes the lookup input",
			run: func(ctx context.Context, _ *bun.DB, repo corerepos.UserRepository) (string, error) {
				if _, err := repo.Create(ctx, &models.User{ID: "user-1", Name: "User", Email: "user@example.com"}); err != nil {
					return "", err
				}
				user, err := repo.GetByEmail(ctx, "  USER@EXAMPLE.COM  ")
				if user == nil {
					return "", err
				}
				return user.Email, err
			},
			want: "user@example.com",
		},
		{
			name: "update normalizes the email",
			run: func(ctx context.Context, _ *bun.DB, repo corerepos.UserRepository) (string, error) {
				user, err := repo.Create(ctx, &models.User{ID: "user-1", Name: "User", Email: "user@example.com"})
				if err != nil {
					return "", err
				}
				user.Email = "  Changed@Example.COM  "
				updated, err := repo.Update(ctx, user)
				if updated == nil {
					return "", err
				}
				return updated.Email, err
			},
			want: "changed@example.com",
		},
		{
			name: "update fields normalizes the email",
			run: func(ctx context.Context, _ *bun.DB, repo corerepos.UserRepository) (string, error) {
				if _, err := repo.Create(ctx, &models.User{ID: "user-1", Name: "User", Email: "user@example.com"}); err != nil {
					return "", err
				}
				if err := repo.UpdateFields(ctx, "user-1", map[string]any{"email": "  Fields@Example.COM  "}); err != nil {
					return "", err
				}
				user, err := repo.GetByID(ctx, "user-1")
				if user == nil {
					return "", err
				}
				return user.Email, err
			},
			want: "fields@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newUserRepositoryTestDB(t)
			repo := corerepos.NewBunUserRepository(db)

			got, err := tt.run(ctx, db, repo)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBunAccountRepository_NormalizesEmailProviderAccountID(t *testing.T) {
	t.Parallel()

	emailProvider := models.AuthProviderEmail.String()

	tests := []struct {
		name string
		run  func(context.Context, corerepos.AccountRepository) (string, error)
		want string
	}{
		{
			name: "create normalizes the account id",
			run: func(ctx context.Context, repo corerepos.AccountRepository) (string, error) {
				account, err := repo.Create(ctx, &models.Account{
					ID:         "account-1",
					UserID:     "user-1",
					ProviderID: emailProvider,
					AccountID:  "  User@Example.COM  ",
				})
				if account == nil {
					return "", err
				}
				return account.AccountID, err
			},
			want: "user@example.com",
		},
		{
			name: "lookup normalizes the account id input",
			run: func(ctx context.Context, repo corerepos.AccountRepository) (string, error) {
				if _, err := repo.Create(ctx, &models.Account{
					ID:         "account-1",
					UserID:     "user-1",
					ProviderID: emailProvider,
					AccountID:  "user@example.com",
				}); err != nil {
					return "", err
				}
				account, err := repo.GetByProviderAndAccountID(ctx, emailProvider, "USER@EXAMPLE.COM")
				if account == nil {
					return "", err
				}
				return account.AccountID, err
			},
			want: "user@example.com",
		},
		{
			name: "update normalizes the account id",
			run: func(ctx context.Context, repo corerepos.AccountRepository) (string, error) {
				account, err := repo.Create(ctx, &models.Account{
					ID:         "account-1",
					UserID:     "user-1",
					ProviderID: emailProvider,
					AccountID:  "user@example.com",
				})
				if err != nil {
					return "", err
				}
				account.AccountID = "  Changed@Example.COM  "
				updated, err := repo.Update(ctx, account)
				if updated == nil {
					return "", err
				}
				return updated.AccountID, err
			},
			want: "changed@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			db := newAccountRepositoryTestDB(t)
			repo := corerepos.NewBunAccountRepository(db)

			got, err := tt.run(ctx, repo)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
