package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/plugins/jwt/constants"
)

func TestJWTTokenType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tokenType JWTTokenType
		want      string
	}{
		{
			name:      "access_token",
			tokenType: JWTTokenTypeAccess,
			want:      "access_token",
		},
		{
			name:      "refresh_token",
			tokenType: JWTTokenTypeRefresh,
			want:      "refresh_token",
		},
		{
			name:      "custom_value",
			tokenType: JWTTokenType("custom"),
			want:      "custom",
		},
		{
			name:      "empty_value",
			tokenType: JWTTokenType(""),
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.tokenType.String())
		})
	}
}

func TestRefreshTokenRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request RefreshTokenRequest
		wantErr error
	}{
		{
			name:    "rejects_empty_token",
			request: RefreshTokenRequest{},
			wantErr: constants.ErrRefreshTokenIsRequired,
		},
		{
			name:    "accepts_non_empty_token",
			request: RefreshTokenRequest{RefreshToken: "refresh-token-value"},
		},
		{
			// Validation only checks for an empty string, so whitespace is accepted.
			name:    "accepts_whitespace_only_token",
			request: RefreshTokenRequest{RefreshToken: "   "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.request.Validate()
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
