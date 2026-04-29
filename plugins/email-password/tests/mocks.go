package tests

import (
	"context"
	"encoding/json"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/stretchr/testify/mock"
)

type MockSignUpUseCase struct{ mock.Mock }

func (m *MockSignUpUseCase) SignUp(ctx context.Context, name string, email string, password string, image *string, metadata json.RawMessage, callbackURL *string, ipAddress *string, userAgent *string) (*types.SignUpResult, error) {
	args := m.Called(ctx, name, email, password, image, metadata, callbackURL, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SignUpResult), args.Error(1)
}

type MockSignInUseCase struct{ mock.Mock }

func (m *MockSignInUseCase) SignIn(ctx context.Context, email string, password string, callbackURL *string, ipAddress *string, userAgent *string) (*types.SignInResult, error) {
	args := m.Called(ctx, email, password, callbackURL, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SignInResult), args.Error(1)
}

func (m *MockSignInUseCase) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSignInUseCase) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

type MockVerifyEmailUseCase struct{ mock.Mock }

func (m *MockVerifyEmailUseCase) VerifyEmail(ctx context.Context, tokenStr string) (models.VerificationType, error) {
	args := m.Called(ctx, tokenStr)
	return args.Get(0).(models.VerificationType), args.Error(1)
}

type MockSendEmailVerificationUseCase struct{ mock.Mock }

func (m *MockSendEmailVerificationUseCase) Send(ctx context.Context, userID string, callbackURL *string) error {
	args := m.Called(ctx, userID, callbackURL)
	return args.Error(0)
}

type MockRequestPasswordResetUseCase struct{ mock.Mock }

func (m *MockRequestPasswordResetUseCase) RequestReset(ctx context.Context, email string, callbackURL *string) error {
	args := m.Called(ctx, email, callbackURL)
	return args.Error(0)
}

type MockChangePasswordUseCase struct{ mock.Mock }

func (m *MockChangePasswordUseCase) ChangePassword(ctx context.Context, tokenValue string, newPassword string) error {
	args := m.Called(ctx, tokenValue, newPassword)
	return args.Error(0)
}

type MockRequestEmailChangeUseCase struct{ mock.Mock }

func (m *MockRequestEmailChangeUseCase) RequestChange(ctx context.Context, userID string, newEmail string, callbackURL *string) error {
	args := m.Called(ctx, userID, newEmail, callbackURL)
	return args.Error(0)
}
