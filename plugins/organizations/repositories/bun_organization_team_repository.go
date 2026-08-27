package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type BunOrganizationTeamRepository struct {
	db bun.IDB
}

func NewBunOrganizationTeamRepository(db bun.IDB) OrganizationTeamRepository {
	return &BunOrganizationTeamRepository{db: db}
}

func (r *BunOrganizationTeamRepository) Create(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(team).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(team).WherePK().Scan(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (r *BunOrganizationTeamRepository) GetByID(ctx context.Context, teamID string) (*types.OrganizationTeam, error) {
	team := new(types.OrganizationTeam)
	err := r.db.NewSelect().Model(team).Where("id = ?", teamID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return team, err
}

func (r *BunOrganizationTeamRepository) GetByOrganizationIDAndSlug(ctx context.Context, organizationID, slug string) (*types.OrganizationTeam, error) {
	team := new(types.OrganizationTeam)
	err := r.db.NewSelect().Model(team).
		Where("organization_id = ? AND slug = ?", organizationID, slug).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return team, err
}

func (r *BunOrganizationTeamRepository) ListAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationTeam, int, error) {
	teams := make([]types.OrganizationTeam, 0)
	limit = pageLimit(limit)
	total, err := r.db.NewSelect().Model(&teams).
		Where("organization_id = ?", organizationID).
		OrderExpr("created_at DESC, id DESC").
		Offset(pageOffset(page, limit)).Limit(limit).
		ScanAndCount(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationTeam{}, total, nil
	}
	return teams, total, err
}

func (r *BunOrganizationTeamRepository) GetAllByOrganizationID(ctx context.Context, organizationID string) ([]types.OrganizationTeam, error) {
	teams := make([]types.OrganizationTeam, 0)
	err := r.db.NewSelect().Model(&teams).
		Where("organization_id = ?", organizationID).
		OrderExpr("created_at DESC, id DESC").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationTeam{}, nil
	}
	return teams, err
}

func (r *BunOrganizationTeamRepository) Update(ctx context.Context, team *types.OrganizationTeam) (*types.OrganizationTeam, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model(team).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(team).WherePK().Scan(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (r *BunOrganizationTeamRepository) Delete(ctx context.Context, teamID string) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model(&types.OrganizationTeam{}).Where("id = ?", teamID).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (r *BunOrganizationTeamRepository) WithTx(tx bun.IDB) OrganizationTeamRepository {
	return &BunOrganizationTeamRepository{db: tx}
}
