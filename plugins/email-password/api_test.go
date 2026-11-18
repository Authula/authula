package email_password

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	plugintests "github.com/Authula/authula/plugins/email-password/tests"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/Authula/authula/plugins/email-password/usecases"
)

func TestAPI_SignUp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	image := new("https://example.com/avatar.png")
	metadata := map[string]any{"locale": "en"}
	callbackURL := new("https://app.example.com/callback")
	ipAddress := new("203.0.113.10")
	userAgent := new("test-agent/1.0")
	result := &types.SignUpResult{SessionToken: "session-token"}
	errSignUp := errors.New("sign up failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockSignUpUseCase)
		want    *types.SignUpResult
		wantErr error
	}{
		{
			name: "returns the result from the use case",
			setup: func(m *plugintests.MockSignUpUseCase) {
				m.On("SignUp", ctx, "Ada", "ada@example.com", "secret", image, metadata, callbackURL, ipAddress, userAgent).Return(result, nil).Once()
			},
			want: result,
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockSignUpUseCase) {
				m.On("SignUp", ctx, "Ada", "ada@example.com", "secret", image, metadata, callbackURL, ipAddress, userAgent).Return(nil, errSignUp).Once()
			},
			wantErr: errSignUp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockSignUpUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{SignUpUseCase: useCase}}

			got, err := api.SignUp(ctx, "Ada", "ada@example.com", "secret", image, metadata, callbackURL, ipAddress, userAgent)

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.want, got)
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_SignIn(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	callbackURL := new("https://app.example.com/callback")
	ipAddress := new("203.0.113.10")
	userAgent := new("test-agent/1.0")
	result := &types.SignInResult{SessionToken: "session-token"}
	errSignIn := errors.New("sign in failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockSignInUseCase)
		want    *types.SignInResult
		wantErr error
	}{
		{
			name: "returns the result from the use case",
			setup: func(m *plugintests.MockSignInUseCase) {
				m.On("SignIn", ctx, "ada@example.com", "secret", callbackURL, ipAddress, userAgent).Return(result, nil).Once()
			},
			want: result,
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockSignInUseCase) {
				m.On("SignIn", ctx, "ada@example.com", "secret", callbackURL, ipAddress, userAgent).Return(nil, errSignIn).Once()
			},
			wantErr: errSignIn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockSignInUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{SignInUseCase: useCase}}

			got, err := api.SignIn(ctx, "ada@example.com", "secret", callbackURL, ipAddress, userAgent)

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.want, got)
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_VerifyEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errVerify := errors.New("verification failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockVerifyEmailUseCase)
		want    models.VerificationType
		wantErr error
	}{
		{
			name: "returns the verification type from the use case",
			setup: func(m *plugintests.MockVerifyEmailUseCase) {
				m.On("VerifyEmail", ctx, "token-123").Return(models.TypeEmailVerification, nil).Once()
			},
			want: models.TypeEmailVerification,
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockVerifyEmailUseCase) {
				m.On("VerifyEmail", ctx, "token-123").Return(models.VerificationType(""), errVerify).Once()
			},
			wantErr: errVerify,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockVerifyEmailUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{VerifyEmailUseCase: useCase}}

			got, err := api.VerifyEmail(ctx, "token-123")

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.want, got)
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_SendEmailVerification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	callbackURL := new("https://app.example.com/callback")
	errSend := errors.New("send failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockSendEmailVerificationUseCase)
		wantErr error
	}{
		{
			name: "succeeds when the use case succeeds",
			setup: func(m *plugintests.MockSendEmailVerificationUseCase) {
				m.On("Send", ctx, "user-1", callbackURL).Return(nil).Once()
			},
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockSendEmailVerificationUseCase) {
				m.On("Send", ctx, "user-1", callbackURL).Return(errSend).Once()
			},
			wantErr: errSend,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockSendEmailVerificationUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{SendEmailVerificationUseCase: useCase}}

			err := api.SendEmailVerification(ctx, "user-1", callbackURL)

			require.ErrorIs(t, err, tt.wantErr)
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_RequestPasswordReset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	callbackURL := new("https://app.example.com/callback")
	errReset := errors.New("reset failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockRequestPasswordResetUseCase)
		wantErr error
	}{
		{
			name: "succeeds when the use case succeeds",
			setup: func(m *plugintests.MockRequestPasswordResetUseCase) {
				m.On("RequestReset", ctx, "ada@example.com", callbackURL).Return(nil).Once()
			},
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockRequestPasswordResetUseCase) {
				m.On("RequestReset", ctx, "ada@example.com", callbackURL).Return(errReset).Once()
			},
			wantErr: errReset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockRequestPasswordResetUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{RequestPasswordResetUseCase: useCase}}

			err := api.RequestPasswordReset(ctx, "ada@example.com", callbackURL)

			require.ErrorIs(t, err, tt.wantErr)
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_ChangePassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errChange := errors.New("change password failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockChangePasswordUseCase)
		wantErr error
	}{
		{
			name: "succeeds when the use case succeeds",
			setup: func(m *plugintests.MockChangePasswordUseCase) {
				m.On("ChangePassword", ctx, "token-123", "new-secret").Return(nil).Once()
			},
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockChangePasswordUseCase) {
				m.On("ChangePassword", ctx, "token-123", "new-secret").Return(errChange).Once()
			},
			wantErr: errChange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockChangePasswordUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{ChangePasswordUseCase: useCase}}

			err := api.ChangePassword(ctx, "token-123", "new-secret")

			require.ErrorIs(t, err, tt.wantErr)
			useCase.AssertExpectations(t)
		})
	}
}

func TestAPI_RequestEmailChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	callbackURL := new("https://app.example.com/callback")
	errChange := errors.New("request email change failed")

	tests := []struct {
		name    string
		setup   func(*plugintests.MockRequestEmailChangeUseCase)
		wantErr error
	}{
		{
			name: "succeeds when the use case succeeds",
			setup: func(m *plugintests.MockRequestEmailChangeUseCase) {
				m.On("RequestChange", ctx, "user-1", "new@example.com", callbackURL).Return(nil).Once()
			},
		},
		{
			name: "returns the error from the use case",
			setup: func(m *plugintests.MockRequestEmailChangeUseCase) {
				m.On("RequestChange", ctx, "user-1", "new@example.com", callbackURL).Return(errChange).Once()
			},
			wantErr: errChange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			useCase := new(plugintests.MockRequestEmailChangeUseCase)
			tt.setup(useCase)
			api := &API{useCases: &usecases.UseCases{RequestEmailChangeUseCase: useCase}}

			err := api.RequestEmailChange(ctx, "user-1", "new@example.com", callbackURL)

			require.ErrorIs(t, err, tt.wantErr)
			useCase.AssertExpectations(t)
		})
	}
}
