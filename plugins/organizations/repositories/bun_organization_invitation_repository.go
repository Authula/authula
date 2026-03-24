package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type BunOrganizationInvitationRepository struct {
	db    bun.IDB
	hooks OrganizationInvitationHookExecutor
}

func NewBunOrganizationInvitationRepository(db bun.IDB, hooks ...OrganizationInvitationHookExecutor) OrganizationInvitationRepository {
	var hook OrganizationInvitationHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &BunOrganizationInvitationRepository{db: db, hooks: hook}
}

func (r *BunOrganizationInvitationRepository) Create(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if r.hooks != nil {
			if err := r.hooks.BeforeCreateOrganizationInvitation(invitation); err != nil {
				return err
			}
		}

		_, err := tx.NewInsert().Model(invitation).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(invitation).WherePK().Scan(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterCreateOrganizationInvitation(*invitation); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return invitation, nil
}

func (r *BunOrganizationInvitationRepository) GetByID(ctx context.Context, invitationID string) (*types.OrganizationInvitation, error) {
	invitation := new(types.OrganizationInvitation)
	err := r.db.NewSelect().Model(invitation).Where("id = ?", invitationID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return invitation, err
}

func (r *BunOrganizationInvitationRepository) GetByOrganizationIDAndEmail(ctx context.Context, organizationID, email string, status ...types.OrganizationInvitationStatus) (*types.OrganizationInvitation, error) {
	invitation := new(types.OrganizationInvitation)
	query := r.db.NewSelect().Model(invitation).
		Where("organization_id = ? AND email = ?", organizationID, email).
		OrderExpr("created_at DESC, id DESC")
	statusValues := make([]string, 0, len(status))
	for _, invitationStatus := range status {
		if invitationStatus == "" {
			continue
		}
		statusValues = append(statusValues, string(invitationStatus))
	}
	if len(statusValues) > 0 {
		query = query.Where("status IN (?)", bun.List(statusValues))
	}
	err := query.Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return invitation, err
}

func (r *BunOrganizationInvitationRepository) GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationInvitation, error) {
	invitations := make([]types.OrganizationInvitation, 0)
	err := r.db.NewSelect().Model(&invitations).
		Where("organization_id = ?", organizationID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationInvitation{}, nil
	}
	return invitations, err
}

func (r *BunOrganizationInvitationRepository) GetAllPendingByEmail(ctx context.Context, email string) ([]types.OrganizationInvitation, error) {
	invites := make([]types.OrganizationInvitation, 0)
	err := r.db.NewSelect().Model(&invites).
		Where("email = ? AND status = ? AND expires_at > ?", email, types.OrganizationInvitationStatusPending, time.Now().UTC()).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationInvitation{}, nil
	}
	return invites, err
}

func (r *BunOrganizationInvitationRepository) Update(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if r.hooks != nil {
			if err := r.hooks.BeforeUpdateOrganizationInvitation(invitation); err != nil {
				return err
			}
		}

		_, err := tx.NewUpdate().Model(invitation).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(invitation).WherePK().Scan(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterUpdateOrganizationInvitation(*invitation); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return invitation, nil
}

func (r *BunOrganizationInvitationRepository) CountByOrganizationIDAndEmail(ctx context.Context, organizationID, email string) (int, error) {
	return r.db.NewSelect().Model((*types.OrganizationInvitation)(nil)).Where("organization_id = ? AND email = ?", organizationID, email).Count(ctx)
}

func (r *BunOrganizationInvitationRepository) WithTx(tx bun.IDB) OrganizationInvitationRepository {
	return &BunOrganizationInvitationRepository{db: tx, hooks: r.hooks}
}
