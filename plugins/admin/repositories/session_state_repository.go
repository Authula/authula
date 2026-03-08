package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type BunSessionStateRepository struct {
	db bun.IDB
}

func NewBunSessionStateRepository(db bun.IDB) *BunSessionStateRepository {
	return &BunSessionStateRepository{db: db}
}

func (r *BunSessionStateRepository) GetBySessionID(ctx context.Context, sessionID string) (*types.AdminSessionState, error) {
	row := &types.AdminSessionState{}
	err := r.db.NewSelect().Model(row).Where("session_id = ?", sessionID).Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session state: %w", err)
	}

	return row, nil
}

func (r *BunSessionStateRepository) Upsert(ctx context.Context, state *types.AdminSessionState) error {
	now := time.Now().UTC()
	state.UpdatedAt = now

	var err error
	dialect := r.db.Dialect()

	// Check which database dialect we're using
	switch dialect.(type) {
	case *mysqldialect.Dialect:
		// MySQL uses ON DUPLICATE KEY UPDATE with VALUES() function
		_, err = r.db.NewInsert().
			Model(state).
			On("DUPLICATE KEY UPDATE").
			Set("revoked_at = VALUES(revoked_at)").
			Set("revoked_reason = VALUES(revoked_reason)").
			Set("revoked_by_user_id = VALUES(revoked_by_user_id)").
			Set("impersonator_user_id = VALUES(impersonator_user_id)").
			Set("impersonation_reason = VALUES(impersonation_reason)").
			Set("impersonation_expires_at = VALUES(impersonation_expires_at)").
			Set("updated_at = ?", now).
			Exec(ctx)
	case *pgdialect.Dialect:
		// PostgreSQL uses ON CONFLICT with EXCLUDED
		_, err = r.db.NewInsert().
			Model(state).
			On("CONFLICT (session_id) DO UPDATE").
			Set("revoked_at = EXCLUDED.revoked_at").
			Set("revoked_reason = EXCLUDED.revoked_reason").
			Set("revoked_by_user_id = EXCLUDED.revoked_by_user_id").
			Set("impersonator_user_id = EXCLUDED.impersonator_user_id").
			Set("impersonation_reason = EXCLUDED.impersonation_reason").
			Set("impersonation_expires_at = EXCLUDED.impersonation_expires_at").
			Set("updated_at = ?", now).
			Exec(ctx)
	case *sqlitedialect.Dialect:
		// SQLite uses ON CONFLICT with excluded (lowercase)
		_, err = r.db.NewInsert().
			Model(state).
			On("CONFLICT (session_id) DO UPDATE").
			Set("revoked_at = excluded.revoked_at").
			Set("revoked_reason = excluded.revoked_reason").
			Set("revoked_by_user_id = excluded.revoked_by_user_id").
			Set("impersonator_user_id = excluded.impersonator_user_id").
			Set("impersonation_reason = excluded.impersonation_reason").
			Set("impersonation_expires_at = excluded.impersonation_expires_at").
			Set("updated_at = ?", now).
			Exec(ctx)
	default:
		// Fallback for unknown dialects - try PostgreSQL syntax
		_, err = r.db.NewInsert().
			Model(state).
			On("CONFLICT (session_id) DO UPDATE").
			Set("revoked_at = EXCLUDED.revoked_at").
			Set("revoked_reason = EXCLUDED.revoked_reason").
			Set("revoked_by_user_id = EXCLUDED.revoked_by_user_id").
			Set("impersonator_user_id = EXCLUDED.impersonator_user_id").
			Set("impersonation_reason = EXCLUDED.impersonation_reason").
			Set("impersonation_expires_at = EXCLUDED.impersonation_expires_at").
			Set("updated_at = ?", now).
			Exec(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to upsert session state: %w", err)
	}

	return nil
}

func (r *BunSessionStateRepository) Delete(ctx context.Context, sessionID string) error {
	_, err := r.db.NewDelete().Model((*types.AdminSessionState)(nil)).Where("session_id = ?", sessionID).Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete session state: %w", err)
	}

	return nil
}

func (r *BunSessionStateRepository) GetRevoked(ctx context.Context) ([]types.AdminSessionState, error) {
	var rows []types.AdminSessionState

	err := r.db.NewSelect().
		Model(&rows).
		Where("revoked_at IS NOT NULL").
		OrderExpr("updated_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get revoked session states: %w", err)
	}

	return rows, nil
}

func (r *BunSessionStateRepository) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	count, err := r.db.NewSelect().Table("sessions").Where("id = ?", sessionID).Count(ctx)

	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}

func (r *BunSessionStateRepository) GetByUserID(ctx context.Context, userID string) ([]types.AdminUserSession, error) {
	var sessions []models.Session

	err := r.db.NewSelect().
		Model(&sessions).
		Where("user_id = ?", userID).
		OrderExpr("created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user id: %w", err)
	}

	if len(sessions) == 0 {
		return []types.AdminUserSession{}, nil
	}

	sessionIDs := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
	}

	var states []types.AdminSessionState

	err = r.db.NewSelect().
		Model(&states).
		Where("session_id IN (?)", bun.List(sessionIDs)).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get session states by user id: %w", err)
	}

	statesBySessionID := make(map[string]*types.AdminSessionState, len(states))
	for i := range states {
		statesBySessionID[states[i].SessionID] = &states[i]
	}

	rows := make([]types.AdminUserSession, 0, len(sessions))
	for _, session := range sessions {
		rows = append(rows, types.AdminUserSession{
			Session: session,
			State:   statesBySessionID[session.ID],
		})
	}

	return rows, nil
}
