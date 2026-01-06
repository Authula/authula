package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type BunSessionRepository struct {
	db bun.IDB
}

func NewBunSessionRepository(db bun.IDB) SessionRepository {
	return &BunSessionRepository{db: db}
}

func (r *BunSessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	s := new(models.Session)
	err := r.db.NewSelect().Model(s).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *BunSessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	s := new(models.Session)
	err := r.db.NewSelect().Model(s).Where("token = ?", token).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *BunSessionRepository) GetByUserID(ctx context.Context, userID string) (*models.Session, error) {
	s := new(models.Session)
	err := r.db.NewSelect().Model(s).Where("user_id = ?", userID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *BunSessionRepository) Create(ctx context.Context, session *models.Session) (*models.Session, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(session).
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(session).
			WherePK().
			Scan(ctx)
		return err
	})

	return session, err
}

func (r *BunSessionRepository) Update(ctx context.Context, session *models.Session) (*models.Session, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(session).
			WherePK().
			Exec(ctx)
		if err != nil {
			return err
		}

		err = tx.NewSelect().
			Model(session).
			WherePK().
			Scan(ctx)
		return err
	})

	return session, err
}

func (r *BunSessionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model(&models.Session{}).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *BunSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.NewDelete().Model(&models.Session{}).Where("user_id = ?", userID).Exec(ctx)
	return err
}

func (r *BunSessionRepository) WithTx(tx bun.IDB) SessionRepository {
	return &BunSessionRepository{db: tx}
}
