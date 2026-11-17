package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

func TestVerifyTOTPRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request VerifyTOTPRequest
		wantErr string
	}{
		{
			name:    "accepts_code",
			request: VerifyTOTPRequest{Code: "123456"},
		},
		{
			name:    "accepts_trust_device_flag",
			request: VerifyTOTPRequest{Code: "123456", TrustDevice: true},
		},
		{
			name:    "rejects_empty_code",
			request: VerifyTOTPRequest{},
			wantErr: "backup code is required",
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

func TestVerifyBackupCodeRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request VerifyBackupCodeRequest
		wantErr string
	}{
		{
			name:    "accepts_code",
			request: VerifyBackupCodeRequest{Code: "backup-code"},
		},
		{
			name:    "accepts_trust_device_flag",
			request: VerifyBackupCodeRequest{Code: "backup-code", TrustDevice: true},
		},
		{
			name:    "rejects_empty_code",
			request: VerifyBackupCodeRequest{},
			wantErr: "backup code is required",
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

func TestEnableResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := EnableResponse{
		TotpURI:     "otpauth://totp/Authula:john.doe@example.com?secret=ABC123",
		BackupCodes: []string{"code-1", "code-2"},
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)
	require.JSONEq(t, `{"totp_uri":"otpauth://totp/Authula:john.doe@example.com?secret=ABC123","backup_codes":["code-1","code-2"]}`, string(data))

	var got EnableResponse
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}

func TestVerifyTOTPRequest_JSONOmitsUntrustedDeviceFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request VerifyTOTPRequest
		want    string
	}{
		{
			name:    "omits_false_trust_device",
			request: VerifyTOTPRequest{Code: "123456"},
			want:    `{"code":"123456"}`,
		},
		{
			name:    "includes_true_trust_device",
			request: VerifyTOTPRequest{Code: "123456", TrustDevice: true},
			want:    `{"code":"123456","trust_device":true}`,
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

func TestVerifyTOTPResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := VerifyTOTPResponse{
		User: &models.User{
			ID:            "user-1",
			Name:          "John Doe",
			Email:         "john.doe@example.com",
			EmailVerified: true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Session: &models.Session{
			ID:        "session-1",
			UserID:    "user-1",
			Token:     "token-123",
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	data, err := json.Marshal(want)
	require.NoError(t, err)

	var got VerifyTOTPResponse
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}
