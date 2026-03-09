package repositories

import (
	"context"

	"github.com/GoBetterAuth/go-better-auth/v2/plugins/admin/types"
)

type UserStateRepository interface {
	GetByUserID(ctx context.Context, userID string) (*types.AdminUserState, error)
	Upsert(ctx context.Context, state *types.AdminUserState) error
	Delete(ctx context.Context, userID string) error
	GetBanned(ctx context.Context) ([]types.AdminUserState, error)
}

type SessionStateRepository interface {
	GetBySessionID(ctx context.Context, sessionID string) (*types.AdminSessionState, error)
	Upsert(ctx context.Context, state *types.AdminSessionState) error
	Delete(ctx context.Context, sessionID string) error
	GetRevoked(ctx context.Context) ([]types.AdminSessionState, error)
	SessionExists(ctx context.Context, sessionID string) (bool, error)
	GetByUserID(ctx context.Context, userID string) ([]types.AdminUserSession, error)
}

type ImpersonationRepository interface {
	CreateImpersonation(ctx context.Context, impersonation *types.Impersonation) error
	GetAllImpersonations(ctx context.Context) ([]types.Impersonation, error)
	GetImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error)
	GetActiveImpersonationByID(ctx context.Context, impersonationID string) (*types.Impersonation, error)
	GetLatestActiveImpersonationByActor(ctx context.Context, actorUserID string) (*types.Impersonation, error)
	EndImpersonation(ctx context.Context, impersonationID string, endedByUserID *string) error
	UserExists(ctx context.Context, userID string) (bool, error)
}
