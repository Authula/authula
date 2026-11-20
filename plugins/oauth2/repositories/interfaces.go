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

// Sentinel errors returned by single-record getters when no row matches.
var (
	ErrOAuthClientNotFound    = errors.New("oauth client not found")
	ErrAuthRequestNotFound    = errors.New("oauth auth request not found")
	ErrConsentRequestNotFound = errors.New("oauth consent request not found")
	ErrConsentNotFound        = errors.New("oauth consent not found")
	ErrAccessGrantNotFound    = errors.New("oauth access grant not found")
	ErrRefreshTokenNotFound   = errors.New("oauth refresh token not found")
	ErrFlowStateNotFound      = errors.New("oauth flow state not found")
)

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
	// GetByID returns ErrOAuthClientNotFound when no active client matches.
	GetByID(ctx context.Context, id string) (*types.OAuthClient, error)
	// GetByIDIncludingDeleted returns ErrOAuthClientNotFound when no client matches.
	GetByIDIncludingDeleted(ctx context.Context, id string) (*types.OAuthClient, error)
	// GetByClientID returns ErrOAuthClientNotFound when no active client matches.
	GetByClientID(ctx context.Context, clientID string) (*types.OAuthClient, error)
	// GetByClientIDIncludingDeleted returns ErrOAuthClientNotFound when no client matches.
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
	// GetByID returns ErrAuthRequestNotFound when no request matches.
	GetByID(ctx context.Context, id string) (*types.OAuthAuthRequest, error)
	// GetByBindingHash returns ErrAuthRequestNotFound when no request matches.
	GetByBindingHash(ctx context.Context, bindingHash string) (*types.OAuthAuthRequest, error)
	// GetByCodeHash returns ErrAuthRequestNotFound when no request matches.
	GetByCodeHash(ctx context.Context, codeHash string) (*types.OAuthAuthRequest, error)
	// ConsumeCode returns (nil, nil) when the code is missing, expired, already
	// consumed or belongs to another client; those outcomes are expected and do
	// not indicate an infrastructure failure.
	ConsumeCode(ctx context.Context, codeHash string, clientID string, consumedAt time.Time) (*types.OAuthAuthRequest, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) AuthRequestRepository
}

// ConsentRequestRepository owns one-time consent challenges. Consume is atomic
// so a challenge cannot be completed twice.
type ConsentRequestRepository interface {
	Create(ctx context.Context, request *types.OAuthConsentRequest) (*types.OAuthConsentRequest, error)
	// GetByID returns ErrConsentRequestNotFound when no request matches.
	GetByID(ctx context.Context, id string) (*types.OAuthConsentRequest, error)
	// GetByTokenHash returns ErrConsentRequestNotFound when no request matches.
	GetByTokenHash(ctx context.Context, tokenHash string) (*types.OAuthConsentRequest, error)
	// GetByCompletionCodeHash returns ErrConsentRequestNotFound when no request matches.
	GetByCompletionCodeHash(ctx context.Context, completionCodeHash string) (*types.OAuthConsentRequest, error)
	// Consume returns (nil, nil) when the challenge is missing, expired or
	// already consumed; those outcomes are expected and not infrastructure failures.
	Consume(ctx context.Context, tokenHash string, consumedAt time.Time) (*types.OAuthConsentRequest, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) ConsentRequestRepository
}

type ConsentRepository interface {
	Create(ctx context.Context, consent *types.OAuthConsent) (*types.OAuthConsent, error)
	// GetByID returns ErrConsentNotFound when no consent matches.
	GetByID(ctx context.Context, id string) (*types.OAuthConsent, error)
	// GetActiveByUserAndClient returns ErrConsentNotFound when the user has no
	// active consent for the client.
	GetActiveByUserAndClient(ctx context.Context, userID string, clientID string) (*types.OAuthConsent, error)
	ListActiveByUser(ctx context.Context, userID string) ([]*types.OAuthConsent, error)
	// UpdateScope returns ErrConsentNotFound when no active consent matches.
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
	// GetByID returns ErrAccessGrantNotFound when no grant matches.
	GetByID(ctx context.Context, id string) (*types.OAuthAccessGrant, error)
	// GetByJTI returns ErrAccessGrantNotFound when no grant matches.
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
	// GetByID returns ErrRefreshTokenNotFound when no token matches.
	GetByID(ctx context.Context, id string) (*types.OAuthRefreshToken, error)
	// GetByTokenHash returns ErrRefreshTokenNotFound when no token matches.
	GetByTokenHash(ctx context.Context, tokenHash string) (*types.OAuthRefreshToken, error)
	// Rotate returns (nil, nil) when the token is missing, expired, revoked or
	// already consumed; those outcomes are expected and not infrastructure failures.
	Rotate(ctx context.Context, oldTokenHash string, replacement *types.OAuthRefreshToken, consumedAt time.Time) (*types.OAuthRefreshToken, error)
	RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) (int64, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) RefreshTokenRepository
}

type FlowStateRepository interface {
	Create(ctx context.Context, state *types.OAuthFlowState) (*types.OAuthFlowState, error)
	// GetByStateHash returns ErrFlowStateNotFound when no state matches.
	GetByStateHash(ctx context.Context, stateHash string) (*types.OAuthFlowState, error)
	// Consume returns (nil, nil) when the state is missing, expired, already
	// consumed or does not match the provider/purpose; those outcomes are
	// expected and not infrastructure failures.
	Consume(ctx context.Context, stateHash string, provider string, purpose string, consumedAt time.Time) (*types.OAuthFlowState, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithTx(tx bun.IDB) FlowStateRepository
}
