package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestSignInUseCase(t *testing.T) {
	t.Parallel()

	errUserLookup := errors.New("user lookup failed")
	errAccountLookup := errors.New("account lookup failed")
	errTokenGenerate := errors.New("token generation failed")
	errSessionCreate := errors.New("session creation failed")

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *types.SignInResult, error)
	}{
		{
			name: "invalid credentials when user missing",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidCredentials)
				require.Nil(t, result)
			},
		},
		{
			name: "invalid credentials when password mismatches",
			setup: func(f *emailPasswordTestFixture) {
				password := "hashed"
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", Password: &password}, nil).Once()
				f.passwordSvc.On("Verify", "wrong-password", "hashed").Return(false).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidCredentials)
				require.Nil(t, result)
			},
		},
		{
			name: "signs in existing user",
			setup: func(f *emailPasswordTestFixture) {
				password := "hashed-password"
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", Password: &password}, nil).Once()
				f.passwordSvc.On("Verify", "Password1!", "hashed-password").Return(true).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session-token").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session-token", (*string)(nil), (*string)(nil), time.Hour).Return(&models.Session{ID: "session-1"}, nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "session-token", result.SessionToken)
			},
		},
		{
			name: "returns error when user lookup fails",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errUserLookup).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, errUserLookup)
				require.Nil(t, result)
			},
		},
		{
			name: "returns error when account lookup fails",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(nil, errAccountLookup).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, errAccountLookup)
				require.Nil(t, result)
			},
		},
		{
			name: "returns error when token generation fails",
			setup: func(f *emailPasswordTestFixture) {
				password := "hashed-password"
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", Password: &password}, nil).Once()
				f.passwordSvc.On("Verify", "Password1!", "hashed-password").Return(true).Once()
				f.tokenSvc.On("Generate").Return("", errTokenGenerate).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, errTokenGenerate)
				require.Nil(t, result)
			},
		},
		{
			name: "returns error when session creation fails",
			setup: func(f *emailPasswordTestFixture) {
				password := "hashed-password"
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1", Password: &password}, nil).Once()
				f.passwordSvc.On("Verify", "Password1!", "hashed-password").Return(true).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session-token").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session-token", (*string)(nil), (*string)(nil), time.Hour).Return(nil, errSessionCreate).Once()
			},
			assert: func(t *testing.T, result *types.SignInResult, err error) {
				require.ErrorIs(t, err, errSessionCreate)
				require.Nil(t, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			password := "Password1!"
			if tc.name == "invalid credentials when password mismatches" {
				password = "wrong-password"
			}
			result, err := f.signInUseCase().SignIn(context.Background(), "test@example.com", password, nil, nil, nil)
			tc.assert(t, result, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestSignInUseCaseGetSessionByID(t *testing.T) {
	t.Parallel()

	errSessionLookup := errors.New("session lookup failed")

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *models.Session, error)
	}{
		{
			name: "returns session when it exists",
			setup: func(f *emailPasswordTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "session-1").Return(&models.Session{ID: "session-1", UserID: "user-1"}, nil).Once()
			},
			assert: func(t *testing.T, session *models.Session, err error) {
				require.NoError(t, err)
				require.NotNil(t, session)
				require.Equal(t, "session-1", session.ID)
			},
		},
		{
			name: "returns nil without error when session is not found",
			setup: func(f *emailPasswordTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "session-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, session *models.Session, err error) {
				require.NoError(t, err)
				require.Nil(t, session)
			},
		},
		{
			name: "returns error when session lookup fails",
			setup: func(f *emailPasswordTestFixture) {
				f.sessionSvc.On("GetByID", mock.Anything, "session-1").Return(nil, errSessionLookup).Once()
			},
			assert: func(t *testing.T, session *models.Session, err error) {
				require.ErrorIs(t, err, errSessionLookup)
				require.Nil(t, session)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tc.setup(f)
			session, err := f.signInUseCase().GetSessionByID(context.Background(), "session-1")
			tc.assert(t, session, err)
			f.sessionSvc.AssertExpectations(t)
		})
	}
}

func TestSignInUseCaseGetUserByID(t *testing.T) {
	t.Parallel()

	errUserLookup := errors.New("user lookup failed")

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *models.User, error)
	}{
		{
			name: "returns user when it exists",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
			},
			assert: func(t *testing.T, user *models.User, err error) {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.Equal(t, "user-1", user.ID)
			},
		},
		{
			name: "returns nil without error when user is not found",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, user *models.User, err error) {
				require.NoError(t, err)
				require.Nil(t, user)
			},
		},
		{
			name: "returns error when user lookup fails",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, errUserLookup).Once()
			},
			assert: func(t *testing.T, user *models.User, err error) {
				require.ErrorIs(t, err, errUserLookup)
				require.Nil(t, user)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			tc.setup(f)
			user, err := f.signInUseCase().GetUserByID(context.Background(), "user-1")
			tc.assert(t, user, err)
			f.userSvc.AssertExpectations(t)
		})
	}
}
