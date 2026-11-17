package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coreerrors "github.com/Authula/authula/core/errors"
	"github.com/Authula/authula/models"
)

func TestMagicLinkSignInRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request MagicLinkSignInRequest
		want    MagicLinkSignInRequest
		wantIs  error
	}{
		{
			name: "trims_and_lowercases_email",
			request: MagicLinkSignInRequest{
				Email:       "  ADA@Example.COM ",
				Name:        new(" Ada Lovelace "),
				CallbackURL: new(" /welcome "),
			},
			want: MagicLinkSignInRequest{
				Email:       "ada@example.com",
				Name:        new(" Ada Lovelace "),
				CallbackURL: new(" /welcome "),
			},
		},
		{
			name:    "accepts_minimal_request",
			request: MagicLinkSignInRequest{Email: "ada@example.com"},
			want:    MagicLinkSignInRequest{Email: "ada@example.com"},
		},
		{
			name:   "rejects_empty_email",
			want:   MagicLinkSignInRequest{},
			wantIs: coreerrors.ErrEmailRequired,
		},
		{
			name:    "rejects_whitespace_email",
			request: MagicLinkSignInRequest{Email: "   "},
			want:    MagicLinkSignInRequest{Email: ""},
			wantIs:  coreerrors.ErrEmailRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := tt.request
			err := request.Validate()
			if tt.wantIs != nil {
				require.ErrorIs(t, err, tt.wantIs)
				require.Equal(t, tt.want, request)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, request)
		})
	}
}

func TestMagicLinkVerifyRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request MagicLinkVerifyRequest
		want    MagicLinkVerifyRequest
	}{
		{
			name:    "trims_token",
			request: MagicLinkVerifyRequest{Token: "  token-123  ", CallbackURL: "/welcome"},
			want:    MagicLinkVerifyRequest{Token: "token-123", CallbackURL: "/welcome"},
		},
		{
			name:    "accepts_zero_value",
			request: MagicLinkVerifyRequest{},
			want:    MagicLinkVerifyRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := tt.request
			require.NoError(t, request.Validate())
			require.Equal(t, tt.want, request)
		})
	}
}

func TestMagicLinkExchangeRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request MagicLinkExchangeRequest
		wantErr string
	}{
		{
			name:    "accepts_token",
			request: MagicLinkExchangeRequest{Token: "token-123"},
		},
		{
			name:    "rejects_empty_token",
			request: MagicLinkExchangeRequest{},
			wantErr: "token is required",
		},
		{
			name:    "rejects_whitespace_token",
			request: MagicLinkExchangeRequest{Token: "   "},
			wantErr: "token is required",
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

func TestMagicLinkSignInRequest_JSONOmitempty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request MagicLinkSignInRequest
		want    string
	}{
		{
			name:    "omits_unset_optionals",
			request: MagicLinkSignInRequest{Email: "ada@example.com"},
			want:    `{"email":"ada@example.com"}`,
		},
		{
			name: "includes_populated_optionals",
			request: MagicLinkSignInRequest{
				Email:       "ada@example.com",
				Name:        new("Ada Lovelace"),
				CallbackURL: new("/welcome"),
			},
			want: `{"email":"ada@example.com","name":"Ada Lovelace","callback_url":"/welcome"}`,
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

func TestMagicLinkVerifyResponse_JSONOmitsEmptyToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response MagicLinkVerifyResponse
		want     string
	}{
		{
			name:     "omits_empty_token",
			response: MagicLinkVerifyResponse{Message: "verified"},
			want:     `{"message":"verified"}`,
		},
		{
			name:     "includes_token",
			response: MagicLinkVerifyResponse{Message: "verified", Token: "token-123"},
			want:     `{"message":"verified","token":"token-123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestMagicLinkExchangeResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 17, 10, 0, 0, 0, time.UTC)
	want := MagicLinkExchangeResponse{
		User: &models.User{
			ID:            "user-1",
			Name:          "Ada Lovelace",
			Email:         "ada@example.com",
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

	var got MagicLinkExchangeResponse
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}
