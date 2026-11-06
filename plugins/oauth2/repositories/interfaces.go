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

type AuthorizationCodeRepository interface {
	Create(ctx context.Context, code *types.OAuthAuthorizationCode) (*types.OAuthAuthorizationCode, error)
	GetByCodeHash(ctx context.Context, codeHash string) (*types.OAuthAuthorizationCode, error)
	Consume(ctx context.Context, codeHash string, clientID string, consumedAt time.Time) (*types.OAuthAuthorizationCode, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) AuthorizationCodeRepository
}

type ConsentRequestRepository interface {
	Create(ctx context.Context, request *types.OAuthConsentRequest) (*types.OAuthConsentRequest, error)
	GetByRequestHash(ctx context.Context, requestHash string) (*types.OAuthConsentRequest, error)
	Consume(ctx context.Context, requestHash string, consumedAt time.Time) (*types.OAuthConsentRequest, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) ConsentRequestRepository
}

type ConsentRepository interface {
	Create(ctx context.Context, consent *types.OAuthConsent) (*types.OAuthConsent, error)
	GetByID(ctx context.Context, id string) (*types.OAuthConsent, error)
	GetActiveByUserAndClient(ctx context.Context, userID string, clientID string) (*types.OAuthConsent, error)
	ListActiveByUser(ctx context.Context, userID string) ([]*types.OAuthConsent, error)
	Revoke(ctx context.Context, id string, revokedAt time.Time) (bool, error)
	RevokeByUserAndClient(ctx context.Context, userID string, clientID string, revokedAt time.Time) (int64, error)
	RevokeByClient(ctx context.Context, clientID string, revokedAt time.Time) (int64, error)
	WithTx(tx bun.IDB) ConsentRepository
}

type LoginTransactionRepository interface {
	Create(ctx context.Context, transaction *types.OAuthLoginTransaction) (*types.OAuthLoginTransaction, error)
	GetByTransactionHash(ctx context.Context, transactionHash string) (*types.OAuthLoginTransaction, error)
	MarkCompleted(ctx context.Context, transactionHash string, completedAt time.Time) (bool, error)
	Consume(ctx context.Context, transactionHash string, consumedAt time.Time) (*types.OAuthLoginTransaction, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) LoginTransactionRepository
}

type FlowStateRepository interface {
	Create(ctx context.Context, state *types.OAuthFlowState) (*types.OAuthFlowState, error)
	GetByStateHash(ctx context.Context, stateHash string) (*types.OAuthFlowState, error)
	Consume(ctx context.Context, stateHash string, provider string, purpose string, consumedAt time.Time) (*types.OAuthFlowState, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) FlowStateRepository
}
