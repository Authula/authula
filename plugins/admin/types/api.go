package types

import (
	"time"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/constants"
)

type CreateUserRequest struct {
	Name          string         `json:"name"`
	Email         string         `json:"email"`
	EmailVerified *bool          `json:"email_verified,omitempty"`
	Image         *string        `json:"image,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type CreateUserResponse struct {
	User *models.User `json:"user"`
}

type GetUserByIDResponse struct {
	User *models.User `json:"user"`
}

type UpdateUserRequest struct {
	Name          *string        `json:"name,omitempty"`
	Email         *string        `json:"email,omitempty"`
	EmailVerified *bool          `json:"email_verified,omitempty"`
	Image         *string        `json:"image,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
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

func (req *CreateAccountRequest) Validate() error {
	if req.ProviderID == "" {
		return constants.ErrProviderIDRequired
	}
	if req.AccountID == "" {
		return constants.ErrAccountIDRequired
	}
	if req.AccessTokenExpiresAt != nil && req.AccessTokenExpiresAt.Before(time.Now()) {
		return constants.ErrAccessTokenExpiresAtBeforeNow
	}
	if req.RefreshTokenExpiresAt != nil && req.RefreshTokenExpiresAt.Before(time.Now()) {
		return constants.ErrRefreshTokenExpiresAtBeforeNow
	}
	return nil
}

type CreateAccountResponse struct {
	Account *models.Account `json:"account"`
}

type GetAccountByIDResponse struct {
	Account *models.Account `json:"account"`
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

func (req *UpdateAccountRequest) Validate() error {
	if req.ProviderID == nil &&
		req.AccountID == nil &&
		req.AccessToken == nil &&
		req.RefreshToken == nil &&
		req.IDToken == nil &&
		req.AccessTokenExpiresAt == nil &&
		req.RefreshTokenExpiresAt == nil &&
		req.Scope == nil &&
		req.Password == nil {
		return constants.ErrNoPropertiesProvided
	}
	return nil
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

type CreateUserStateRequest struct {
	Banned       bool       `json:"banned"`
	BannedUntil  *time.Time `json:"banned_until,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
}

type UpsertUserStateRequest struct {
	Banned       bool       `json:"banned"`
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

type CreateSessionStateRequest struct {
	Revoke                 bool       `json:"revoke"`
	RevokedReason          *string    `json:"revoked_reason,omitempty"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty"`
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
