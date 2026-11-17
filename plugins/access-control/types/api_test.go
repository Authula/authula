package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
)

func TestCreateRoleRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreateRoleRequest
		wantIs  error
	}{
		{
			name:    "accepts_named_role",
			request: CreateRoleRequest{Name: "editor"},
		},
		{
			name: "accepts_all_fields",
			request: CreateRoleRequest{
				Name:        "editor",
				Description: new("Can edit content"),
				Weight:      new(10),
				IsSystem:    true,
			},
		},
		{
			name:    "rejects_empty_name",
			request: CreateRoleRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
		},
		{
			name:    "accepts_whitespace_name",
			request: CreateRoleRequest{Name: " "},
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

func TestUpdateRoleRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request UpdateRoleRequest
		wantIs  error
	}{
		{
			name:    "rejects_a_request_without_properties",
			request: UpdateRoleRequest{},
			wantIs:  accesscontrolconstants.ErrAtleastOneFieldRequiredToUpdateResource,
		},
		{
			name:    "accepts_name",
			request: UpdateRoleRequest{Name: new("editor")},
		},
		{
			name:    "accepts_description",
			request: UpdateRoleRequest{Description: new("Can edit content")},
		},
		{
			name:    "accepts_weight",
			request: UpdateRoleRequest{Weight: new(10)},
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

func TestCreatePermissionRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreatePermissionRequest
		wantIs  error
	}{
		{
			name:    "accepts_key",
			request: CreatePermissionRequest{Key: "posts:read"},
		},
		{
			name:    "accepts_all_fields",
			request: CreatePermissionRequest{Key: "posts:read", Description: new("Read posts"), IsSystem: true},
		},
		{
			name:    "rejects_empty_key",
			request: CreatePermissionRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
		},
		{
			name:    "accepts_whitespace_key",
			request: CreatePermissionRequest{Key: " "},
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

func TestAddRolePermissionRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request AddRolePermissionRequest
		wantIs  error
	}{
		{
			name:    "accepts_permission_id",
			request: AddRolePermissionRequest{PermissionID: "permission-1"},
		},
		{
			name:    "rejects_empty_permission_id",
			request: AddRolePermissionRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
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

func TestReplaceRolePermissionsRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request ReplaceRolePermissionsRequest
		wantIs  error
	}{
		{
			name:    "accepts_permission_ids",
			request: ReplaceRolePermissionsRequest{PermissionIDs: []string{"permission-1", "permission-2"}},
		},
		{
			name:    "rejects_empty_list",
			request: ReplaceRolePermissionsRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
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

func TestAssignUserRoleRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request AssignUserRoleRequest
		wantIs  error
	}{
		{
			name:    "accepts_role_id",
			request: AssignUserRoleRequest{RoleID: "role-1"},
		},
		{
			name:    "accepts_role_with_expiry",
			request: AssignUserRoleRequest{RoleID: "role-1", ExpiresAt: new(time.Now().Add(time.Hour))},
		},
		{
			name:    "rejects_empty_role_id",
			request: AssignUserRoleRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
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

func TestReplaceUserRolesRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request ReplaceUserRolesRequest
		wantIs  error
	}{
		{
			name:    "accepts_role_ids",
			request: ReplaceUserRolesRequest{RoleIDs: []string{"role-1", "role-2"}},
		},
		{
			name:    "rejects_empty_list",
			request: ReplaceUserRolesRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
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

func TestCheckUserPermissionsRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CheckUserPermissionsRequest
		wantIs  error
	}{
		{
			name:    "accepts_permission_keys",
			request: CheckUserPermissionsRequest{PermissionKeys: []string{"posts:read"}},
		},
		{
			name:    "rejects_empty_list",
			request: CheckUserPermissionsRequest{},
			wantIs:  coreerrors.ErrUnprocessableEntity,
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

func TestRole_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := Role{
		ID:          "role-1",
		Name:        "editor",
		Description: new("Can edit content"),
		Weight:      10,
		IsSystem:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got Role
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestPermission_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := Permission{
		ID:          "permission-1",
		Key:         "posts:read",
		Description: new("Read posts"),
		IsSystem:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got Permission
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestRolePermission_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	grantedAt := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := RolePermission{
		RoleID:          "role-1",
		PermissionID:    "permission-1",
		GrantedByUserID: new("admin-1"),
		GrantedAt:       grantedAt,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got RolePermission
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestUserRole_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	assignedAt := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := UserRole{
		UserID:           "user-1",
		RoleID:           "role-1",
		AssignedByUserID: new("admin-1"),
		AssignedAt:       assignedAt,
		ExpiresAt:        new(assignedAt.Add(time.Hour)),
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got UserRole
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestUserPermissionInfo_JSONOmitsEmptySources(t *testing.T) {
	t.Parallel()

	info := UserPermissionInfo{
		PermissionID:  "permission-1",
		PermissionKey: "posts:read",
	}

	data, err := json.Marshal(info)
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	require.NotContains(t, fields, "sources")

	info.Sources = []PermissionGrantSource{{
		RoleID:   "role-1",
		RoleName: "editor",
	}}
	data, err = json.Marshal(info)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(data, &fields))
	require.Contains(t, fields, "sources")
}

func TestCreateRoleRequest_JSONOmitempty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreateRoleRequest
		want    string
	}{
		{
			name:    "omits_unset_optionals",
			request: CreateRoleRequest{Name: "editor"},
			want:    `{"name":"editor","is_system":false}`,
		},
		{
			name: "includes_populated_optionals",
			request: CreateRoleRequest{
				Name:        "editor",
				Description: new("Can edit content"),
				Weight:      new(10),
				IsSystem:    true,
			},
			want: `{"name":"editor","description":"Can edit content","weight":10,"is_system":true}`,
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
