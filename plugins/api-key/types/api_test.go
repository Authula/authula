package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateApiKeyRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreateApiKeyRequest
		wantErr string
	}{
		{
			name:    "accepts_user_owned_key",
			request: CreateApiKeyRequest{Name: "CI key", OwnerType: OwnerTypeUser, OwnerID: "user-1"},
		},
		{
			name:    "accepts_organization_owned_key",
			request: CreateApiKeyRequest{Name: "CI key", OwnerType: OwnerTypeOrganization, OwnerID: "org-1"},
		},
		{
			name: "accepts_all_optional_fields",
			request: CreateApiKeyRequest{
				Name:                 "CI key",
				OwnerType:            OwnerTypeUser,
				OwnerID:              "user-1",
				Prefix:               new("ci"),
				Enabled:              new(true),
				ExpiresAt:            new(time.Now().Add(time.Hour)),
				RateLimitEnabled:     new(true),
				RateLimitTimeWindow:  new(60),
				RateLimitMaxRequests: new(100),
				Permissions:          []string{"read"},
				Metadata:             map[string]any{"env": "ci"},
			},
		},
		{
			name:    "rejects_blank_name",
			request: CreateApiKeyRequest{Name: "   ", OwnerType: OwnerTypeUser, OwnerID: "user-1"},
			wantErr: "name is required",
		},
		{
			name:    "rejects_unknown_owner_type",
			request: CreateApiKeyRequest{Name: "CI key", OwnerType: "team", OwnerID: "user-1"},
			wantErr: "owner_type must be either 'user' or 'organization'",
		},
		{
			name:    "rejects_empty_owner_type",
			request: CreateApiKeyRequest{Name: "CI key", OwnerID: "user-1"},
			wantErr: "owner_type must be either 'user' or 'organization'",
		},
		{
			name:    "rejects_empty_owner_id",
			request: CreateApiKeyRequest{Name: "CI key", OwnerType: OwnerTypeUser},
			wantErr: "owner_id is required",
		},
		{
			name: "accepts_disabled_rate_limiting_without_limits",
			request: CreateApiKeyRequest{
				Name:             "CI key",
				OwnerType:        OwnerTypeUser,
				OwnerID:          "user-1",
				RateLimitEnabled: new(false),
			},
		},
		{
			name: "rejects_enabled_rate_limiting_without_window",
			request: CreateApiKeyRequest{
				Name:                 "CI key",
				OwnerType:            OwnerTypeUser,
				OwnerID:              "user-1",
				RateLimitEnabled:     new(true),
				RateLimitMaxRequests: new(100),
			},
			wantErr: "rate_limit_time_window is required when rate limiting is enabled",
		},
		{
			name: "rejects_enabled_rate_limiting_with_non_positive_window",
			request: CreateApiKeyRequest{
				Name:                 "CI key",
				OwnerType:            OwnerTypeUser,
				OwnerID:              "user-1",
				RateLimitEnabled:     new(true),
				RateLimitTimeWindow:  new(0),
				RateLimitMaxRequests: new(100),
			},
			wantErr: "rate_limit_time_window is required when rate limiting is enabled",
		},
		{
			name: "rejects_enabled_rate_limiting_without_max_requests",
			request: CreateApiKeyRequest{
				Name:                "CI key",
				OwnerType:           OwnerTypeUser,
				OwnerID:             "user-1",
				RateLimitEnabled:    new(true),
				RateLimitTimeWindow: new(60),
			},
			wantErr: "rate_limit_max_requests is required when rate limiting is enabled",
		},
		{
			name: "rejects_enabled_rate_limiting_with_non_positive_max_requests",
			request: CreateApiKeyRequest{
				Name:                 "CI key",
				OwnerType:            OwnerTypeUser,
				OwnerID:              "user-1",
				RateLimitEnabled:     new(true),
				RateLimitTimeWindow:  new(60),
				RateLimitMaxRequests: new(-1),
			},
			wantErr: "rate_limit_max_requests is required when rate limiting is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestUpdateApiKeyRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request UpdateApiKeyRequest
		wantErr string
	}{
		{
			name:    "accepts_empty_request",
			request: UpdateApiKeyRequest{},
		},
		{
			name:    "accepts_non_blank_name",
			request: UpdateApiKeyRequest{Name: new("Renamed key")},
		},
		{
			name:    "rejects_blank_name",
			request: UpdateApiKeyRequest{Name: new("   ")},
			wantErr: "name cannot be empty",
		},
		{
			name:    "accepts_disabled_rate_limiting_without_limits",
			request: UpdateApiKeyRequest{RateLimitEnabled: new(false)},
		},
		{
			name:    "accepts_enabled_rate_limiting_with_positive_limits",
			request: UpdateApiKeyRequest{RateLimitEnabled: new(true), RateLimitTimeWindow: new(60), RateLimitMaxRequests: new(10)},
		},
		{
			name:    "rejects_enabled_rate_limiting_without_window",
			request: UpdateApiKeyRequest{RateLimitEnabled: new(true), RateLimitMaxRequests: new(10)},
			wantErr: "rate_limit_time_window is required when rate limiting is enabled",
		},
		{
			name:    "rejects_enabled_rate_limiting_with_zero_window",
			request: UpdateApiKeyRequest{RateLimitEnabled: new(true), RateLimitTimeWindow: new(0), RateLimitMaxRequests: new(10)},
			wantErr: "rate_limit_time_window is required when rate limiting is enabled",
		},
		{
			name:    "rejects_enabled_rate_limiting_without_max_requests",
			request: UpdateApiKeyRequest{RateLimitEnabled: new(true), RateLimitTimeWindow: new(60)},
			wantErr: "rate_limit_max_requests is required when rate limiting is enabled",
		},
		{
			name:    "rejects_enabled_rate_limiting_with_zero_max_requests",
			request: UpdateApiKeyRequest{RateLimitEnabled: new(true), RateLimitTimeWindow: new(60), RateLimitMaxRequests: new(0)},
			wantErr: "rate_limit_max_requests is required when rate limiting is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestVerifyApiKeyRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request VerifyApiKeyRequest
		wantErr string
	}{
		{
			name:    "accepts_non_blank_key",
			request: VerifyApiKeyRequest{Key: "abc123"},
		},
		{
			name:    "rejects_empty_key",
			request: VerifyApiKeyRequest{},
			wantErr: "key is required",
		},
		{
			name:    "rejects_blank_key",
			request: VerifyApiKeyRequest{Key: "   "},
			wantErr: "key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestApiKey_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := ApiKey{
		ID:               "key-1",
		KeyHash:          "hash",
		Name:             "CI key",
		OwnerType:        OwnerTypeUser,
		OwnerID:          "user-1",
		Start:            "abc",
		Last:             "xyz",
		Prefix:           new("ci"),
		Enabled:          true,
		RateLimitEnabled: true,
		LastRequestedAt:  new(now),
		ExpiresAt:        new(now.Add(time.Hour)),
		Permissions:      []string{"read", "write"},
		Metadata:         map[string]any{"env": "ci"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got ApiKey
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestCreateApiKeyRequest_JSONOmitempty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request CreateApiKeyRequest
		want    string
	}{
		{
			name:    "omits_unset_optionals",
			request: CreateApiKeyRequest{Name: "CI key", OwnerType: OwnerTypeUser, OwnerID: "user-1"},
			want:    `{"name":"CI key","owner_type":"user","owner_id":"user-1"}`,
		},
		{
			name: "includes_populated_optionals",
			request: CreateApiKeyRequest{
				Name:                 "CI key",
				OwnerType:            OwnerTypeUser,
				OwnerID:              "user-1",
				Prefix:               new("ci"),
				Enabled:              new(true),
				RateLimitEnabled:     new(true),
				RateLimitTimeWindow:  new(60),
				RateLimitMaxRequests: new(10),
				Permissions:          []string{"read"},
				Metadata:             map[string]any{"env": "ci"},
			},
			want: `{"name":"CI key","owner_type":"user","owner_id":"user-1","prefix":"ci","enabled":true,"rate_limit_enabled":true,"rate_limit_time_window":60,"rate_limit_max_requests":10,"permissions":["read"],"metadata":{"env":"ci"}}`,
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
