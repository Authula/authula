package types

import (
	"time"

	"github.com/uptrace/bun"
)

// OAuth client ownership and protocol constants shared by the authorization
// server repositories and services.
const (
	OAuthClientOwnerTypeUser         = "user"
	OAuthClientOwnerTypeOrganization = "organization"

	OAuthGrantTypeAuthorizationCode = "authorization_code"
	OAuthGrantTypeClientCredentials = "client_credentials"

	OAuthConsentSourceExplicit   = "explicit"
	OAuthConsentSourceTrusted    = "trusted"
	OAuthConsentSourcePromptNone = "prompt_none"

	CodeChallengeMethodS256 = "S256"

	OAuthFlowPurposeAuthorization = "authorization"
	OAuthFlowPurposeCallback      = "callback"

	ScopeOpenID  = "openid"
	ScopeProfile = "profile"
	ScopeEmail   = "email"
)

// OAuthClient is a registered OAuth 2.0 client. Client secrets and other
// one-time credentials are never stored in plain text.
type OAuthClient struct {
	bun.BaseModel `bun:"table:oauth_clients"`

	ID                 string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	ClientID           string     `json:"client_id" required:"true" nullable:"false" bun:"column:client_id"`
	ClientSecretHash   string     `json:"-" required:"true" nullable:"false" bun:"column:client_secret_hash"`
	ClientSecretSalt   string     `json:"-" required:"true" nullable:"false" bun:"column:client_secret_salt"`
	ClientSecretPrefix string     `json:"client_secret_prefix" required:"true" nullable:"false" bun:"column:client_secret_prefix"`
	Name               string     `json:"name" required:"true" nullable:"false" bun:"column:name"`
	RedirectURIs       []string   `json:"redirect_uris" required:"true" nullable:"false" bun:"column:redirect_uris"`
	AllowedGrants      []string   `json:"allowed_grants" required:"true" nullable:"false" bun:"column:allowed_grants"`
	AllowedScopes      []string   `json:"allowed_scopes" required:"true" nullable:"false" bun:"column:allowed_scopes"`
	OwnerType          string     `json:"owner_type" required:"true" nullable:"false" bun:"column:owner_type"`
	OwnerID            string     `json:"owner_id" required:"true" nullable:"false" bun:"column:owner_id"`
	CreatorID          string     `json:"creator_id" required:"true" nullable:"false" bun:"column:creator_id"`
	OrganizationID     *string    `json:"organization_id" nullable:"true" bun:"column:organization_id"`
	IsActive           bool       `json:"is_active" required:"true" nullable:"false" bun:"column:is_active"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty" nullable:"true" bun:"column:deleted_at,nullzero"`
	CreatedAt          time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt          time.Time  `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

// OAuthAuthorizationCode is a single-use authorization code. Only the SHA-256
// hash of the code is stored.
type OAuthAuthorizationCode struct {
	bun.BaseModel `bun:"table:authorization_codes"`

	ID                  string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	CodeHash            string     `json:"code_hash" required:"true" nullable:"false" bun:"column:code_hash"`
	ClientID            string     `json:"client_id" required:"true" nullable:"false" bun:"column:client_id"`
	UserID              string     `json:"user_id" required:"true" nullable:"false" bun:"column:user_id"`
	SessionID           string     `json:"session_id" required:"true" nullable:"false" bun:"column:session_id"`
	ConsentID           string     `json:"consent_id" required:"true" nullable:"false" bun:"column:consent_id"`
	RedirectURI         string     `json:"redirect_uri" required:"true" nullable:"false" bun:"column:redirect_uri"`
	Scope               string     `json:"scope" required:"true" nullable:"false" bun:"column:scope"`
	Nonce               *string    `json:"nonce" nullable:"true" bun:"column:nonce"`
	CodeChallenge       string     `json:"code_challenge" required:"true" nullable:"false" bun:"column:code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method" required:"true" nullable:"false" bun:"column:code_challenge_method"`
	AuthTime            time.Time  `json:"auth_time" required:"true" nullable:"false" bun:"column:auth_time"`
	ExpiresAt           time.Time  `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty" nullable:"true" bun:"column:consumed_at,nullzero"`
	CreatedAt           time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
}

// OAuthConsentRequest preserves an authorization request while the user is
// asked for consent. The raw consent token is only returned to the caller.
type OAuthConsentRequest struct {
	bun.BaseModel `bun:"table:oauth_consent_requests"`

	ID                  string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	RequestHash         string     `json:"-" required:"true" nullable:"false" bun:"column:request_hash"`
	ClientID            string     `json:"client_id" required:"true" nullable:"false" bun:"column:client_id"`
	UserID              string     `json:"user_id" required:"true" nullable:"false" bun:"column:user_id"`
	SessionID           string     `json:"session_id" required:"true" nullable:"false" bun:"column:session_id"`
	RedirectURI         string     `json:"redirect_uri" required:"true" nullable:"false" bun:"column:redirect_uri"`
	Scope               string     `json:"scope" required:"true" nullable:"false" bun:"column:scope"`
	State               *string    `json:"state" nullable:"true" bun:"column:state"`
	Nonce               *string    `json:"nonce" nullable:"true" bun:"column:nonce"`
	CodeChallenge       string     `json:"code_challenge" required:"true" nullable:"false" bun:"column:code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method" required:"true" nullable:"false" bun:"column:code_challenge_method"`
	Prompt              string     `json:"prompt" nullable:"true" bun:"column:prompt"`
	MaxAge              *int       `json:"max_age" nullable:"true" bun:"column:max_age"`
	AuthTime            time.Time  `json:"auth_time" required:"true" nullable:"false" bun:"column:auth_time"`
	ExpiresAt           time.Time  `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty" nullable:"true" bun:"column:consumed_at,nullzero"`
	CreatedAt           time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
}

// OAuthConsent records a user's grant of scopes to a client. Revoked rows are
// retained so scope changes can be audited.
type OAuthConsent struct {
	bun.BaseModel `bun:"table:oauth_consents"`

	ID        string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	UserID    string     `json:"user_id" required:"true" nullable:"false" bun:"column:user_id"`
	ClientID  string     `json:"client_id" required:"true" nullable:"false" bun:"column:client_id"`
	Scope     string     `json:"scope" required:"true" nullable:"false" bun:"column:scope"`
	Source    string     `json:"source" required:"true" nullable:"false" bun:"column:source"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" nullable:"true" bun:"column:revoked_at,nullzero"`
	CreatedAt time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt time.Time  `json:"updated_at" required:"true" nullable:"false" bun:"column:updated_at,default:current_timestamp"`
}

// OAuthLoginTransaction preserves an authorize request across a login redirect
// for prompt=login and max_age handling.
type OAuthLoginTransaction struct {
	bun.BaseModel `bun:"table:oauth_login_transactions"`

	ID                  string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	TransactionHash     string     `json:"-" required:"true" nullable:"false" bun:"column:transaction_hash"`
	ClientID            string     `json:"client_id" required:"true" nullable:"false" bun:"column:client_id"`
	RedirectURI         string     `json:"redirect_uri" required:"true" nullable:"false" bun:"column:redirect_uri"`
	Scope               string     `json:"scope" required:"true" nullable:"false" bun:"column:scope"`
	State               *string    `json:"state" nullable:"true" bun:"column:state"`
	Nonce               *string    `json:"nonce" nullable:"true" bun:"column:nonce"`
	CodeChallenge       string     `json:"code_challenge" required:"true" nullable:"false" bun:"column:code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method" required:"true" nullable:"false" bun:"column:code_challenge_method"`
	Prompt              string     `json:"prompt" nullable:"true" bun:"column:prompt"`
	MaxAge              *int       `json:"max_age" nullable:"true" bun:"column:max_age"`
	AuthTime            time.Time  `json:"auth_time" required:"true" nullable:"false" bun:"column:auth_time"`
	CompletedAt         *time.Time `json:"completed_at,omitempty" nullable:"true" bun:"column:completed_at,nullzero"`
	ExpiresAt           time.Time  `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty" nullable:"true" bun:"column:consumed_at,nullzero"`
	CreatedAt           time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
}

// OAuthFlowState replaces the outbound provider flow cookies. The stored PKCE
// verifier is encrypted with the JWT token service, not hashed.
type OAuthFlowState struct {
	bun.BaseModel `bun:"table:oauth_flow_states"`

	ID                    string     `json:"id" required:"true" nullable:"false" bun:"column:id,pk"`
	StateHash             string     `json:"-" required:"true" nullable:"false" bun:"column:state_hash"`
	Provider              string     `json:"provider" required:"true" nullable:"false" bun:"column:provider"`
	Purpose               string     `json:"purpose" required:"true" nullable:"false" bun:"column:purpose"`
	UserID                *string    `json:"user_id" nullable:"true" bun:"column:user_id"`
	RedirectTo            *string    `json:"redirect_to" nullable:"true" bun:"column:redirect_to"`
	CodeVerifierEncrypted string     `json:"-" required:"true" nullable:"false" bun:"column:code_verifier_encrypted"`
	ExpiresAt             time.Time  `json:"expires_at" required:"true" nullable:"false" bun:"column:expires_at"`
	ConsumedAt            *time.Time `json:"consumed_at,omitempty" nullable:"true" bun:"column:consumed_at,nullzero"`
	CreatedAt             time.Time  `json:"created_at" required:"true" nullable:"false" bun:"column:created_at,default:current_timestamp"`
}
