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

	OAuthFlowPurposeLogin = "login"
	OAuthFlowPurposeLink  = "link"

	ScopeOpenID  = "openid"
	ScopeProfile = "profile"
	ScopeEmail   = "email"
)

// OAuthClient is a registered OAuth 2.0 client. Client secrets and other
// one-time credentials are never stored in plain text.
type OAuthClient struct {
	bun.BaseModel `bun:"table:oauth_clients"`

	ID                 string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	ClientID           string     `json:"client_id" bun:"column:client_id,unique" required:"true" nullable:"false"`
	ClientSecretHash   string     `json:"-" bun:"column:client_secret_hash" required:"true" nullable:"false"`
	ClientSecretSalt   string     `json:"-" bun:"column:client_secret_salt" required:"true" nullable:"false"`
	ClientSecretPrefix string     `json:"client_secret_prefix" bun:"column:client_secret_prefix" required:"true" nullable:"false"`
	Name               string     `json:"name" bun:"column:name" required:"true" nullable:"false"`
	RedirectURIs       []string   `json:"redirect_uris" bun:"column:redirect_uris" required:"true" nullable:"false"`
	AllowedGrants      []string   `json:"allowed_grants" bun:"column:allowed_grants" required:"true" nullable:"false"`
	AllowedScopes      []string   `json:"allowed_scopes" bun:"column:allowed_scopes" required:"true" nullable:"false"`
	OwnerType          string     `json:"owner_type" bun:"column:owner_type" required:"true" nullable:"false"`
	OwnerID            string     `json:"owner_id" bun:"column:owner_id" required:"true" nullable:"false"`
	CreatorID          string     `json:"creator_id" bun:"column:creator_id" required:"true" nullable:"false"`
	OrganizationID     *string    `json:"organization_id" bun:"column:organization_id" nullable:"true"`
	IsActive           bool       `json:"is_active" bun:"column:is_active" required:"true" nullable:"false"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty" bun:"column:deleted_at,nullzero" nullable:"true"`
	CreatedAt          time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
	UpdatedAt          time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthAuthorizationCode is a single-use authorization code. Only the SHA-256
// hash of the code is stored.
type OAuthAuthorizationCode struct {
	bun.BaseModel `bun:"table:authorization_codes"`

	ID                  string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	CodeHash            string     `json:"code_hash" bun:"column:code_hash,unique" required:"true" nullable:"false"`
	ClientID            string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	UserID              string     `json:"user_id" bun:"column:user_id" required:"true" nullable:"false"`
	SessionID           string     `json:"session_id" bun:"column:session_id" required:"true" nullable:"false"`
	ConsentID           string     `json:"consent_id" bun:"column:consent_id" required:"true" nullable:"false"`
	RedirectURI         string     `json:"redirect_uri" bun:"column:redirect_uri" required:"true" nullable:"false"`
	Scope               string     `json:"scope" bun:"column:scope" required:"true" nullable:"false"`
	Nonce               *string    `json:"nonce" bun:"column:nonce" nullable:"true"`
	CodeChallenge       string     `json:"code_challenge" bun:"column:code_challenge" required:"true" nullable:"false"`
	CodeChallengeMethod string     `json:"code_challenge_method" bun:"column:code_challenge_method" required:"true" nullable:"false"`
	AuthTime            time.Time  `json:"auth_time" bun:"column:auth_time" required:"true" nullable:"false"`
	ExpiresAt           time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty" bun:"column:consumed_at,nullzero" nullable:"true"`
	CreatedAt           time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthConsentRequest preserves an authorization request while the user is
// asked for consent. The raw consent token is only returned to the caller.
type OAuthConsentRequest struct {
	bun.BaseModel `bun:"table:oauth_consent_requests"`

	ID                  string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	RequestHash         string     `json:"-" bun:"column:request_hash,unique" required:"true" nullable:"false"`
	ClientID            string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	UserID              string     `json:"user_id" bun:"column:user_id" required:"true" nullable:"false"`
	SessionID           string     `json:"session_id" bun:"column:session_id" required:"true" nullable:"false"`
	RedirectURI         string     `json:"redirect_uri" bun:"column:redirect_uri" required:"true" nullable:"false"`
	Scope               string     `json:"scope" bun:"column:scope" required:"true" nullable:"false"`
	State               *string    `json:"state" bun:"column:state" nullable:"true"`
	Nonce               *string    `json:"nonce" bun:"column:nonce" nullable:"true"`
	CodeChallenge       string     `json:"code_challenge" bun:"column:code_challenge" required:"true" nullable:"false"`
	CodeChallengeMethod string     `json:"code_challenge_method" bun:"column:code_challenge_method" required:"true" nullable:"false"`
	Prompt              string     `json:"prompt" bun:"column:prompt" nullable:"true"`
	MaxAge              *int       `json:"max_age" bun:"column:max_age" nullable:"true"`
	AuthTime            time.Time  `json:"auth_time" bun:"column:auth_time" required:"true" nullable:"false"`
	ExpiresAt           time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty" bun:"column:consumed_at,nullzero" nullable:"true"`
	CreatedAt           time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthConsent records a user's grant of scopes to a client. Revoked rows are
// retained so scope changes can be audited.
type OAuthConsent struct {
	bun.BaseModel `bun:"table:oauth_consents"`

	ID        string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	UserID    string     `json:"user_id" bun:"column:user_id" required:"true" nullable:"false"`
	ClientID  string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	Scope     string     `json:"scope" bun:"column:scope" required:"true" nullable:"false"`
	Source    string     `json:"source" bun:"column:source" required:"true" nullable:"false"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" bun:"column:revoked_at,nullzero" nullable:"true"`
	CreatedAt time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
	UpdatedAt time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthLoginTransaction preserves an authorize request across a login redirect
// for prompt=login and max_age handling.
type OAuthLoginTransaction struct {
	bun.BaseModel `bun:"table:oauth_login_transactions"`

	ID                  string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	TransactionHash     string     `json:"-" bun:"column:transaction_hash,unique" required:"true" nullable:"false"`
	ClientID            string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	RedirectURI         string     `json:"redirect_uri" bun:"column:redirect_uri" required:"true" nullable:"false"`
	Scope               string     `json:"scope" bun:"column:scope" required:"true" nullable:"false"`
	State               *string    `json:"state" bun:"column:state" nullable:"true"`
	Nonce               *string    `json:"nonce" bun:"column:nonce" nullable:"true"`
	CodeChallenge       string     `json:"code_challenge" bun:"column:code_challenge" required:"true" nullable:"false"`
	CodeChallengeMethod string     `json:"code_challenge_method" bun:"column:code_challenge_method" required:"true" nullable:"false"`
	Prompt              string     `json:"prompt" bun:"column:prompt" nullable:"true"`
	MaxAge              *int       `json:"max_age" bun:"column:max_age" nullable:"true"`
	AuthTime            time.Time  `json:"auth_time" bun:"column:auth_time" required:"true" nullable:"false"`
	CompletedAt         *time.Time `json:"completed_at,omitempty" bun:"column:completed_at,nullzero" nullable:"true"`
	ExpiresAt           time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty" bun:"column:consumed_at,nullzero" nullable:"true"`
	CreatedAt           time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthFlowState stores a single-use outbound provider flow. The stored PKCE
// verifier is encrypted with the token service, not hashed.
type OAuthFlowState struct {
	bun.BaseModel `bun:"table:oauth_flow_states"`

	ID                    string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	StateHash             string     `json:"-" bun:"column:state_hash,unique" required:"true" nullable:"false"`
	Provider              string     `json:"provider" bun:"column:provider" required:"true" nullable:"false"`
	Purpose               string     `json:"purpose" bun:"column:purpose" required:"true" nullable:"false"`
	UserID                *string    `json:"user_id" bun:"column:user_id" nullable:"true"`
	RedirectTo            *string    `json:"redirect_to" bun:"column:redirect_to" nullable:"true"`
	CodeVerifierEncrypted *string    `json:"-" bun:"column:code_verifier_encrypted" nullable:"true"`
	ExpiresAt             time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	ConsumedAt            *time.Time `json:"consumed_at,omitempty" bun:"column:consumed_at,nullzero" nullable:"true"`
	CreatedAt             time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
}
