package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type BunUserRepository struct {
	db bun.IDB
}

func NewBunUserRepository(db bun.IDB) UserRepository {
	return &BunUserRepository{db: db}
}

func (r *BunUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(user).
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(user).
			WherePK().
			Scan(ctx)
		return err
	})

	return user, err
}

func (r *BunUserRepository) GetAll(ctx context.Context, cursor *string, limit int) ([]models.User, *string, error) {
	query := r.db.NewSelect().
		Model((*models.User)(nil)).
		OrderExpr("id ASC").
		Limit(limit + 1)

	if cursor != nil && strings.TrimSpace(*cursor) != "" {
		query = query.Where("id > ?", strings.TrimSpace(*cursor))
	}

	var users []models.User
	if err := query.Scan(ctx, &users); err != nil {
		return []models.User{}, nil, fmt.Errorf("failed to list users: %w", err)
	}

	if users == nil {
		users = []models.User{}
	}

	if len(users) <= limit {
		return users, nil, nil
	}

	next := users[limit-1].ID
	return users[:limit], &next, nil
}

func (r *BunUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	user := new(models.User)
	err := r.db.NewSelect().
		Model(user).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *BunUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	err := r.db.NewSelect().
		Model(user).
		Where("email = ?", email).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return user, err
}

func (r *BunUserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(user).
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(user).
			WherePK().
			Scan(ctx)
		return err
	})

	return user, err
}

func (r *BunUserRepository) UpdateFields(ctx context.Context, id string, fields map[string]any) error {
	q := r.db.NewUpdate().
		Model(&models.User{}).
		Where("id = ?", id)

	q = util.ApplyFieldUpdates(q, fields)

	_, err := q.Exec(ctx)
	return err
}

func (r *BunUserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model(&models.User{}).
		Where("id = ?", id).
		Exec(ctx)
	return err
}
