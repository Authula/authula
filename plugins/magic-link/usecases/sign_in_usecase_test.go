package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/magic-link/types"
)

func TestSignInUseCase(t *testing.T) {
	name := "John Doe"
	upperEmail := strings.ToUpper("test@example.com")

	testCases := []struct {
		name         string
		inputName    *string
		email        string
		setup        func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService)
		assertResult func(t *testing.T, result *types.SignInResult, err error)
	}{
		{
			name:  "existing user returns token",
			email: "test@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return nil
				}

				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				tokenSvc.On("Generate").Return("token-123", nil).Once()
				tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "token-123", result.Token)
			},
		},
		{
			name:      "new user signs up and gets a token",
			inputName: &name,
			email:     "newuser@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return nil
				}

				userSvc.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, nil).Once()
				userSvc.On("Create", mock.Anything, "John Doe", "newuser@example.com", false, (*string)(nil), mock.Anything).
					Return(&models.User{ID: "user-1", Name: "John Doe", Email: "newuser@example.com"}, nil).Once()
				accountSvc.On("Create", mock.Anything, "user-1", "newuser@example.com", models.AuthProviderMagicLink.String(), (*string)(nil)).
					Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				tokenSvc.On("Generate").Return("token-123", nil).Once()
				tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "newuser@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					accountSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "token-123", result.Token)
			},
		},
		{
			name:  "new user signup disabled returns error",
			email: "newuser@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.DisableSignUp = true

				userSvc.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, nil).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "disabled")
			},
		},
		{
			name:  "email normalization lowercases the address",
			email: upperEmail,
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return nil
				}

				capturedEmail := ""
				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Run(func(args mock.Arguments) {
					capturedEmail = args.String(1)
				}).Return(nil, nil).Once()
				userSvc.On("Create", mock.Anything, "", "test@example.com", false, (*string)(nil), mock.Anything).
					Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				accountSvc.On("Create", mock.Anything, "user-1", "test@example.com", models.AuthProviderMagicLink.String(), (*string)(nil)).
					Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				tokenSvc.On("Generate").Return("token-123", nil).Once()
				tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()

				t.Cleanup(func() {
					assert.Equal(t, "test@example.com", capturedEmail)
					userSvc.AssertExpectations(t)
					accountSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			},
		},
		{
			name:  "get by email error returns the error",
			email: "test@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database error")).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "database error")
			},
		},
		{
			name:  "token generation error returns the error",
			email: "test@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return nil
				}

				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				tokenSvc.On("Generate").Return("", errors.New("token generation failed")).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "token generation failed")
			},
		},
		{
			name:  "verification creation error returns the error",
			email: "test@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return nil
				}

				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				tokenSvc.On("Generate").Return("token-123", nil).Once()
				tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(nil, errors.New("verification creation failed")).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), "verification creation failed")
			},
		},
		{
			name:  "new user without name uses empty string",
			email: upperEmail,
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return nil
				}

				capturedName := "__unset__"
				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil).Once()
				userSvc.On("Create", mock.Anything, mock.AnythingOfType("string"), "test@example.com", false, (*string)(nil), mock.Anything).
					Run(func(args mock.Arguments) {
						capturedName = args.String(1)
					}).
					Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				accountSvc.On("Create", mock.Anything, "user-1", "test@example.com", models.AuthProviderMagicLink.String(), (*string)(nil)).
					Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
				tokenSvc.On("Generate").Return("token-123", nil).Once()
				tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()

				t.Cleanup(func() {
					assert.Equal(t, "", capturedName)
					userSvc.AssertExpectations(t)
					accountSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			},
		},
		{
			name:  "callback skips built-in mailer",
			email: "test@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				uc.MailerService = mailerSvc
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					require.Equal(t, "test@example.com", params.Email)
					require.NotNil(t, reqCtx)
					return nil
				}

				userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				tokenSvc.On("Generate").Return("token-123", nil).Once()
				tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-1"}, nil).Once()
			},
			assertResult: func(t *testing.T, result *types.SignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "token-123", result.Token)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			uc, userSvc, accountSvc, tokenSvc, verificationSvc, mailerSvc := newSignInTestUseCase()

			tt.setup(t, uc, userSvc, accountSvc, tokenSvc, verificationSvc, mailerSvc)

			ctx := models.SetRequestContext(context.Background(), &models.RequestContext{Values: map[string]any{}})
			result, err := uc.SignIn(ctx, tt.inputName, tt.email, nil)
			tt.assertResult(t, result, err)
		})
	}
}
