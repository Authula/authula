package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/types"
)

type teamMemberMemberUserRow struct {
	ID        string    `bun:"column:id"`
	TeamID    string    `bun:"column:team_id"`
	CreatedAt time.Time `bun:"column:created_at"`

	MemberID         string    `bun:"column:member_id"`
	MemberOrgID      string    `bun:"column:member_organization_id"`
	MemberRole       string    `bun:"column:member_role"`
	MemberCreatedAt  time.Time `bun:"column:member_created_at"`
	MemberUpdatedAt  time.Time `bun:"column:member_updated_at"`

	UserID            string         `bun:"column:user_id"`
	UserName          string         `bun:"column:user_name"`
	UserEmail         string         `bun:"column:user_email"`
	UserEmailVerified bool           `bun:"column:user_email_verified"`
	UserImage         *string        `bun:"column:user_image"`
	UserMetadata      map[string]any `bun:"column:user_metadata,type:jsonb"`
	UserCreatedAt     time.Time      `bun:"column:user_created_at"`
	UserUpdatedAt     time.Time      `bun:"column:user_updated_at"`
}

func mapToTeamMemberResponse(row teamMemberMemberUserRow) types.OrganizationTeamMemberResponse {
	return types.OrganizationTeamMemberResponse{
		ID:        row.ID,
		TeamID:    row.TeamID,
		CreatedAt: row.CreatedAt,
		Member: types.OrganizationMemberResponse{
			ID:             row.MemberID,
			OrganizationID: row.MemberOrgID,
			Role:           row.MemberRole,
			CreatedAt:      row.MemberCreatedAt,
			UpdatedAt:      row.MemberUpdatedAt,
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
		},
	}
}

const teamMemberWithMemberAndUserColumns = `tm.id, tm.team_id, tm.created_at,` +
	` m.id AS member_id, m.organization_id AS member_organization_id, m.role AS member_role,` +
	` m.created_at AS member_created_at, m.updated_at AS member_updated_at,` +
	` u.id AS user_id, u.name AS user_name, u.email AS user_email,` +
	` u.email_verified AS user_email_verified, u.image AS user_image,` +
	` u.metadata AS user_metadata, u.created_at AS user_created_at,` +
	` u.updated_at AS user_updated_at`

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

func (r *BunOrganizationTeamMemberRepository) GetAllByTeamIDWithMemberAndUser(ctx context.Context, teamID string, page int, limit int) ([]types.OrganizationTeamMemberResponse, error) {
	var rows []teamMemberMemberUserRow
	err := r.db.NewRaw(`
		SELECT `+teamMemberWithMemberAndUserColumns+`
		FROM organization_team_members tm
		INNER JOIN organization_members m ON m.id = tm.member_id
		INNER JOIN users u ON u.id = m.user_id
		WHERE tm.team_id = ?
		ORDER BY tm.created_at DESC, tm.id DESC
		LIMIT ? OFFSET ?
	`, teamID, limit, (page-1)*limit).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	results := make([]types.OrganizationTeamMemberResponse, len(rows))
	for i, row := range rows {
		results[i] = mapToTeamMemberResponse(row)
	}
	return results, nil
}

func (r *BunOrganizationTeamMemberRepository) GetByIDWithMemberAndUser(ctx context.Context, teamMemberID string) (*types.OrganizationTeamMemberResponse, error) {
	var row teamMemberMemberUserRow
	err := r.db.NewRaw(`
		SELECT `+teamMemberWithMemberAndUserColumns+`
		FROM organization_team_members tm
		INNER JOIN organization_members m ON m.id = tm.member_id
		INNER JOIN users u ON u.id = m.user_id
		WHERE tm.id = ?
	`, teamMemberID).Scan(ctx, &row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := mapToTeamMemberResponse(row)
	return &result, nil
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
