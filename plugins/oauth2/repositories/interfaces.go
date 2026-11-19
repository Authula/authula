package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// ErrOAuthClientConflict is returned when an optimistic update does not match
// the stored updated_at value.
var ErrOAuthClientConflict = errors.New("oauth client was modified by another request")

// OAuthClientCursor is the keyset position for client list pagination.
type OAuthClientCursor struct {
	CreatedAt time.Time
	ID        string
}

// OAuthClientListFilter filters and paginates client listings.
type OAuthClientListFilter struct {
	OwnerType      string
	OwnerID        string
	OrganizationID *string
	IncludeDeleted bool
	Limit          int
	Cursor         *OAuthClientCursor
}

type OAuthClientRepository interface {
	Create(ctx context.Context, client *types.OAuthClient) (*types.OAuthClient, error)
	GetByID(ctx context.Context, id string) (*types.OAuthClient, error)
	GetByIDIncludingDeleted(ctx context.Context, id string) (*types.OAuthClient, error)
	GetByClientID(ctx context.Context, clientID string) (*types.OAuthClient, error)
	GetByClientIDIncludingDeleted(ctx context.Context, clientID string) (*types.OAuthClient, error)
	List(ctx context.Context, filter OAuthClientListFilter) ([]*types.OAuthClient, error)
	Update(ctx context.Context, client *types.OAuthClient, expectedUpdatedAt *time.Time) (*types.OAuthClient, error)
	SoftDelete(ctx context.Context, id string, deletedAt time.Time) (bool, error)
	WithTx(tx bun.IDB) OAuthClientRepository
}

// AuthRequestRepository owns the authorization request and its single-use
// authorization code. ConsumeCode is atomic: replaying or racing a code
// exchange cannot succeed twice.
type AuthRequestRepository interface {
	Create(ctx context.Context, request *types.OAuthAuthRequest) (*types.OAuthAuthRequest, error)
	GetByID(ctx context.Context, id string) (*types.OAuthAuthRequest, error)
	GetByBindingHash(ctx context.Context, bindingHash string) (*types.OAuthAuthRequest, error)
	GetByCodeHash(ctx context.Context, codeHash string) (*types.OAuthAuthRequest, error)
	ConsumeCode(ctx context.Context, codeHash string, clientID string, consumedAt time.Time) (*types.OAuthAuthRequest, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) AuthRequestRepository
}

// ConsentRequestRepository owns one-time consent challenges. Consume is atomic
// so a challenge cannot be completed twice.
type ConsentRequestRepository interface {
	Create(ctx context.Context, request *types.OAuthConsentRequest) (*types.OAuthConsentRequest, error)
	GetByID(ctx context.Context, id string) (*types.OAuthConsentRequest, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*types.OAuthConsentRequest, error)
	GetByCompletionCodeHash(ctx context.Context, completionCodeHash string) (*types.OAuthConsentRequest, error)
	Consume(ctx context.Context, tokenHash string, consumedAt time.Time) (*types.OAuthConsentRequest, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) ConsentRequestRepository
}

type ConsentRepository interface {
	Create(ctx context.Context, consent *types.OAuthConsent) (*types.OAuthConsent, error)
	GetByID(ctx context.Context, id string) (*types.OAuthConsent, error)
	GetActiveByUserAndClient(ctx context.Context, userID string, clientID string) (*types.OAuthConsent, error)
	ListActiveByUser(ctx context.Context, userID string) ([]*types.OAuthConsent, error)
	UpdateScope(ctx context.Context, id string, scope string, updatedAt time.Time) (*types.OAuthConsent, error)
	Revoke(ctx context.Context, id string, revokedAt time.Time) (bool, error)
	RevokeByUserAndClient(ctx context.Context, userID string, clientID string, revokedAt time.Time) (int64, error)
	RevokeByClient(ctx context.Context, clientID string, revokedAt time.Time) (int64, error)
	WithTx(tx bun.IDB) ConsentRepository
}

// AccessGrantRepository records issued access tokens and supports family-wide
// revocation.
type AccessGrantRepository interface {
	Create(ctx context.Context, grant *types.OAuthAccessGrant) (*types.OAuthAccessGrant, error)
	GetByID(ctx context.Context, id string) (*types.OAuthAccessGrant, error)
	GetByJTI(ctx context.Context, jti string) (*types.OAuthAccessGrant, error)
	Revoke(ctx context.Context, jti string, revokedAt time.Time) (bool, error)
	RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) (int64, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) AccessGrantRepository
}

// RefreshTokenRepository owns rotating refresh tokens. Rotate consumes the old
// token and inserts its replacement atomically; RevokeFamily revokes every
// token in a lineage after suspected reuse.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *types.OAuthRefreshToken) (*types.OAuthRefreshToken, error)
	GetByID(ctx context.Context, id string) (*types.OAuthRefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*types.OAuthRefreshToken, error)
	Rotate(ctx context.Context, oldTokenHash string, replacement *types.OAuthRefreshToken, consumedAt time.Time) (*types.OAuthRefreshToken, error)
	RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) (int64, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) RefreshTokenRepository
}

type FlowStateRepository interface {
	Create(ctx context.Context, state *types.OAuthFlowState) (*types.OAuthFlowState, error)
	GetByStateHash(ctx context.Context, stateHash string) (*types.OAuthFlowState, error)
	Consume(ctx context.Context, stateHash string, provider string, purpose string, consumedAt time.Time) (*types.OAuthFlowState, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) FlowStateRepository
}
