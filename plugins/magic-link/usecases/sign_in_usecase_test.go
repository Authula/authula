package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	emailtmpl "github.com/Authula/authula/core/email/template"
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
		assertResult func(t *testing.T, result *types.MagicLinkSignInResult, err error)
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			},
		},
		{
			name:  "user creation error returns the error",
			email: "newuser@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				userSvc.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, nil).Once()
				userSvc.On("Create", mock.Anything, "", "newuser@example.com", false, (*string)(nil), mock.Anything).
					Return(nil, errors.New("user creation failed")).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
				assert.EqualError(t, err, "user creation failed")
				assert.Nil(t, result)
			},
		},
		{
			name:  "account creation error returns the error",
			email: "newuser@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				userSvc.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, nil).Once()
				userSvc.On("Create", mock.Anything, "", "newuser@example.com", false, (*string)(nil), mock.Anything).
					Return(&models.User{ID: "user-1", Email: "newuser@example.com"}, nil).Once()
				accountSvc.On("Create", mock.Anything, "user-1", "newuser@example.com", models.AuthProviderMagicLink.String(), (*string)(nil)).
					Return(nil, errors.New("account creation failed")).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					accountSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
				assert.EqualError(t, err, "account creation failed")
				assert.Nil(t, result)
			},
		},
		{
			name:  "callback error is logged and the token is still returned",
			email: "test@example.com",
			setup: func(t *testing.T, uc *SignInUseCaseImpl, userSvc *mockUserService, accountSvc *mockAccountService, tokenSvc *mockTokenService, verificationSvc *mockVerificationService, mailerSvc *mockMailerService) {
				logger := newErrorSignalingLogger()
				uc.Logger = logger
				uc.PluginConfig.SendMagicLinkVerificationEmail = func(params types.SendMagicLinkVerificationEmailParams, reqCtx *models.RequestContext) error {
					return errors.New("callback failed")
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "token-123", result.Token)
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
			assertResult: func(t *testing.T, result *types.MagicLinkSignInResult, err error) {
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

type sentMagicLinkEmail struct {
	to      string
	subject string
	text    string
	html    string
}

func TestSignInUseCase_BuiltInMailer(t *testing.T) {
	testCases := []struct {
		name            string
		templateManager func(t *testing.T) *emailtmpl.Manager
		wantMailer      bool
		wantLog         string
	}{
		{
			name:            "sends the magic link email in the background",
			templateManager: newSignInTemplateManager,
			wantMailer:      true,
		},
		{
			name: "logs a background render failure",
			templateManager: func(t *testing.T) *emailtmpl.Manager {
				t.Helper()
				return emailtmpl.NewManager()
			},
			wantLog: "failed to send magic link verification email via built-in email service",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			uc, userSvc, _, tokenSvc, verificationSvc, mailerSvc := newSignInTestUseCase()

			logger := newErrorSignalingLogger()
			uc.Logger = logger
			uc.MailerService = mailerSvc
			uc.EmailTemplateManager = tt.templateManager(t)

			userSvc.On("GetByEmail", mock.Anything, "test@example.com").
				Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
			tokenSvc.On("Generate").Return("token-123", nil).Once()
			tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
			verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
				Return(&models.Verification{ID: "verif-1"}, nil).Once()

			t.Cleanup(func() {
				userSvc.AssertExpectations(t)
				tokenSvc.AssertExpectations(t)
				verificationSvc.AssertExpectations(t)
				mailerSvc.AssertExpectations(t)
			})

			sent := &sentMagicLinkEmail{}
			mailerDone := make(chan struct{})
			if tt.wantMailer {
				mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent.to = args.String(1)
						sent.subject = args.String(2)
						sent.text = args.String(3)
						sent.html = args.String(4)
						close(mailerDone)
					}).
					Return(nil).Once()
			}

			result, err := uc.SignIn(context.Background(), nil, "test@example.com", nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "token-123", result.Token)

			select {
			case <-mailerDone:
			case <-logger.done:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for the background email work")
			}

			if tt.wantLog != "" {
				assert.Contains(t, logger.messagesSnapshot(), tt.wantLog)
				return
			}

			assert.Empty(t, logger.messagesSnapshot())
			assert.Equal(t, "test@example.com", sent.to)
			assert.Contains(t, sent.subject, "TestApp")
			assert.Contains(t, sent.text, "token-123")
			assert.Contains(t, sent.html, "token-123")
		})
	}
}

func TestSignInUseCase_SendMagicLinkVerificationEmail(t *testing.T) {
	testCases := []struct {
		name            string
		templateManager func(t *testing.T) *emailtmpl.Manager
		mailerErr       error
		wantMailer      bool
		wantErr         string
	}{
		{
			name:            "renders and sends the email",
			templateManager: newSignInTemplateManager,
			wantMailer:      true,
		},
		{
			name: "render failure is returned",
			templateManager: func(t *testing.T) *emailtmpl.Manager {
				t.Helper()
				return emailtmpl.NewManager()
			},
			wantErr: "not found",
		},
		{
			name:            "mailer failure is returned",
			templateManager: newSignInTemplateManager,
			mailerErr:       errors.New("smtp unavailable"),
			wantMailer:      true,
			wantErr:         "smtp unavailable",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			uc, _, _, _, _, mailerSvc := newSignInTestUseCase()

			uc.EmailTemplateManager = tt.templateManager(t)
			uc.MailerService = mailerSvc

			sent := &sentMagicLinkEmail{}
			if tt.wantMailer {
				mailerSvc.On("SendEmail", mock.Anything, "test@example.com", mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						sent.to = args.String(1)
						sent.subject = args.String(2)
						sent.text = args.String(3)
						sent.html = args.String(4)
					}).
					Return(tt.mailerErr).Once()
			}

			err := uc.sendMagicLinkVerificationEmail(context.Background(), "test@example.com", "http://localhost/auth/magic-link/verify?token=abc")

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				mailerSvc.AssertExpectations(t)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "test@example.com", sent.to)
			assert.Contains(t, sent.subject, "TestApp")
			assert.Contains(t, sent.text, "http://localhost/auth/magic-link/verify?token=abc")
			assert.Contains(t, sent.html, "http://localhost/auth/magic-link/verify?token=abc")
			mailerSvc.AssertExpectations(t)
		})
	}
}
