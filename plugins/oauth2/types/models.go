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

	OAuthApplicationTypeWeb    = "web"
	OAuthApplicationTypeNative = "native"

	OAuthAuthMethodClientSecretBasic = "client_secret_basic"
	OAuthAuthMethodClientSecretPost  = "client_secret_post"
	OAuthAuthMethodNone              = "none"

	OAuthAccessTokenTypeJWT    = "jwt"
	OAuthAccessTokenTypeOpaque = "opaque"

	OAuthGrantTypeAuthorizationCode = "authorization_code"
	OAuthGrantTypeClientCredentials = "client_credentials"
	OAuthGrantTypeRefreshToken      = "refresh_token"

	OAuthConsentSourceExplicit   = "explicit"
	OAuthConsentSourceTrusted    = "trusted"
	OAuthConsentSourcePromptNone = "prompt_none"

	OAuthConsentStatusPending = "pending"
	OAuthConsentStatusGranted = "granted"
	OAuthConsentStatusDenied  = "denied"

	OAuthConsentDecisionAllow = "allow"
	OAuthConsentDecisionDeny  = "deny"

	CodeChallengeMethodS256 = "S256"

	OAuthFlowPurposeLogin = "login"
	OAuthFlowPurposeLink  = "link"

	ScopeOpenID        = "openid"
	ScopeProfile       = "profile"
	ScopeEmail         = "email"
	ScopeOfflineAccess = "offline_access"
)

// OAuthClient is a registered OAuth 2.0 client. Client secrets and other
// one-time credentials are never stored in plain text.
type OAuthClient struct {
	bun.BaseModel `bun:"table:oauth_clients"`

	ID                       string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	ClientID                 string     `json:"client_id" bun:"column:client_id,unique" required:"true" nullable:"false"`
	ClientSecretHash         string     `json:"-" bun:"column:client_secret_hash" required:"true" nullable:"false"`
	ClientSecretSalt         string     `json:"-" bun:"column:client_secret_salt" required:"true" nullable:"false"`
	ClientSecretPrefix       string     `json:"client_secret_prefix" bun:"column:client_secret_prefix" required:"true" nullable:"false"`
	Name                     string     `json:"name" bun:"column:name" required:"true" nullable:"false"`
	RedirectURIs             []string   `json:"redirect_uris" bun:"column:redirect_uris" required:"true" nullable:"false"`
	PostLogoutRedirectURIs   []string   `json:"post_logout_redirect_uris" bun:"column:post_logout_redirect_uris" required:"true" nullable:"false"`
	AllowedGrants            []string   `json:"allowed_grants" bun:"column:allowed_grants" required:"true" nullable:"false"`
	AllowedScopes            []string   `json:"allowed_scopes" bun:"column:allowed_scopes" required:"true" nullable:"false"`
	AllowedAudiences         []string   `json:"allowed_audiences" bun:"column:allowed_audiences" required:"true" nullable:"false"`
	OwnerType                string     `json:"owner_type" bun:"column:owner_type" required:"true" nullable:"false"`
	OwnerID                  string     `json:"owner_id" bun:"column:owner_id" required:"true" nullable:"false"`
	CreatorID                string     `json:"creator_id" bun:"column:creator_id" required:"true" nullable:"false"`
	OrganizationID           *string    `json:"organization_id" bun:"column:organization_id" nullable:"true"`
	ApplicationType          string     `json:"application_type" bun:"column:application_type" required:"true" nullable:"false"`
	AuthMethod               string     `json:"auth_method" bun:"column:auth_method" required:"true" nullable:"false"`
	TrustedFirstParty        bool       `json:"trusted_first_party" bun:"column:trusted_first_party" required:"true" nullable:"false"`
	RequireConsent           bool       `json:"require_consent" bun:"column:require_consent" required:"true" nullable:"false"`
	AccessTokenType          string     `json:"access_token_type" bun:"column:access_token_type" required:"true" nullable:"false"`
	AccessTokenLifetime      int        `json:"access_token_lifetime" bun:"column:access_token_lifetime" required:"true" nullable:"false"`
	IDTokenLifetime          int        `json:"id_token_lifetime" bun:"column:id_token_lifetime" required:"true" nullable:"false"`
	LoginURL                 *string    `json:"login_url" bun:"column:login_url" nullable:"true"`
	JWKSURI                  *string    `json:"jwks_uri" bun:"column:jwks_uri" nullable:"true"`
	TokenEndpointAuthSignAlg *string    `json:"token_endpoint_auth_signing_alg" bun:"column:token_endpoint_auth_signing_alg" nullable:"true"`
	IsActive                 bool       `json:"is_active" bun:"column:is_active" required:"true" nullable:"false"`
	DeletedAt                *time.Time `json:"deleted_at,omitempty" bun:"column:deleted_at,nullzero" nullable:"true"`
	CreatedAt                time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
	UpdatedAt                time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthAuthRequest preserves an authorization request for the lifetime of the
// authorization-code flow. The authorization code itself is stored only as a
// hash and may be consumed exactly once.
type OAuthAuthRequest struct {
	bun.BaseModel `bun:"table:oauth_auth_requests"`

	ID                  string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	ClientID            string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	UserID              *string    `json:"user_id" bun:"column:user_id" nullable:"true"`
	SessionID           *string    `json:"session_id" bun:"column:session_id" nullable:"true"`
	RedirectURI         string     `json:"redirect_uri" bun:"column:redirect_uri" required:"true" nullable:"false"`
	Scopes              string     `json:"scopes" bun:"column:scopes" required:"true" nullable:"false"`
	Audience            *string    `json:"audience" bun:"column:audience" nullable:"true"`
	State               *string    `json:"state" bun:"column:state" nullable:"true"`
	Nonce               *string    `json:"nonce" bun:"column:nonce" nullable:"true"`
	ResponseType        string     `json:"response_type" bun:"column:response_type" required:"true" nullable:"false"`
	ResponseMode        *string    `json:"response_mode" bun:"column:response_mode" nullable:"true"`
	CodeChallenge       string     `json:"code_challenge" bun:"column:code_challenge" required:"true" nullable:"false"`
	CodeChallengeMethod string     `json:"code_challenge_method" bun:"column:code_challenge_method" required:"true" nullable:"false"`
	Prompt              string     `json:"prompt" bun:"column:prompt" nullable:"true"`
	MaxAge              *int       `json:"max_age" bun:"column:max_age" nullable:"true"`
	UILocales           *string    `json:"ui_locales" bun:"column:ui_locales" nullable:"true"`
	LoginHint           *string    `json:"login_hint" bun:"column:login_hint" nullable:"true"`
	ACRValues           *string    `json:"acr_values" bun:"column:acr_values" nullable:"true"`
	AuthTime            *time.Time `json:"auth_time" bun:"column:auth_time" nullable:"true"`
	AMR                 []string   `json:"amr" bun:"column:amr" nullable:"true"`
	ACR                 *string    `json:"acr" bun:"column:acr" nullable:"true"`
	BindingHash         string     `json:"-" bun:"column:binding_hash,unique" required:"true" nullable:"false"`
	ConsentStatus       string     `json:"consent_status" bun:"column:consent_status" required:"true" nullable:"false"`
	CodeHash            *string    `json:"-" bun:"column:code_hash,unique" nullable:"true"`
	CompletedAt         *time.Time `json:"completed_at,omitempty" bun:"column:completed_at,nullzero" nullable:"true"`
	CodeIssuedAt        *time.Time `json:"code_issued_at,omitempty" bun:"column:code_issued_at,nullzero" nullable:"true"`
	CodeConsumedAt      *time.Time `json:"code_consumed_at,omitempty" bun:"column:code_consumed_at,nullzero" nullable:"true"`
	CreatedAt           time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
	ExpiresAt           time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
}

// OAuthConsentRequest is a one-time consent challenge. The raw challenge token
// and completion code are only known to the caller; only hashes are stored.
type OAuthConsentRequest struct {
	bun.BaseModel `bun:"table:oauth_consent_requests"`

	ID                 string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	TokenHash          string     `json:"-" bun:"column:token_hash,unique" required:"true" nullable:"false"`
	AuthRequestID      string     `json:"auth_request_id" bun:"column:auth_request_id" required:"true" nullable:"false"`
	UserID             string     `json:"user_id" bun:"column:user_id" required:"true" nullable:"false"`
	SessionID          string     `json:"session_id" bun:"column:session_id" required:"true" nullable:"false"`
	ClientID           string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	RequestedScopes    string     `json:"requested_scopes" bun:"column:requested_scopes" required:"true" nullable:"false"`
	Decision           *string    `json:"decision" bun:"column:decision" nullable:"true"`
	GrantedScopes      *string    `json:"granted_scopes" bun:"column:granted_scopes" nullable:"true"`
	CompletionCodeHash *string    `json:"-" bun:"column:completion_code_hash,unique" nullable:"true"`
	Origin             *string    `json:"origin" bun:"column:origin" nullable:"true"`
	ExpiresAt          time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	ConsumedAt         *time.Time `json:"consumed_at,omitempty" bun:"column:consumed_at,nullzero" nullable:"true"`
	CreatedAt          time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
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

// OAuthAccessGrant records an issued access token so it can be revoked or
// introspected. JWT access tokens remain verifiable offline, but this record is
// the revocation source of truth.
type OAuthAccessGrant struct {
	bun.BaseModel `bun:"table:oauth_access_grants"`

	ID        string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	JTI       string     `json:"jti" bun:"column:jti,unique" required:"true" nullable:"false"`
	ClientID  string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	UserID    string     `json:"user_id" bun:"column:user_id" required:"true" nullable:"false"`
	SessionID string     `json:"session_id" bun:"column:session_id" required:"true" nullable:"false"`
	FamilyID  string     `json:"family_id" bun:"column:family_id" required:"true" nullable:"false"`
	Scopes    string     `json:"scopes" bun:"column:scopes" required:"true" nullable:"false"`
	Audience  *string    `json:"audience" bun:"column:audience" nullable:"true"`
	AuthTime  *time.Time `json:"auth_time" bun:"column:auth_time" nullable:"true"`
	AMR       []string   `json:"amr" bun:"column:amr" nullable:"true"`
	ExpiresAt time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" bun:"column:revoked_at,nullzero" nullable:"true"`
	CreatedAt time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
}

// OAuthRefreshToken is a rotating refresh token. Only the SHA-256 hash is
// stored. Tokens are grouped by family so reuse can revoke the whole lineage.
type OAuthRefreshToken struct {
	bun.BaseModel `bun:"table:oauth_refresh_tokens"`

	ID            string     `json:"id" bun:"column:id,pk" required:"true" nullable:"false"`
	TokenHash     string     `json:"-" bun:"column:token_hash,unique" required:"true" nullable:"false"`
	FamilyID      string     `json:"family_id" bun:"column:family_id" required:"true" nullable:"false"`
	ClientID      string     `json:"client_id" bun:"column:client_id" required:"true" nullable:"false"`
	UserID        string     `json:"user_id" bun:"column:user_id" required:"true" nullable:"false"`
	SessionID     string     `json:"session_id" bun:"column:session_id" required:"true" nullable:"false"`
	AuthRequestID *string    `json:"auth_request_id" bun:"column:auth_request_id" nullable:"true"`
	Scopes        string     `json:"scopes" bun:"column:scopes" required:"true" nullable:"false"`
	Audience      *string    `json:"audience" bun:"column:audience" nullable:"true"`
	AuthTime      *time.Time `json:"auth_time" bun:"column:auth_time" nullable:"true"`
	AMR           []string   `json:"amr" bun:"column:amr" nullable:"true"`
	RotatedFrom   *string    `json:"rotated_from" bun:"column:rotated_from" nullable:"true"`
	ConsumedAt    *time.Time `json:"consumed_at,omitempty" bun:"column:consumed_at,nullzero" nullable:"true"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty" bun:"column:revoked_at,nullzero" nullable:"true"`
	ExpiresAt     time.Time  `json:"expires_at" bun:"column:expires_at" required:"true" nullable:"false"`
	CreatedAt     time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp" required:"true" nullable:"false"`
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
