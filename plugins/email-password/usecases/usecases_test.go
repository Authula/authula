package usecases

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type emailPasswordTestFixture struct {
	globalConfig    *models.Config
	pluginConfig    types.EmailPasswordPluginConfig
	userSvc         *inttests.MockUserService
	accountSvc      *inttests.MockAccountService
	sessionSvc      *inttests.MockSessionService
	verificationSvc *inttests.MockVerificationService
	tokenSvc        *inttests.MockTokenService
	passwordSvc     *inttests.MockPasswordService
	mailerSvc       *inttests.MockMailerService
	logger          *inttests.MockLogger
	eventBus        *inttests.MockEventBus
}

func newEmailPasswordTestFixture() *emailPasswordTestFixture {
	return &emailPasswordTestFixture{
		globalConfig: &models.Config{
			BaseURL:  "http://localhost",
			BasePath: "/auth",
			AppName:  "TestApp",
			Session:  models.SessionConfig{ExpiresIn: time.Hour},
		},
		pluginConfig: types.EmailPasswordPluginConfig{
			MinPasswordLength:           8,
			MaxPasswordLength:           128,
			RequireEmailVerification:    true,
			AutoSignIn:                  true,
			EmailVerificationExpiresIn:  24 * time.Hour,
			PasswordResetExpiresIn:      time.Hour,
			RequestEmailChangeExpiresIn: time.Hour,
		},
		userSvc:         &inttests.MockUserService{},
		accountSvc:      &inttests.MockAccountService{},
		sessionSvc:      &inttests.MockSessionService{},
		verificationSvc: &inttests.MockVerificationService{},
		tokenSvc:        &inttests.MockTokenService{},
		passwordSvc:     &inttests.MockPasswordService{},
		mailerSvc:       &inttests.MockMailerService{},
		logger:          &inttests.MockLogger{},
		eventBus:        &inttests.MockEventBus{},
	}
}

func (f *emailPasswordTestFixture) signUpUseCase() SignUpUseCase {
	return NewSignUpUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.sessionSvc, f.tokenSvc, f.passwordSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) signInUseCase() SignInUseCase {
	return NewSignInUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.sessionSvc, f.tokenSvc, f.passwordSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) verifyEmailUseCase() VerifyEmailUseCase {
	return NewVerifyEmailUseCase(f.pluginConfig, f.logger, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) sendEmailVerificationUseCase() SendEmailVerificationUseCase {
	return NewSendEmailVerificationUseCase(f.globalConfig, f.pluginConfig, f.logger, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc)
}

func (f *emailPasswordTestFixture) requestPasswordResetUseCase() RequestPasswordResetUseCase {
	return NewRequestPasswordResetUseCase(f.logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc)
}

func (f *emailPasswordTestFixture) changePasswordUseCase() ChangePasswordUseCase {
	return NewChangePasswordUseCase(f.logger, f.pluginConfig, f.userSvc, f.accountSvc, f.verificationSvc, f.tokenSvc, f.passwordSvc, f.mailerSvc, f.eventBus)
}

func (f *emailPasswordTestFixture) requestEmailChangeUseCase() RequestEmailChangeUseCase {
	return NewRequestEmailChangeUseCase(f.logger, f.globalConfig, f.pluginConfig, f.userSvc, f.verificationSvc, f.tokenSvc, f.mailerSvc)
}

func testRequestContext() context.Context {
	return models.SetRequestContext(context.Background(), &models.RequestContext{Values: map[string]any{}})
}

func TestSignUpUseCase_SignUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, *types.SignUpResult, error)
	}{
		{
			name:  "disabled sign up returns error",
			setup: func(f *emailPasswordTestFixture) { f.pluginConfig.DisableSignUp = true },
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.ErrorIs(t, err, constants.ErrSignUpDisabled)
				require.Nil(t, result)
			},
		},
		{
			name:  "invalid password length returns error",
			setup: func(f *emailPasswordTestFixture) { f.pluginConfig.MinPasswordLength = 12 },
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidPasswordLength)
				require.Nil(t, result)
			},
		},
		{
			name: "existing email returns conflict",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.ErrorIs(t, err, constants.ErrEmailAlreadyExists)
				require.Nil(t, result)
			},
		},
		{
			name: "creates user account and session",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
				f.passwordSvc.On("Hash", "Password1!").Return("hashed-password", nil).Once()
				f.userSvc.On("Create", mock.Anything, "Test User", "test@example.com", false, (*string)(nil), json.RawMessage(`{"role":"member"}`)).Return(&models.User{ID: "user-1", Name: "Test User", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("Create", mock.Anything, "user-1", "test@example.com", models.AuthProviderEmail.String(), mock.AnythingOfType("*string")).Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				f.tokenSvc.On("Generate").Return("session-token", nil).Once()
				f.tokenSvc.On("Hash", "session-token").Return("hashed-session-token").Once()
				f.sessionSvc.On("Create", mock.Anything, "user-1", "hashed-session-token", (*string)(nil), (*string)(nil), time.Hour).Return(&models.Session{ID: "session-1", UserID: "user-1"}, nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, result *types.SignUpResult, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "session-token", result.SessionToken)
				require.Equal(t, "user-1", result.User.ID)
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
			result, err := f.signUpUseCase().SignUp(context.Background(), "Test User", "test@example.com", "Password1!", nil, json.RawMessage(`{"role":"member"}`), nil, nil, nil)
			tc.assert(t, result, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
			f.sessionSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestSignInUseCase_SignIn(t *testing.T) {
	t.Parallel()

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

func TestVerifyEmailUseCase_VerifyEmail(t *testing.T) {
	t.Parallel()

	userID := "user-1"

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, models.VerificationType, error)
	}{
		{
			name: "invalid token returns error",
			setup: func(f *emailPasswordTestFixture) {
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.ErrorIs(t, err, constants.ErrInvalidOrExpiredToken)
				require.Empty(t, vt)
			},
		},
		{
			name: "email verification marks user verified",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypeEmailVerification, UserID: &userID, ExpiresAt: time.Now().Add(time.Hour)}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "test@example.com"}, nil).Once()
				f.userSvc.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.EmailVerified })).Return(&models.User{ID: userID, EmailVerified: true}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailVerification, vt)
			},
		},
		{
			name: "email reset request returns type",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypeEmailResetRequest, UserID: &userID, ExpiresAt: time.Now().Add(time.Hour), Identifier: "new@example.com"}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, userID).Return(&models.User{ID: userID, Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, userID, models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: userID}, nil).Once()
				f.userSvc.On("Update", mock.Anything, mock.Anything).Return(&models.User{ID: userID, Email: "new@example.com"}, nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.Anything).Return(&models.Account{ID: "account-1"}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "old@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				f.mailerSvc.On("SendEmail", mock.Anything, "new@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, vt models.VerificationType, err error) {
				require.NoError(t, err)
				require.Equal(t, models.TypeEmailResetRequest, vt)
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
			vt, err := f.verifyEmailUseCase().VerifyEmail(context.Background(), "token-123")
			tc.assert(t, vt, err)
			f.userSvc.AssertExpectations(t)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestSendEmailVerificationUseCase_Send(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name:   "disabled verification returns nil",
			setup:  func(f *emailPasswordTestFixture) { f.pluginConfig.RequireEmailVerification = false },
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "missing user is swallowed",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "verified user returns nil",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com", EmailVerified: true}, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "creates verification and calls hook",
			setup: func(f *emailPasswordTestFixture) {
				f.pluginConfig.SendEmailVerification = func(params types.SendEmailVerificationParams, reqCtx *models.RequestContext) error {
					require.Equal(t, "test@example.com", params.User.Email)
					require.NotNil(t, reqCtx)
					return nil
				}
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("verify-token", nil).Once()
				f.tokenSvc.On("Hash", "verify-token").Return("hashed-token").Once()
				f.verificationSvc.On("DeleteByUserIDAndType", mock.Anything, "user-1", models.TypeEmailVerification).Return(nil).Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailVerification, "test@example.com", 24*time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			err := f.sendEmailVerificationUseCase().Send(testRequestContext(), "test@example.com", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestRequestPasswordResetUseCase_RequestReset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name: "missing user returns nil",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
		{
			name: "creates reset verification",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.tokenSvc.On("Generate").Return("reset-token", nil).Once()
				f.tokenSvc.On("Hash", "reset-token").Return("hashed-token").Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypePasswordResetRequest, "test@example.com", time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			err := f.requestPasswordResetUseCase().RequestReset(testRequestContext(), "test@example.com", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}

func TestChangePasswordUseCase_ChangePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name:   "invalid length returns error",
			setup:  func(f *emailPasswordTestFixture) {},
			assert: func(t *testing.T, err error) { require.ErrorIs(t, err, constants.ErrInvalidPasswordLength) },
		},
		{
			name: "updates password on valid token",
			setup: func(f *emailPasswordTestFixture) {
				verification := &models.Verification{ID: "verif-1", Type: models.TypePasswordResetRequest, UserID: func() *string { s := "user-1"; return &s }(), ExpiresAt: time.Now().Add(time.Hour)}
				f.tokenSvc.On("Hash", "token-123").Return("hashed-token").Once()
				f.verificationSvc.On("GetByToken", mock.Anything, "hashed-token").Return(verification, nil).Once()
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				f.accountSvc.On("GetByUserIDAndProvider", mock.Anything, "user-1", models.AuthProviderEmail.String()).Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				f.passwordSvc.On("Hash", "NewPassword1!").Return("hashed-password", nil).Once()
				f.accountSvc.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Account) bool { return a.Password != nil && *a.Password == "hashed-password" })).Return(&models.Account{ID: "account-1"}, nil).Once()
				f.verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				f.eventBus.On("Publish", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			password := "short"
			if tc.name == "updates password on valid token" {
				password = "NewPassword1!"
			}
			err := f.changePasswordUseCase().ChangePassword(context.Background(), "token-123", password)
			tc.assert(t, err)
			f.accountSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
			f.passwordSvc.AssertExpectations(t)
		})
	}
}

func TestRequestEmailChangeUseCase_RequestChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(*emailPasswordTestFixture)
		assert func(*testing.T, error)
	}{
		{
			name: "missing user returns error",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()
			},
			assert: func(t *testing.T, err error) { require.ErrorIs(t, err, constants.ErrUserNotFound) },
		},
		{
			name: "creates email change verification",
			setup: func(f *emailPasswordTestFixture) {
				f.userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "old@example.com"}, nil).Once()
				f.userSvc.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, nil).Once()
				f.tokenSvc.On("Generate").Return("change-token", nil).Once()
				f.tokenSvc.On("Hash", "change-token").Return("hashed-token").Once()
				f.verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token", models.TypeEmailResetRequest, "new@example.com", time.Hour).Return(&models.Verification{ID: "verif-1"}, nil).Once()
				f.mailerSvc.On("SendEmail", mock.Anything, "new@example.com", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			assert: func(t *testing.T, err error) { require.NoError(t, err) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newEmailPasswordTestFixture()
			if tc.setup != nil {
				tc.setup(f)
			}
			err := f.requestEmailChangeUseCase().RequestChange(testRequestContext(), "user-1", "new@example.com", nil)
			tc.assert(t, err)
			f.userSvc.AssertExpectations(t)
			f.verificationSvc.AssertExpectations(t)
			f.tokenSvc.AssertExpectations(t)
		})
	}
}
