package repositories

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type BunOrganizationMemberRepository struct {
	db    bun.IDB
	hooks OrganizationMemberHookExecutor
}

func NewBunOrganizationMemberRepository(db bun.IDB, hooks ...OrganizationMemberHookExecutor) OrganizationMemberRepository {
	var hook OrganizationMemberHookExecutor
	if len(hooks) > 0 {
		hook = hooks[0]
	}
	return &BunOrganizationMemberRepository{db: db, hooks: hook}
}

func (r *BunOrganizationMemberRepository) Create(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if r.hooks != nil {
			if err := r.hooks.BeforeCreateOrganizationMember(member); err != nil {
				return err
			}
		}

		_, err := tx.NewInsert().Model(member).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(member).WherePK().Scan(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterCreateOrganizationMember(*member); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (r *BunOrganizationMemberRepository) CountByOrganizationID(ctx context.Context, organizationID string) (int, error) {
	return r.db.NewSelect().Model((*types.OrganizationMember)(nil)).Where("organization_id = ?", organizationID).Count(ctx)
}

func (r *BunOrganizationMemberRepository) GetByID(ctx context.Context, memberID string) (*types.OrganizationMember, error) {
	member := new(types.OrganizationMember)
	err := r.db.NewSelect().Model(member).Where("id = ?", memberID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return member, err
}

func (r *BunOrganizationMemberRepository) GetAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMember, error) {
	members := make([]types.OrganizationMember, 0)
	err := r.db.NewSelect().Model(&members).
		Where("organization_id = ?", organizationID).
		OrderExpr("created_at DESC, id DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationMember{}, nil
	}
	return members, err
}

func (r *BunOrganizationMemberRepository) GetAllByUserID(ctx context.Context, userID string) ([]types.OrganizationMember, error) {
	members := make([]types.OrganizationMember, 0)
	err := r.db.NewSelect().Model(&members).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationMember{}, nil
	}
	return members, err
}

func (r *BunOrganizationMemberRepository) GetByOrganizationIDAndUserID(ctx context.Context, organizationID string, userID string) (*types.OrganizationMember, error) {
	member := new(types.OrganizationMember)
	err := r.db.NewSelect().Model(member).
		Where("organization_id = ? AND user_id = ?", organizationID, userID).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return member, err
}

func (r *BunOrganizationMemberRepository) Update(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if r.hooks != nil {
			if err := r.hooks.BeforeUpdateOrganizationMember(member); err != nil {
				return err
			}
		}

		_, err := tx.NewUpdate().Model(member).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(member).WherePK().Scan(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterUpdateOrganizationMember(*member); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return member, nil
}

func (r *BunOrganizationMemberRepository) Delete(ctx context.Context, memberID string) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		memberRepo := r.WithTx(tx)
		member, err := memberRepo.GetByID(ctx, memberID)
		if err != nil {
			return err
		}
		if member == nil {
			return nil
		}

		if r.hooks != nil {
			if err := r.hooks.BeforeDeleteOrganizationMember(member); err != nil {
				return err
			}
		}

		if _, err := tx.NewDelete().Model(&types.OrganizationMember{}).Where("id = ?", memberID).Exec(ctx); err != nil {
			return err
		}

		if r.hooks != nil {
			if err := r.hooks.AfterDeleteOrganizationMember(*member); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *BunOrganizationMemberRepository) WithTx(tx bun.IDB) OrganizationMemberRepository {
	return &BunOrganizationMemberRepository{db: tx, hooks: r.hooks}
}
