package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
			return fmt.Errorf("insert oauth consent: %w", err)
		}
		if err := tx.NewSelect().Model(consent).WherePK().Scan(ctx); err != nil {
			return fmt.Errorf("reload created oauth consent: %w", err)
		}
		return nil
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
			return nil, fmt.Errorf("%w: id=%s", ErrConsentNotFound, id)
		}
		return nil, fmt.Errorf("get oauth consent by id: %w", err)
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
			return nil, fmt.Errorf("%w: user_id=%s client_id=%s", ErrConsentNotFound, userID, clientID)
		}
		return nil, fmt.Errorf("get active oauth consent: %w", err)
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
		return nil, fmt.Errorf("list active oauth consents: %w", err)
	}

	return consents, nil
}

func (r *BunConsentRepository) UpdateScope(ctx context.Context, id string, scope string, updatedAt time.Time) (*types.OAuthConsent, error) {
	consent := new(types.OAuthConsent)
	updated := false
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.NewUpdate().
			Model((*types.OAuthConsent)(nil)).
			Set("scope = ?", scope).
			Set("updated_at = ?", updatedAt.UTC()).
			Where("id = ? AND revoked_at IS NULL", id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("update oauth consent scope: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read affected rows for oauth consent scope update: %w", err)
		}
		if affected == 0 {
			return nil
		}
		updated = true
		if err := tx.NewSelect().Model(consent).Where("id = ?", id).Scan(ctx); err != nil {
			return fmt.Errorf("reload updated oauth consent: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, fmt.Errorf("%w: id=%s", ErrConsentNotFound, id)
	}
	return consent, nil
}

func (r *BunConsentRepository) Revoke(ctx context.Context, id string, revokedAt time.Time) (bool, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthConsent)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Set("updated_at = ?", revokedAt.UTC()).
		Where("id = ? AND revoked_at IS NULL", id).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("revoke oauth consent: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected rows for oauth consent revoke: %w", err)
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
		return 0, fmt.Errorf("revoke oauth consents by user and client: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows for oauth consent revoke by user and client: %w", err)
	}

	return affected, nil
}

func (r *BunConsentRepository) RevokeByClient(ctx context.Context, clientID string, revokedAt time.Time) (int64, error) {
	result, err := r.db.NewUpdate().
		Model((*types.OAuthConsent)(nil)).
		Set("revoked_at = ?", revokedAt.UTC()).
		Set("updated_at = ?", revokedAt.UTC()).
		Where("client_id = ? AND revoked_at IS NULL", clientID).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("revoke oauth consents by client: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows for oauth consent revoke by client: %w", err)
	}

	return affected, nil
}

func (r *BunConsentRepository) WithTx(tx bun.IDB) ConsentRepository {
	return &BunConsentRepository{db: tx}
}
