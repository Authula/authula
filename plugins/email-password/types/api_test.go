package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
)

func TestSignUpRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request SignUpRequest
		want    SignUpRequest
	}{
		{
			name: "trims_and_lowercases_all_string_fields",
			request: SignUpRequest{
				Name:        "  Ada Lovelace  ",
				Email:       "  ADA@Example.COM ",
				Password:    "  s3cret  ",
				Image:       new("  /avatar.png  "),
				CallbackURL: new("  /welcome  "),
			},
			want: SignUpRequest{
				Name:        "Ada Lovelace",
				Email:       "ada@example.com",
				Password:    "s3cret",
				Image:       new("/avatar.png"),
				CallbackURL: new("/welcome"),
			},
		},
		{
			name: "leaves_nil_optionals_untouched",
			request: SignUpRequest{
				Name:     "Ada Lovelace",
				Email:    "ada@example.com",
				Password: "s3cret",
			},
			want: SignUpRequest{
				Name:     "Ada Lovelace",
				Email:    "ada@example.com",
				Password: "s3cret",
			},
		},
		{
			name: "keeps_metadata_as_provided",
			request: SignUpRequest{
				Name:     "Ada",
				Email:    "ada@example.com",
				Password: "s3cret",
				Metadata: map[string]any{"plan": "pro"},
			},
			want: SignUpRequest{
				Name:     "Ada",
				Email:    "ada@example.com",
				Password: "s3cret",
				Metadata: map[string]any{"plan": "pro"},
			},
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

func TestSignInRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request SignInRequest
		want    SignInRequest
	}{
		{
			name: "trims_and_lowercases_fields",
			request: SignInRequest{
				Email:       "  ADA@Example.COM ",
				Password:    "  s3cret  ",
				CallbackURL: new("  /dashboard  "),
			},
			want: SignInRequest{
				Email:       "ada@example.com",
				Password:    "s3cret",
				CallbackURL: new("/dashboard"),
			},
		},
		{
			name: "leaves_nil_callback_url_untouched",
			request: SignInRequest{
				Email:    "ada@example.com",
				Password: "s3cret",
			},
			want: SignInRequest{
				Email:    "ada@example.com",
				Password: "s3cret",
			},
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

func TestVerifyEmailRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request VerifyEmailRequest
		want    VerifyEmailRequest
	}{
		{
			name: "trims_token_and_callback_url",
			request: VerifyEmailRequest{
				Token:       "  token-123  ",
				CallbackURL: new("  /verified  "),
			},
			want: VerifyEmailRequest{
				Token:       "token-123",
				CallbackURL: new("/verified"),
			},
		},
		{
			name:    "accepts_zero_value",
			request: VerifyEmailRequest{},
			want:    VerifyEmailRequest{},
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

func TestSendEmailVerificationRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request SendEmailVerificationRequest
		want    SendEmailVerificationRequest
	}{
		{
			name:    "trims_callback_url",
			request: SendEmailVerificationRequest{CallbackURL: new("  /sent  ")},
			want:    SendEmailVerificationRequest{CallbackURL: new("/sent")},
		},
		{
			name:    "leaves_nil_callback_url_untouched",
			request: SendEmailVerificationRequest{},
			want:    SendEmailVerificationRequest{},
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

func TestRequestPasswordResetRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request RequestPasswordResetRequest
		want    RequestPasswordResetRequest
	}{
		{
			name: "trims_and_lowercases_email",
			request: RequestPasswordResetRequest{
				Email:       "  ADA@Example.COM ",
				CallbackURL: new("  /reset  "),
			},
			want: RequestPasswordResetRequest{
				Email:       "ada@example.com",
				CallbackURL: new("/reset"),
			},
		},
		{
			name:    "leaves_nil_callback_url_untouched",
			request: RequestPasswordResetRequest{Email: "ada@example.com"},
			want:    RequestPasswordResetRequest{Email: "ada@example.com"},
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

func TestRequestEmailChangeRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request RequestEmailChangeRequest
		want    RequestEmailChangeRequest
	}{
		{
			name: "trims_and_lowercases_new_email",
			request: RequestEmailChangeRequest{
				NewEmail:    "  NEW@Example.COM ",
				CallbackURL: new("  /changed  "),
			},
			want: RequestEmailChangeRequest{
				NewEmail:    "new@example.com",
				CallbackURL: new("/changed"),
			},
		},
		{
			name:    "leaves_nil_callback_url_untouched",
			request: RequestEmailChangeRequest{NewEmail: "new@example.com"},
			want:    RequestEmailChangeRequest{NewEmail: "new@example.com"},
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

func TestChangePasswordRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request ChangePasswordRequest
		want    ChangePasswordRequest
	}{
		{
			name:    "trims_token_and_password",
			request: ChangePasswordRequest{Token: "  token-123  ", Password: "  s3cret  "},
			want:    ChangePasswordRequest{Token: "token-123", Password: "s3cret"},
		},
		{
			name:    "accepts_zero_value",
			request: ChangePasswordRequest{},
			want:    ChangePasswordRequest{},
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

func TestSignUpRequest_JSONOmitempty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request SignUpRequest
		want    string
	}{
		{
			name: "omits_unset_optionals",
			request: SignUpRequest{
				Name:     "Ada",
				Email:    "ada@example.com",
				Password: "s3cret",
			},
			want: `{"name":"Ada","email":"ada@example.com","password":"s3cret"}`,
		},
		{
			name: "includes_populated_optionals",
			request: SignUpRequest{
				Name:        "Ada",
				Email:       "ada@example.com",
				Password:    "s3cret",
				Image:       new("/avatar.png"),
				Metadata:    map[string]any{"plan": "pro"},
				CallbackURL: new("/welcome"),
			},
			want: `{"name":"Ada","email":"ada@example.com","password":"s3cret","image":"/avatar.png","metadata":{"plan":"pro"},"callback_url":"/welcome"}`,
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

func TestSignUpRequest_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	payload := `{"name":"Ada","email":"ada@example.com","password":"s3cret","image":"/avatar.png","metadata":{"plan":"pro"},"callback_url":"/welcome"}`

	var request SignUpRequest
	require.NoError(t, json.Unmarshal([]byte(payload), &request))
	require.Equal(t, SignUpRequest{
		Name:        "Ada",
		Email:       "ada@example.com",
		Password:    "s3cret",
		Image:       new("/avatar.png"),
		Metadata:    map[string]any{"plan": "pro"},
		CallbackURL: new("/welcome"),
	}, request)
}

func TestSignInResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 17, 12, 30, 0, 0, time.UTC)
	want := SignInResponse{
		User: &models.User{
			ID:            "user-1",
			Name:          "Ada Lovelace",
			Email:         "ada@example.com",
			EmailVerified: true,
			Image:         new("/avatar.png"),
			Metadata:      map[string]any{"plan": "pro"},
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
	require.JSONEq(t, `{"user":{"id":"user-1","name":"Ada Lovelace","email":"ada@example.com","email_verified":true,"image":"/avatar.png","metadata":{"plan":"pro"},"created_at":"2026-11-17T12:30:00Z","updated_at":"2026-11-17T12:30:00Z"},"session":{"id":"session-1","user_id":"user-1","token":"token-123","expires_at":"2026-11-17T13:30:00Z","ip_address":null,"user_agent":null,"created_at":"2026-11-17T12:30:00Z","updated_at":"2026-11-17T12:30:00Z"}}`, string(data))

	var got SignInResponse
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}
