package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/organizations/types"
)

type BunOrganizationInvitationRepository struct {
	db bun.IDB
}

func NewBunOrganizationInvitationRepository(db bun.IDB) OrganizationInvitationRepository {
	return &BunOrganizationInvitationRepository{db: db}
}

func (r *BunOrganizationInvitationRepository) Create(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(invitation).Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(invitation).WherePK().Scan(ctx); err != nil {
			return err
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

// pendingByEmailWhere matches invitations that are still awaiting a decision and
// have not expired. Both pending lookups order oldest first, because acceptance
// resolves role conflicts in favour of the earliest invitation.
const pendingByEmailWhere = "email = ? AND status = ? AND expires_at > ?"

func (r *BunOrganizationInvitationRepository) ListAllPendingByEmail(ctx context.Context, email string, page int, limit int) ([]types.OrganizationInvitation, int, error) {
	invites := make([]types.OrganizationInvitation, 0)
	limit = pageLimit(limit)
	total, err := r.db.NewSelect().Model(&invites).
		Where(pendingByEmailWhere, email, types.OrganizationInvitationStatusPending, time.Now().UTC()).
		OrderExpr("created_at ASC, id ASC").
		Offset(pageOffset(page, limit)).Limit(limit).
		ScanAndCount(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationInvitation{}, total, nil
	}
	return invites, total, err
}

func (r *BunOrganizationInvitationRepository) GetAllPendingByEmail(ctx context.Context, email string) ([]types.OrganizationInvitation, error) {
	invites := make([]types.OrganizationInvitation, 0)
	err := r.db.NewSelect().Model(&invites).
		Where(pendingByEmailWhere, email, types.OrganizationInvitationStatusPending, time.Now().UTC()).
		OrderExpr("created_at ASC, id ASC").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return []types.OrganizationInvitation{}, nil
	}
	return invites, err
}

func (r *BunOrganizationInvitationRepository) Update(ctx context.Context, invitation *types.OrganizationInvitation) (*types.OrganizationInvitation, error) {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model(invitation).WherePK().Exec(ctx)
		if err != nil {
			return err
		}

		if err := tx.NewSelect().Model(invitation).WherePK().Scan(ctx); err != nil {
			return err
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
	return &BunOrganizationInvitationRepository{db: tx}
}

type invitationOrgRow struct {
	ID             string    `bun:"column:id"`
	Email          string    `bun:"column:email"`
	InviterID      string    `bun:"column:inviter_id"`
	OrganizationID string    `bun:"column:organization_id"`
	Role           string    `bun:"column:role"`
	Status         string    `bun:"column:status"`
	ExpiresAt      time.Time `bun:"column:expires_at"`
	CreatedAt      time.Time `bun:"column:created_at"`

	OrgID       string         `bun:"column:org_id"`
	OrgOwnerID  string         `bun:"column:org_owner_id"`
	OrgName     string         `bun:"column:org_name"`
	OrgSlug     string         `bun:"column:org_slug"`
	OrgLogo     *string        `bun:"column:org_logo"`
	OrgMetadata map[string]any `bun:"column:org_metadata,type:jsonb"`
}

const invitationWithOrgColumns = `i.id, i.email, i.inviter_id, i.organization_id, i.role, i.status, i.expires_at, i.created_at,` +
	` o.id AS org_id, o.owner_id AS org_owner_id, o.name AS org_name,` +
	` o.slug AS org_slug, o.logo AS org_logo, o.metadata AS org_metadata`

const invitationWithOrgByOrganizationFrom = ` FROM organization_invitations i` +
	` INNER JOIN organizations o ON o.id = i.organization_id` +
	` WHERE i.organization_id = ?`

func mapToInvitationWithOrgResponse(row invitationOrgRow) types.GetOrganizationInvitationResponse {
	return types.GetOrganizationInvitationResponse{
		Invitation: &types.OrganizationInvitation{
			ID:             row.ID,
			Email:          row.Email,
			InviterID:      row.InviterID,
			OrganizationID: row.OrganizationID,
			Role:           row.Role,
			Status:         types.OrganizationInvitationStatus(row.Status),
			ExpiresAt:      row.ExpiresAt,
			CreatedAt:      row.CreatedAt,
		},
		Organization: types.OrganizationSummary{
			ID:       row.OrgID,
			OwnerID:  row.OrgOwnerID,
			Name:     row.OrgName,
			Slug:     row.OrgSlug,
			Logo:     row.OrgLogo,
			Metadata: row.OrgMetadata,
		},
	}
}

func (r *BunOrganizationInvitationRepository) GetByIDWithOrg(ctx context.Context, invitationID string) (*types.GetOrganizationInvitationResponse, error) {
	var row invitationOrgRow
	err := r.db.NewRaw(`
		SELECT `+invitationWithOrgColumns+`
		FROM organization_invitations i
		INNER JOIN organizations o ON o.id = i.organization_id
		WHERE i.id = ?
	`, invitationID).Scan(ctx, &row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := mapToInvitationWithOrgResponse(row)
	return &result, nil
}

func (r *BunOrganizationInvitationRepository) ListAllByOrganizationIDWithOrg(ctx context.Context, organizationID string, page int, limit int) ([]types.GetOrganizationInvitationResponse, int, error) {
	limit = pageLimit(limit)

	var total int
	if err := r.db.NewRaw(`SELECT COUNT(*)`+invitationWithOrgByOrganizationFrom, organizationID).Scan(ctx, &total); err != nil {
		return nil, 0, err
	}

	var rows []invitationOrgRow
	err := r.db.NewRaw(`SELECT `+invitationWithOrgColumns+invitationWithOrgByOrganizationFrom+`
		ORDER BY i.created_at DESC, i.id DESC
		LIMIT ? OFFSET ?
	`, organizationID, limit, pageOffset(page, limit)).Scan(ctx, &rows)
	if err == sql.ErrNoRows {
		return []types.GetOrganizationInvitationResponse{}, total, nil
	}
	if err != nil {
		return nil, 0, err
	}
	results := make([]types.GetOrganizationInvitationResponse, len(rows))
	for i, row := range rows {
		results[i] = mapToInvitationWithOrgResponse(row)
	}
	return results, total, nil
}

func (r *BunOrganizationInvitationRepository) GetAllByOrganizationIDWithOrg(ctx context.Context, organizationID string) ([]types.GetOrganizationInvitationResponse, error) {
	var rows []invitationOrgRow
	err := r.db.NewRaw(`SELECT `+invitationWithOrgColumns+invitationWithOrgByOrganizationFrom+`
		ORDER BY i.created_at DESC, i.id DESC
	`, organizationID).Scan(ctx, &rows)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	results := make([]types.GetOrganizationInvitationResponse, len(rows))
	for i, row := range rows {
		results[i] = mapToInvitationWithOrgResponse(row)
	}
	return results, nil
}
