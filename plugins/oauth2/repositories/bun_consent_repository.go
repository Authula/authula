package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

type BunConsentRepository struct {
	db bun.IDB
}

func NewBunConsentRepository(db bun.IDB) ConsentRepository {
	return &BunConsentRepository{db: db}
}

func (r *BunConsentRepository) Create(ctx context.Context, consent *types.OAuthConsent) (*types.OAuthConsent, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(consent).Exec(ctx); err != nil {
			return err
		}
		return tx.NewSelect().Model(consent).WherePK().Scan(ctx)
	})
	if err != nil {
		return nil, err
	}

	return consent, nil
}

func (r *BunConsentRepository) GetByID(ctx context.Context, id string) (*types.OAuthConsent, error) {
	consent := new(types.OAuthConsent)
	if err := r.db.NewSelect().Model(consent).Where("id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return consent, nil
}

func (r *BunConsentRepository) GetActiveByUserAndClient(ctx context.Context, userID string, clientID string) (*types.OAuthConsent, error) {
	consent := new(types.OAuthConsent)
	err := r.db.NewSelect().
		Model(consent).
		Where("user_id = ? AND client_id = ? AND revoked_at IS NULL", userID, clientID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return consent, nil
}

func (r *BunConsentRepository) ListActiveByUser(ctx context.Context, userID string) ([]*types.OAuthConsent, error) {
	consents := make([]*types.OAuthConsent, 0)
	if err := r.db.NewSelect().
		Model(&consents).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		OrderExpr("created_at DESC, id DESC").
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*types.OAuthConsent{}, nil
		}
		return nil, err
	}

	return consents, nil
}

func (r *BunConsentRepository) Revoke(ctx context.Context, id string, revokedAt time.Time) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthConsent)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Set("updated_at = ?", revokedAt.UTC()).
		Where("id = ? AND revoked_at IS NULL", id).
		Exec(ctx)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *BunConsentRepository) RevokeByUserAndClient(ctx context.Context, userID string, clientID string, revokedAt time.Time) (int64, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthConsent)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Set("updated_at = ?", revokedAt.UTC()).
		Where("user_id = ? AND client_id = ? AND revoked_at IS NULL", userID, clientID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *BunConsentRepository) RevokeByClient(ctx context.Context, clientID string, revokedAt time.Time) (int64, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthConsent)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Set("updated_at = ?", revokedAt.UTC()).
		Where("client_id = ? AND revoked_at IS NULL", clientID).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *BunConsentRepository) WithTx(tx bun.IDB) ConsentRepository {
	return &BunConsentRepository{db: tx}
}
