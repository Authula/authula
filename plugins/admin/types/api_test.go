package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/admin/constants"
)

type testValidator interface {
	Validate() error
}

func TestCreateUserRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreateUserRequest
		wantErr bool
	}{
		{
			name:    "accepts_required_fields",
			request: CreateUserRequest{Name: "John Doe", Email: "john.doe@example.com"},
		},
		{
			name: "accepts_all_optional_fields",
			request: CreateUserRequest{
				Name:          "John Doe",
				Email:         "john.doe@example.com",
				EmailVerified: new(true),
				Image:         new("/avatar.png"),
				Metadata:      map[string]any{"plan": "pro"},
			},
		},
		{
			name:    "rejects_missing_name",
			request: CreateUserRequest{Email: "john.doe@example.com"},
			wantErr: true,
		},
		{
			name:    "rejects_missing_email",
			request: CreateUserRequest{Name: "John Doe"},
			wantErr: true,
		},
		{
			name:    "rejects_invalid_email",
			request: CreateUserRequest{Name: "John Doe", Email: "not-an-email"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateUserRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request UpdateUserRequest
		wantIs  error
		wantErr bool
	}{
		{
			name:    "rejects_a_request_without_properties",
			request: UpdateUserRequest{},
			wantIs:  constants.ErrNoPropertiesProvided,
		},
		{
			name:    "accepts_name_only",
			request: UpdateUserRequest{Name: new("John Doe")},
		},
		{
			name:    "accepts_valid_email_only",
			request: UpdateUserRequest{Email: new("john.doe@example.com")},
		},
		{
			name:    "rejects_invalid_email",
			request: UpdateUserRequest{Email: new("not-an-email")},
			wantErr: true,
		},
		{
			name:    "accepts_image_only",
			request: UpdateUserRequest{Image: new("/avatar.png")},
		},
		{
			name:    "accepts_metadata_only",
			request: UpdateUserRequest{Metadata: map[string]any{"plan": "pro"}},
		},
		{
			name:    "accepts_email_verified_only",
			request: UpdateUserRequest{EmailVerified: new(false)},
		},
		{
			name:    "accepts_blank_name_because_it_is_optional",
			request: UpdateUserRequest{Name: new("")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				return
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCreateAccountRequest_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		request CreateAccountRequest
		wantIs  error
		wantErr bool
	}{
		{
			name:    "accepts_required_fields",
			request: CreateAccountRequest{ProviderID: "credential", AccountID: "account-1"},
		},
		{
			name: "accepts_all_optional_fields",
			request: CreateAccountRequest{
				ProviderID:            "credential",
				AccountID:             "account-1",
				AccessToken:           new("access-token"),
				RefreshToken:          new("refresh-token"),
				IDToken:               new("id-token"),
				AccessTokenExpiresAt:  new(now.Add(time.Hour)),
				RefreshTokenExpiresAt: new(now.Add(2 * time.Hour)),
				Scope:                 new("openid"),
				Password:              new("hashed-password"),
			},
		},
		{
			name:    "rejects_missing_provider_id",
			request: CreateAccountRequest{AccountID: "account-1"},
			wantErr: true,
		},
		{
			name:    "rejects_missing_account_id",
			request: CreateAccountRequest{ProviderID: "credential"},
			wantErr: true,
		},
		{
			name: "rejects_access_token_expiry_in_the_past",
			request: CreateAccountRequest{
				ProviderID:           "credential",
				AccountID:            "account-1",
				AccessTokenExpiresAt: new(now.Add(-time.Hour)),
			},
			wantIs: constants.ErrAccessTokenExpiresAtBeforeNow,
		},
		{
			name: "rejects_refresh_token_expiry_in_the_past",
			request: CreateAccountRequest{
				ProviderID:            "credential",
				AccountID:             "account-1",
				RefreshTokenExpiresAt: new(now.Add(-time.Hour)),
			},
			wantIs: constants.ErrRefreshTokenExpiresAtBeforeNow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				return
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateAccountRequest_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		request UpdateAccountRequest
		wantIs  error
	}{
		{
			name:    "rejects_a_request_without_properties",
			request: UpdateAccountRequest{},
			wantIs:  constants.ErrNoPropertiesProvided,
		},
		{
			name:    "accepts_provider_id",
			request: UpdateAccountRequest{ProviderID: new("credential")},
		},
		{
			name:    "accepts_account_id",
			request: UpdateAccountRequest{AccountID: new("account-1")},
		},
		{
			name:    "accepts_access_token",
			request: UpdateAccountRequest{AccessToken: new("access-token")},
		},
		{
			name:    "accepts_refresh_token",
			request: UpdateAccountRequest{RefreshToken: new("refresh-token")},
		},
		{
			name:    "accepts_id_token",
			request: UpdateAccountRequest{IDToken: new("id-token")},
		},
		{
			name:    "accepts_access_token_expiry",
			request: UpdateAccountRequest{AccessTokenExpiresAt: new(now.Add(time.Hour))},
		},
		{
			name:    "accepts_refresh_token_expiry",
			request: UpdateAccountRequest{RefreshTokenExpiresAt: new(now.Add(time.Hour))},
		},
		{
			name:    "accepts_scope",
			request: UpdateAccountRequest{Scope: new("openid")},
		},
		{
			name:    "accepts_password",
			request: UpdateAccountRequest{Password: new("hashed-password")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUserStateRequests_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		request testValidator
		wantIs  error
		wantErr bool
	}{
		{
			name:    "create_accepts_banned_without_expiry",
			request: &CreateUserStateRequest{Banned: new(true)},
		},
		{
			name:    "create_accepts_banned_with_future_expiry",
			request: &CreateUserStateRequest{Banned: new(true), BannedUntil: new(now.Add(time.Hour))},
		},
		{
			name:    "create_rejects_banned_with_past_expiry",
			request: &CreateUserStateRequest{Banned: new(true), BannedUntil: new(now.Add(-time.Hour))},
			wantIs:  constants.ErrBannedUntilBeforeNow,
		},
		{
			name:    "create_accepts_explicit_unbanned",
			request: &CreateUserStateRequest{Banned: new(false)},
		},
		{
			name:    "create_rejects_unbanned_zero_value",
			request: &CreateUserStateRequest{},
			wantErr: true,
		},
		{
			name:    "upsert_accepts_banned_without_expiry",
			request: &UpsertUserStateRequest{Banned: new(true)},
		},
		{
			name:    "upsert_accepts_banned_with_future_expiry",
			request: &UpsertUserStateRequest{Banned: new(true), BannedUntil: new(now.Add(time.Hour))},
		},
		{
			name:    "upsert_rejects_banned_with_past_expiry",
			request: &UpsertUserStateRequest{Banned: new(true), BannedUntil: new(now.Add(-time.Hour))},
			wantIs:  constants.ErrBannedUntilBeforeNow,
		},
		{
			name:    "upsert_rejects_unbanned_zero_value",
			request: &UpsertUserStateRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				return
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBanUserRequest_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		request BanUserRequest
		wantIs  error
	}{
		{
			name:    "accepts_no_expiry",
			request: BanUserRequest{},
		},
		{
			name:    "accepts_future_expiry",
			request: BanUserRequest{BannedUntil: new(now.Add(time.Hour))},
		},
		{
			name:    "rejects_past_expiry",
			request: BanUserRequest{BannedUntil: new(now.Add(-time.Hour))},
			wantIs:  constants.ErrBannedUntilBeforeNow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSessionStateRequests_Validate(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		request testValidator
		wantIs  error
		wantErr bool
	}{
		{
			name:    "create_accepts_revoked_session",
			request: &CreateSessionStateRequest{Revoke: new(true)},
		},
		{
			name: "create_accepts_future_impersonation_expiry",
			request: &CreateSessionStateRequest{
				Revoke:                 new(true),
				ImpersonationExpiresAt: new(now.Add(time.Hour)),
			},
		},
		{
			name: "create_rejects_past_impersonation_expiry",
			request: &CreateSessionStateRequest{
				Revoke:                 new(true),
				ImpersonationExpiresAt: new(now.Add(-time.Hour)),
			},
			wantIs: constants.ErrImpersonationExpiresAtBeforeNow,
		},
		{
			name:    "create_accepts_explicit_not_revoked",
			request: &CreateSessionStateRequest{Revoke: new(false)},
		},
		{
			name:    "create_rejects_zero_value",
			request: &CreateSessionStateRequest{},
			wantErr: true,
		},
		{
			name:    "upsert_accepts_revoked_session",
			request: &UpsertSessionStateRequest{Revoke: new(true)},
		},
		{
			name: "upsert_accepts_future_impersonation_expiry",
			request: &UpsertSessionStateRequest{
				Revoke:                 new(true),
				ImpersonationExpiresAt: new(now.Add(time.Hour)),
			},
		},
		{
			name: "upsert_rejects_past_impersonation_expiry",
			request: &UpsertSessionStateRequest{
				Revoke:                 new(true),
				ImpersonationExpiresAt: new(now.Add(-time.Hour)),
			},
			wantIs: constants.ErrImpersonationExpiresAtBeforeNow,
		},
		{
			name:    "upsert_rejects_zero_value",
			request: &UpsertSessionStateRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				return
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStartImpersonationRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request StartImpersonationRequest
		wantErr bool
	}{
		{
			name:    "accepts_required_fields",
			request: StartImpersonationRequest{TargetUserID: "user-2", Reason: "support"},
		},
		{
			name: "accepts_expiry_override",
			request: StartImpersonationRequest{
				TargetUserID:     "user-2",
				Reason:           "support",
				ExpiresInSeconds: new(300),
			},
		},
		{
			name:    "rejects_missing_target_user",
			request: StartImpersonationRequest{Reason: "support"},
			wantErr: true,
		},
		{
			name:    "rejects_missing_reason",
			request: StartImpersonationRequest{TargetUserID: "user-2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestImpersonation_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := Impersonation{
		ID:                     "impersonation-1",
		ActorUserID:            "admin-1",
		TargetUserID:           "user-1",
		ActorSessionID:         new("session-admin"),
		ImpersonationSessionID: new("session-impersonated"),
		Reason:                 "support",
		StartedAt:              startedAt,
		ExpiresAt:              startedAt.Add(15 * time.Minute),
		EndedAt:                new(startedAt.Add(10 * time.Minute)),
		EndedByUserID:          new("admin-1"),
		CreatedAt:              startedAt,
		UpdatedAt:              startedAt,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got Impersonation
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestAdminUserState_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	bannedAt := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := AdminUserState{
		UserID:         "user-1",
		Banned:         true,
		BannedAt:       new(bannedAt),
		BannedUntil:    new(bannedAt.Add(time.Hour)),
		BannedReason:   new("abuse"),
		BannedByUserID: new("admin-1"),
		CreatedAt:      bannedAt,
		UpdatedAt:      bannedAt,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got AdminUserState
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestAdminSessionState_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	revokedAt := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := AdminSessionState{
		SessionID:              "session-1",
		RevokedAt:              new(revokedAt),
		RevokedReason:          new("logout"),
		RevokedByUserID:        new("admin-1"),
		ImpersonatorUserID:     new("admin-1"),
		ImpersonatorSessionID:  new("session-admin"),
		ImpersonationReason:    new("support"),
		ImpersonationExpiresAt: new(revokedAt.Add(time.Hour)),
		CreatedAt:              revokedAt,
		UpdatedAt:              revokedAt,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got AdminSessionState
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestAdminUserSession_JSONOmitsEmptyState(t *testing.T) {
	t.Parallel()

	session := models.Session{ID: "session-1", UserID: "user-1", Token: "token"}

	withoutState := AdminUserSession{Session: session}
	data, err := json.Marshal(withoutState)
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	require.NotContains(t, fields, "state")

	withState := AdminUserSession{Session: session, State: &AdminSessionState{SessionID: "session-1"}}
	data, err = json.Marshal(withState)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(data, &fields))
	require.Contains(t, fields, "state")
}

func TestStartImpersonationResult_JSONOmitsInternalFields(t *testing.T) {
	t.Parallel()

	result := StartImpersonationResult{
		Impersonation:        &Impersonation{ID: "impersonation-1"},
		ImpersonatorUserID:   "admin-1",
		ImpersonatorScopes:   []string{"admin:*"},
		OriginalCookieToken:  "original-token",
		OriginalCookieMaxAge: 3600,
		TargetUserID:         "user-1",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	require.Equal(t, []string{"impersonation"}, mapKeys(fields))

	result.SessionID = new("session-1")
	result.SessionToken = new("session-token")
	data, err = json.Marshal(result)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(data, &fields))
	require.ElementsMatch(t, []string{"impersonation", "session_id", "session_token"}, mapKeys(fields))
}

func TestCreateUserRequest_JSONOmitempty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreateUserRequest
		want    string
	}{
		{
			name:    "omits_unset_optionals",
			request: CreateUserRequest{Name: "John", Email: "john.doe@example.com"},
			want:    `{"name":"John","email":"john.doe@example.com"}`,
		},
		{
			name: "includes_populated_optionals",
			request: CreateUserRequest{
				Name:          "John",
				Email:         "john.doe@example.com",
				EmailVerified: new(true),
				Image:         new("/avatar.png"),
				Metadata:      map[string]any{"plan": "pro"},
			},
			want: `{"name":"John","email":"john.doe@example.com","email_verified":true,"image":"/avatar.png","metadata":{"plan":"pro"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.request)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(data))
		})
	}
}

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
