package types

import (
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/constants"
)

type CreateUserRequest struct {
	Name          string         `json:"name" validate:"required"`
	Email         string         `json:"email" validate:"required,email"`
	EmailVerified *bool          `json:"email_verified,omitempty"`
	Image         *string        `json:"image,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (req *CreateUserRequest) Validate() error {
	return util.ValidateStruct(req)
}

type CreateUserResponse struct {
	User *models.User `json:"user"`
}

type GetUserByIDResponse struct {
	User *models.User `json:"user"`
}

type UpdateUserRequest struct {
	Name          *string        `json:"name,omitempty" validate:"omitempty"`
	Email         *string        `json:"email,omitempty" validate:"omitempty,email"`
	EmailVerified *bool          `json:"email_verified,omitempty"`
	Image         *string        `json:"image,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (req *UpdateUserRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
	}

	if req.Name == nil && req.Email == nil &&
		req.EmailVerified == nil && req.Image == nil &&
		req.Metadata == nil {
		return constants.ErrNoPropertiesProvided
	}

	return nil
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
	ProviderID            string     `json:"provider_id" validate:"required"`
	AccountID             string     `json:"account_id" validate:"required"`
	AccessToken           *string    `json:"access_token,omitempty"`
	RefreshToken          *string    `json:"refresh_token,omitempty"`
	IDToken               *string    `json:"id_token,omitempty"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty"`
	Scope                 *string    `json:"scope,omitempty"`
	Password              *string    `json:"password,omitempty"`
}

func (req *CreateAccountRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
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
	Banned       bool       `json:"banned" validate:"required"`
	BannedUntil  *time.Time `json:"banned_until,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
}

func (req *CreateUserStateRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
	}

	if req.Banned && req.BannedUntil != nil && req.BannedUntil.Before(time.Now()) {
		return constants.ErrBannedUntilBeforeNow
	}

	return nil
}

type UpsertUserStateRequest struct {
	Banned       bool       `json:"banned" validate:"required"`
	BannedUntil  *time.Time `json:"banned_until,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
}

func (req *UpsertUserStateRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
	}

	if req.Banned && req.BannedUntil != nil && req.BannedUntil.Before(time.Now()) {
		return constants.ErrBannedUntilBeforeNow
	}

	return nil
}

type DeleteUserStateResponse struct {
	Message string `json:"message"`
}

type BanUserRequest struct {
	BannedUntil *time.Time `json:"banned_until,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
}

func (req *BanUserRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
	}

	if req.BannedUntil != nil && req.BannedUntil.Before(time.Now()) {
		return constants.ErrBannedUntilBeforeNow
	}

	return nil
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
	Revoke                 bool       `json:"revoke" validate:"required"`
	RevokedReason          *string    `json:"revoked_reason,omitempty"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty"`
}

func (req *CreateSessionStateRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
	}

	if req.ImpersonationExpiresAt != nil && req.ImpersonationExpiresAt.Before(time.Now()) {
		return constants.ErrImpersonationExpiresAtBeforeNow
	}

	return nil
}

type UpsertSessionStateRequest struct {
	Revoke                 bool       `json:"revoke" validate:"required"`
	RevokedReason          *string    `json:"revoked_reason,omitempty"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty"`
}

func (req *UpsertSessionStateRequest) Validate() error {
	if err := util.ValidateStruct(req); err != nil {
		return err
	}

	if req.ImpersonationExpiresAt != nil && req.ImpersonationExpiresAt.Before(time.Now()) {
		return constants.ErrImpersonationExpiresAtBeforeNow
	}

	return nil
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
	TargetUserID     string `json:"target_user_id" validate:"required"`
	Reason           string `json:"reason" validate:"required"`
	ExpiresInSeconds *int   `json:"expires_in_seconds,omitempty"`
}

func (req *StartImpersonationRequest) Validate() error {
	return util.ValidateStruct(req)
}

type StartImpersonationResult struct {
	Impersonation        *Impersonation `json:"impersonation"`
	SessionID            *string        `json:"session_id,omitempty"`
	SessionToken         *string        `json:"session_token,omitempty"`
	ImpersonatorUserID   string         `json:"-"`
	ImpersonatorScopes   []string       `json:"-"`
	OriginalCookieToken  string         `json:"-"`
	OriginalCookieMaxAge int            `json:"-"`
	TargetUserID         string         `json:"-"`
}

type StartImpersonationResponse struct {
	Impersonation *Impersonation `json:"impersonation"`
}

type StopImpersonationRequest struct {
	ImpersonationID *string `json:"impersonation_id,omitempty"`
}

type StopImpersonationResult struct {
	OriginalSessionToken string `json:"-"`
}

type StopImpersonationResponse struct {
	Message string `json:"message"`
}
