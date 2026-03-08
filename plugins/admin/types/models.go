package types

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

type Impersonation struct {
	bun.BaseModel `bun:"table:admin_impersonations"`

	ID                     string     `json:"id" bun:"column:id,pk"`
	ActorUserID            string     `json:"actor_user_id" bun:"column:actor_user_id"`
	TargetUserID           string     `json:"target_user_id" bun:"column:target_user_id"`
	ActorSessionID         *string    `json:"actor_session_id" bun:"column:actor_session_id"`
	ImpersonationSessionID *string    `json:"impersonation_session_id" bun:"column:impersonation_session_id"`
	Reason                 string     `json:"reason" bun:"column:reason"`
	StartedAt              time.Time  `json:"started_at" bun:"column:started_at"`
	ExpiresAt              time.Time  `json:"expires_at" bun:"column:expires_at"`
	EndedAt                *time.Time `json:"ended_at" bun:"column:ended_at"`
	EndedByUserID          *string    `json:"ended_by_user_id" bun:"column:ended_by_user_id"`
	CreatedAt              time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserState struct {
	bun.BaseModel `bun:"table:admin_user_states"`

	UserID         string     `json:"user_id" bun:"column:user_id,pk"`
	IsBanned       bool       `json:"banned" bun:"column:banned"`
	BannedAt       *time.Time `json:"banned_at" bun:"column:banned_at"`
	BannedUntil    *time.Time `json:"banned_until" bun:"column:banned_until"`
	BannedReason   *string    `json:"banned_reason" bun:"column:banned_reason"`
	BannedByUserID *string    `json:"banned_by_user_id" bun:"column:banned_by_user_id"`
	CreatedAt      time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt      time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminSessionState struct {
	bun.BaseModel `bun:"table:admin_session_states"`

	SessionID              string     `json:"session_id" bun:"column:session_id,pk"`
	RevokedAt              *time.Time `json:"revoked_at" bun:"column:revoked_at"`
	RevokedReason          *string    `json:"revoked_reason" bun:"column:revoked_reason"`
	RevokedByUserID        *string    `json:"revoked_by_user_id" bun:"column:revoked_by_user_id"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id" bun:"column:impersonator_user_id"`
	ImpersonationReason    *string    `json:"impersonation_reason" bun:"column:impersonation_reason"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at" bun:"column:impersonation_expires_at"`
	CreatedAt              time.Time  `json:"created_at" bun:"column:created_at,default:current_timestamp"`
	UpdatedAt              time.Time  `json:"updated_at" bun:"column:updated_at,default:current_timestamp"`
}

type AdminUserSession struct {
	Session models.Session     `json:"session"`
	State   *AdminSessionState `json:"state,omitempty"`
}

type CreateUserRequest struct {
	Name          string          `json:"name"`
	Email         string          `json:"email"`
	EmailVerified *bool           `json:"email_verified,omitempty"`
	Image         *string         `json:"image,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type CreateUserResponse struct {
	User *models.User `json:"user"`
}

type GetUserByIDResponse struct {
	User *models.User `json:"user"`
}

type UpdateUserRequest struct {
	Name          *string         `json:"name,omitempty"`
	Email         *string         `json:"email,omitempty"`
	EmailVerified *bool           `json:"email_verified,omitempty"`
	Image         *string         `json:"image,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type UpdateUserResponse struct {
	User *models.User `json:"user"`
}

type DeleteUserResponse struct {
	Message string `json:"message"`
}

type UsersPage struct {
	Users      []models.User `json:"users"`
	NextCursor *string       `json:"next_cursor,omitempty"`
}

type CreateAccountRequest struct {
	ProviderID            string     `json:"provider_id"`
	AccountID             string     `json:"account_id"`
	AccessToken           *string    `json:"access_token,omitempty"`
	RefreshToken          *string    `json:"refresh_token,omitempty"`
	IDToken               *string    `json:"id_token,omitempty"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty"`
	Scope                 *string    `json:"scope,omitempty"`
	Password              *string    `json:"password,omitempty"`
}

type UpdateAccountRequest struct {
	ProviderID            *string    `json:"provider_id,omitempty"`
	AccountID             *string    `json:"account_id,omitempty"`
	AccessToken           *string    `json:"access_token,omitempty"`
	RefreshToken          *string    `json:"refresh_token,omitempty"`
	IDToken               *string    `json:"id_token,omitempty"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty"`
	Scope                 *string    `json:"scope,omitempty"`
	Password              *string    `json:"password,omitempty"`
}

type CreateAccountResponse struct {
	Account *models.Account `json:"account"`
}

type GetAccountByIDResponse struct {
	Account *models.Account `json:"account"`
}

type UpdateAccountResponse struct {
	Account *models.Account `json:"account"`
}

type DeleteAccountResponse struct {
	Message string `json:"message"`
}

type UserAccountsResponse struct {
	Accounts []models.Account `json:"accounts"`
}

type GetUserStateResponse struct {
	State *AdminUserState `json:"state"`
}

type UpsertUserStateResponse struct {
	State *AdminUserState `json:"state"`
}

type UpsertUserStateRequest struct {
	IsBanned     bool       `json:"banned"`
	BannedUntil  *time.Time `json:"banned_until,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
}

type DeleteUserStateResponse struct {
	Message string `json:"message"`
}

type BanUserRequest struct {
	BannedUntil *time.Time `json:"banned_until,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
}

type BanUserResponse struct {
	State *AdminUserState `json:"state"`
}

type UnbanUserResponse struct {
	State *AdminUserState `json:"state"`
}

type GetSessionStateResponse struct {
	State *AdminSessionState `json:"state"`
}

type UpsertSessionStateRequest struct {
	Revoke                 bool       `json:"revoke"`
	RevokedReason          *string    `json:"revoked_reason,omitempty"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty"`
}

type UpsertSessionStateResponse struct {
	State *AdminSessionState `json:"state"`
}

type DeleteSessionStateResponse struct {
	Message string `json:"message"`
}

type RevokeSessionRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type RevokeSessionResponse struct {
	State *AdminSessionState `json:"state"`
}

type GetImpersonationByIDResponse struct {
	Impersonation *Impersonation `json:"impersonation"`
}

type StartImpersonationRequest struct {
	TargetUserID     string `json:"target_user_id"`
	Reason           string `json:"reason"`
	ExpiresInSeconds *int   `json:"expires_in_seconds,omitempty"`
}

type StartImpersonationResult struct {
	Impersonation *Impersonation `json:"impersonation"`
	SessionID     *string        `json:"session_id,omitempty"`
	SessionToken  *string        `json:"session_token,omitempty"`
}

type StartImpersonationResponse struct {
	Impersonation *Impersonation `json:"impersonation"`
}

type StopImpersonationRequest struct {
	ImpersonationID *string `json:"impersonation_id,omitempty"`
}

type StopImpersonationResponse struct {
	Message string `json:"message"`
}
