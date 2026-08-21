package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

type memberUserRow struct {
	ID             string    `bun:"column:id"`
	OrganizationID string    `bun:"column:organization_id"`
	Role           string    `bun:"column:role"`
	CreatedAt      time.Time `bun:"column:created_at"`
	UpdatedAt      time.Time `bun:"column:updated_at"`

	UserID            string         `bun:"column:user_id"`
	UserName          string         `bun:"column:user_name"`
	UserEmail         string         `bun:"column:user_email"`
	UserEmailVerified bool           `bun:"column:user_email_verified"`
	UserImage         *string        `bun:"column:user_image"`
	UserMetadata      map[string]any `bun:"column:user_metadata,type:jsonb"`
	UserCreatedAt     time.Time      `bun:"column:user_created_at"`
	UserUpdatedAt     time.Time      `bun:"column:user_updated_at"`
}

func mapToMemberResponse(row memberUserRow) types.OrganizationMemberResponse {
	return types.OrganizationMemberResponse{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Role:           row.Role,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		User: models.User{
			ID:            row.UserID,
			Name:          row.UserName,
			Email:         row.UserEmail,
			EmailVerified: row.UserEmailVerified,
			Image:         row.UserImage,
			Metadata:      row.UserMetadata,
			CreatedAt:     row.UserCreatedAt,
			UpdatedAt:     row.UserUpdatedAt,
		},
	}
}

const memberWithUserColumns = `m.id, m.organization_id, m.role, m.created_at, m.updated_at,` +
	` u.id AS user_id, u.name AS user_name, u.email AS user_email,` +
	` u.email_verified AS user_email_verified, u.image AS user_image,` +
	` u.metadata AS user_metadata, u.created_at AS user_created_at,` +
	` u.updated_at AS user_updated_at`

const memberWithUserByOrganizationFrom = ` FROM organization_members m` +
	` INNER JOIN users u ON u.id = m.user_id` +
	` WHERE m.organization_id = ?`

type BunOrganizationMemberRepository struct {
	db bun.IDB
}

func NewBunOrganizationMemberRepository(db bun.IDB) OrganizationMemberRepository {
	return &BunOrganizationMemberRepository{db: db}
}

func (r *BunOrganizationMemberRepository) Create(ctx context.Context, member *types.OrganizationMember) (*types.OrganizationMember, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(member).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(member).WherePK().Scan(ctx); err != nil {
			return err
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

func (r *BunOrganizationMemberRepository) GetAllByOrganizationID(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMember, int, error) {
	members := make([]types.OrganizationMember, 0)
	limit = pageLimit(limit)
	total, err := r.db.NewSelect().Model(&members).
		Where("organization_id = ?", organizationID).
		OrderExpr("created_at DESC, id DESC").
		Offset(pageOffset(page, limit)).Limit(limit).
		ScanAndCount(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationMember{}, total, nil
	}
	return members, total, err
}

func (r *BunOrganizationMemberRepository) GetAllByOrganizationIDWithUser(ctx context.Context, organizationID string, page int, limit int) ([]types.OrganizationMemberResponse, int, error) {
	limit = pageLimit(limit)

	var total int
	if err := r.db.NewRaw(`SELECT COUNT(*)`+memberWithUserByOrganizationFrom, organizationID).Scan(ctx, &total); err != nil {
		return nil, 0, err
	}

	var rows []memberUserRow
	err := r.db.NewRaw(`SELECT `+memberWithUserColumns+memberWithUserByOrganizationFrom+`
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT ? OFFSET ?
	`, organizationID, limit, pageOffset(page, limit)).Scan(ctx, &rows)
	if err != nil {
		return nil, 0, err
	}
	results := make([]types.OrganizationMemberResponse, len(rows))
	for i, row := range rows {
		results[i] = mapToMemberResponse(row)
	}
	return results, total, nil
}

func (r *BunOrganizationMemberRepository) GetByIDWithUser(ctx context.Context, memberID string) (*types.OrganizationMemberResponse, error) {
	var row memberUserRow
	err := r.db.NewRaw(`
		SELECT `+memberWithUserColumns+`
		FROM organization_members m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.id = ?
	`, memberID).Scan(ctx, &row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := mapToMemberResponse(row)
	return &result, nil
}

func (r *BunOrganizationMemberRepository) GetByOrganizationIDAndUserIDWithUser(ctx context.Context, organizationID string, userID string) (*types.OrganizationMemberResponse, error) {
	var row memberUserRow
	err := r.db.NewRaw(`
		SELECT `+memberWithUserColumns+`
		FROM organization_members m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = ? AND m.user_id = ?
	`, organizationID, userID).Scan(ctx, &row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := mapToMemberResponse(row)
	return &result, nil
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
		_, err := tx.NewUpdate().Model(member).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(member).WherePK().Scan(ctx); err != nil {
			return err
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
		if _, err := tx.NewDelete().Model(&types.OrganizationMember{}).Where("id = ?", memberID).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (r *BunOrganizationMemberRepository) WithTx(tx bun.IDB) OrganizationMemberRepository {
	return &BunOrganizationMemberRepository{db: tx}
}
