package types

import (
	"time"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/constants"
)

type AdminUserID struct {
	UserID string `path:"user_id" json:"user_id" required:"true" nullable:"false"`
}

type AdminAccountID struct {
	ID string `path:"id" json:"id" required:"true" nullable:"false"`
}

type AdminSessionID struct {
	SessionID string `path:"session_id" json:"session_id" required:"true" nullable:"false"`
}

type AdminImpersonationID struct {
	ImpersonationID string `path:"impersonation_id" json:"impersonation_id" required:"true" nullable:"false"`
}

type ListUsersQuery struct {
	Cursor *string `query:"cursor" json:"cursor,omitempty" nullable:"true"`
	Limit  *int    `query:"limit" json:"limit,omitempty" nullable:"true"`
}

type CreateUserRequest struct {
	Name          string         `json:"name" required:"true" nullable:"false" validate:"required"`
	Email         string         `json:"email" required:"true" nullable:"false" validate:"required,email"`
	EmailVerified *bool          `json:"email_verified,omitempty" nullable:"true"`
	Image         *string        `json:"image,omitempty" nullable:"true"`
	Metadata      map[string]any `json:"metadata,omitempty" nullable:"true"`
}

func (req *CreateUserRequest) Validate() error {
	return util.ValidateStruct(req)
}

type CreateUserResponse struct {
	User *models.User `json:"user" required:"true" nullable:"false"`
}

type GetUserByIDResponse struct {
	User *models.User `json:"user" required:"true" nullable:"false"`
}

type UpdateUserRequest struct {
	Name          *string        `json:"name,omitempty" nullable:"true" validate:"omitempty"`
	Email         *string        `json:"email,omitempty" nullable:"true" validate:"omitempty,email"`
	EmailVerified *bool          `json:"email_verified,omitempty" nullable:"true"`
	Image         *string        `json:"image,omitempty" nullable:"true"`
	Metadata      map[string]any `json:"metadata,omitempty" nullable:"true"`
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
	User *models.User `json:"user" required:"true" nullable:"false"`
}

type DeleteUserResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type UsersPage struct {
	Users      []models.User `json:"users" required:"true" nullable:"false"`
	NextCursor *string       `json:"next_cursor,omitempty" nullable:"true"`
}

type CreateAccountRequest struct {
	ProviderID            string     `json:"provider_id" required:"true" nullable:"false" validate:"required"`
	AccountID             string     `json:"account_id" required:"true" nullable:"false" validate:"required"`
	AccessToken           *string    `json:"access_token,omitempty" nullable:"true"`
	RefreshToken          *string    `json:"refresh_token,omitempty" nullable:"true"`
	IDToken               *string    `json:"id_token,omitempty" nullable:"true"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty" nullable:"true"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty" nullable:"true"`
	Scope                 *string    `json:"scope,omitempty" nullable:"true"`
	Password              *string    `json:"password,omitempty" nullable:"true"`
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
	Account *models.Account `json:"account" required:"true" nullable:"false"`
}

type GetAccountByIDResponse struct {
	Account *models.Account `json:"account" required:"true" nullable:"false"`
}

type UpdateAccountRequest struct {
	ProviderID            *string    `json:"provider_id,omitempty" nullable:"true"`
	AccountID             *string    `json:"account_id,omitempty" nullable:"true"`
	AccessToken           *string    `json:"access_token,omitempty" nullable:"true"`
	RefreshToken          *string    `json:"refresh_token,omitempty" nullable:"true"`
	IDToken               *string    `json:"id_token,omitempty" nullable:"true"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty" nullable:"true"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty" nullable:"true"`
	Scope                 *string    `json:"scope,omitempty" nullable:"true"`
	Password              *string    `json:"password,omitempty" nullable:"true"`
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
	Account *models.Account `json:"account" required:"true" nullable:"false"`
}

type DeleteAccountResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type UserAccountsResponse struct {
	Accounts []models.Account `json:"accounts" required:"true" nullable:"false"`
}

type GetUserStateResponse struct {
	State *AdminUserState `json:"state" required:"true" nullable:"false"`
}

type UpsertUserStateResponse struct {
	State *AdminUserState `json:"state" required:"true" nullable:"false"`
}

type CreateUserStateRequest struct {
	Banned       bool       `json:"banned" required:"true" nullable:"false" validate:"required"`
	BannedUntil  *time.Time `json:"banned_until,omitempty" nullable:"true"`
	BannedReason *string    `json:"banned_reason,omitempty" nullable:"true"`
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
	Banned       bool       `json:"banned" required:"true" nullable:"false" validate:"required"`
	BannedUntil  *time.Time `json:"banned_until,omitempty" nullable:"true"`
	BannedReason *string    `json:"banned_reason,omitempty" nullable:"true"`
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
	Message string `json:"message" required:"true" nullable:"false"`
}

type BanUserRequest struct {
	BannedUntil *time.Time `json:"banned_until,omitempty" nullable:"true"`
	Reason      *string    `json:"reason,omitempty" nullable:"true"`
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
	State *AdminUserState `json:"state" required:"true" nullable:"false"`
}

type UnbanUserResponse struct {
	State *AdminUserState `json:"state" required:"true" nullable:"false"`
}

type GetSessionStateResponse struct {
	State *AdminSessionState `json:"state" required:"true" nullable:"false"`
}

type CreateSessionStateRequest struct {
	Revoke                 bool       `json:"revoke" required:"true" nullable:"false" validate:"required"`
	RevokedReason          *string    `json:"revoked_reason,omitempty" nullable:"true"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty" nullable:"true"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty" nullable:"true"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty" nullable:"true"`
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
	Revoke                 bool       `json:"revoke" required:"true" nullable:"false" validate:"required"`
	RevokedReason          *string    `json:"revoked_reason,omitempty" nullable:"true"`
	ImpersonatorUserID     *string    `json:"impersonator_user_id,omitempty" nullable:"true"`
	ImpersonationReason    *string    `json:"impersonation_reason,omitempty" nullable:"true"`
	ImpersonationExpiresAt *time.Time `json:"impersonation_expires_at,omitempty" nullable:"true"`
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
	State *AdminSessionState `json:"state" required:"true" nullable:"false"`
}

type DeleteSessionStateResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}

type RevokeSessionRequest struct {
	Reason *string `json:"reason,omitempty" nullable:"true"`
}

type RevokeSessionResponse struct {
	State *AdminSessionState `json:"state" required:"true" nullable:"false"`
}

type GetImpersonationByIDResponse struct {
	Impersonation *Impersonation `json:"impersonation" required:"true" nullable:"false"`
}

type StartImpersonationRequest struct {
	TargetUserID     string `json:"target_user_id" required:"true" nullable:"false" validate:"required"`
	Reason           string `json:"reason" required:"true" nullable:"false" validate:"required"`
	ExpiresInSeconds *int   `json:"expires_in_seconds,omitempty" nullable:"true"`
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
	Impersonation *Impersonation `json:"impersonation" required:"true" nullable:"false"`
}

type StopImpersonationRequest struct {
	ImpersonationID *string `json:"impersonation_id,omitempty"`
}

type StopImpersonationResult struct {
	OriginalSessionToken string `json:"-"`
}

type StopImpersonationResponse struct {
	Message string `json:"message" required:"true" nullable:"false"`
}
