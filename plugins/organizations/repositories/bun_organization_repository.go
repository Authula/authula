package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type BunOrganizationRepository struct {
	db    bun.IDB
	hooks OrganizationHookExecutor
}

func NewBunOrganizationRepository(db bun.IDB, hooks ...OrganizationHookExecutor) OrganizationRepository {
	var hook OrganizationHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &BunOrganizationRepository{db: db, hooks: hook}
}

func (r *BunOrganizationRepository) Create(ctx context.Context, organization *types.Organization) (*types.Organization, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if r.hooks != nil {
			if err := r.hooks.BeforeCreateOrganization(organization); err != nil {
				return err
			}
		}

		_, err := tx.NewInsert().Model(organization).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(organization).WherePK().Scan(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterCreateOrganization(*organization); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (r *BunOrganizationRepository) GetByID(ctx context.Context, organizationID string) (*types.Organization, error) {
	organization := new(types.Organization)
	err := r.db.NewSelect().Model(organization).Where("id = ?", organizationID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return organization, err
}

func (r *BunOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*types.Organization, error) {
	organization := new(types.Organization)
	err := r.db.NewSelect().Model(organization).Where("slug = ?", slug).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return organization, err
}

func (r *BunOrganizationRepository) GetAllByOwnerID(ctx context.Context, ownerID string) ([]types.Organization, error) {
	organizations := make([]types.Organization, 0)
	err := r.db.NewSelect().Model(&organizations).Where("owner_id = ?", ownerID).Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.Organization{}, nil
	}
	return organizations, err
}

func (r *BunOrganizationRepository) Update(ctx context.Context, organization *types.Organization) (*types.Organization, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if r.hooks != nil {
			if err := r.hooks.BeforeUpdateOrganization(organization); err != nil {
				return err
			}
		}

		_, err := tx.NewUpdate().Model(organization).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(organization).WherePK().Scan(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterUpdateOrganization(*organization); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return organization, nil
}

func (r *BunOrganizationRepository) Delete(ctx context.Context, organizationID string) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		orgRepo := r.WithTx(tx)
		organization, err := orgRepo.GetByID(ctx, organizationID)
		if err != nil {
			return err
		}
		if organization == nil {
			return nil
		}

		if r.hooks != nil {
			if err := r.hooks.BeforeDeleteOrganization(organization); err != nil {
				return err
			}
		}

		if _, err := tx.NewDelete().Model(&types.Organization{}).Where("id = ?", organizationID).Exec(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterDeleteOrganization(*organization); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *BunOrganizationRepository) WithTx(tx bun.IDB) OrganizationRepository {
	return &BunOrganizationRepository{db: tx, hooks: r.hooks}
}
