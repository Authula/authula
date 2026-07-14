package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type BunOrganizationTeamMemberRepository struct {
	db bun.IDB
}

func NewBunOrganizationTeamMemberRepository(db bun.IDB) OrganizationTeamMemberRepository {
	return &BunOrganizationTeamMemberRepository{db: db}
}

func (r *BunOrganizationTeamMemberRepository) Create(ctx context.Context, teamMember *types.OrganizationTeamMember) (*types.OrganizationTeamMember, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(teamMember).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(teamMember).WherePK().Scan(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return teamMember, nil
}

func (r *BunOrganizationTeamMemberRepository) GetByID(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMember, error) {
	teamMember := new(types.OrganizationTeamMember)
	err := r.db.NewSelect().Model(teamMember).Where("id = ?", teamMemberID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return teamMember, err
}

func (r *BunOrganizationTeamMemberRepository) GetByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) (*types.OrganizationTeamMember, error) {
	teamMember := new(types.OrganizationTeamMember)
	err := r.db.NewSelect().Model(teamMember).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return teamMember, err
}

func (r *BunOrganizationTeamMemberRepository) GetAllByTeamID(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMember, error) {
	teamMembers := make([]types.OrganizationTeamMember, 0)
	err := r.db.NewSelect().Model(&teamMembers).
		Where("team_id = ?", teamID).
		OrderExpr("created_at DESC, id DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationTeamMember{}, nil
	}
	return teamMembers, err
}

func (r *BunOrganizationTeamMemberRepository) DeleteByTeamIDAndMemberID(ctx context.Context, teamID, memberID string) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().Model(&types.OrganizationTeamMember{}).Where("team_id = ? AND member_id = ?", teamID, memberID).Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *BunOrganizationTeamMemberRepository) WithTx(tx bun.IDB) OrganizationTeamMemberRepository {
	return &BunOrganizationTeamMemberRepository{db: tx}
}
