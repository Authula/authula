package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/internal/util"
	"github.com/GoBetterAuth/go-better-auth/models"
)

type BunUserRepository struct {
	db bun.IDB
}

func NewBunUserRepository(db bun.IDB) UserRepository {
	return &BunUserRepository{db: db}
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

func (r *BunUserRepository) WithTx(tx bun.IDB) UserRepository {
	return &BunUserRepository{db: tx}
}
