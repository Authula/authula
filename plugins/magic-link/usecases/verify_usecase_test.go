package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Authula/authula/models"
)

func TestVerifyUseCase_Verify(t *testing.T) {
	testCases := []struct {
		name         string
		token        string
		verification *models.Verification
		setup        func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification)
		assertResult func(t *testing.T, code string, err error)
	}{
		{
			name:  "valid token issues a new exchange code",
			token: "test-token",
			verification: func() *models.Verification {
				userID := "user-1"
				return &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}
			}(),
			setup: func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification) {
				tokenSvc.On("Hash", "test-token").Return("hashed-test-token").Once()
				verificationSvc.On("GetByToken", mock.Anything, "hashed-test-token").Return(verification, nil).Once()
				verificationSvc.On("IsExpired", verification).Return(false).Once()
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				userSvc.On("UpdateFields", mock.Anything, "user-1", map[string]any{"email_verified": true}).Return(nil).Once()
				verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				tokenSvc.On("Generate").Return("new-exchange-code", nil).Once()
				tokenSvc.On("Hash", "new-exchange-code").Return("hashed-new-exchange-code").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-new-exchange-code", models.TypeMagicLinkExchangeCode, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-2"}, nil).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, code string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "new-exchange-code", code)
			},
		},
		{
			name:  "expired token is rejected",
			token: "expired-token",
			verification: func() *models.Verification {
				userID := "user-1"
				return &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}
			}(),
			setup: func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification) {
				tokenSvc.On("Hash", "expired-token").Return("hashed-expired-token").Once()
				verificationSvc.On("GetByToken", mock.Anything, "hashed-expired-token").Return(verification, nil).Once()
				verificationSvc.On("IsExpired", verification).Return(true).Once()

				t.Cleanup(func() {
					verificationSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, code string, err error) {
				assert.EqualError(t, err, "invalid or expired token")
				assert.Empty(t, code)
			},
		},
		{
			name:  "missing token is rejected",
			token: "invalid-token",
			setup: func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification) {
				tokenSvc.On("Hash", "invalid-token").Return("hashed-invalid-token").Once()
				verificationSvc.On("GetByToken", mock.Anything, "hashed-invalid-token").Return(nil, nil).Once()

				t.Cleanup(func() {
					verificationSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, code string, err error) {
				assert.EqualError(t, err, "invalid or expired token")
				assert.Empty(t, code)
			},
		},
		{
			name:  "invalid token type is rejected",
			token: "test-token",
			verification: func() *models.Verification {
				userID := "user-1"
				return &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeEmailVerification}
			}(),
			setup: func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification) {
				tokenSvc.On("Hash", "test-token").Return("hashed-test-token").Once()
				verificationSvc.On("GetByToken", mock.Anything, "hashed-test-token").Return(verification, nil).Once()
				verificationSvc.On("IsExpired", verification).Return(false).Once()

				t.Cleanup(func() {
					verificationSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, code string, err error) {
				assert.EqualError(t, err, "invalid token type")
				assert.Empty(t, code)
			},
		},
		{
			name:  "user not found is rejected",
			token: "test-token",
			verification: func() *models.Verification {
				userID := "user-1"
				return &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}
			}(),
			setup: func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification) {
				tokenSvc.On("Hash", "test-token").Return("hashed-test-token").Once()
				verificationSvc.On("GetByToken", mock.Anything, "hashed-test-token").Return(verification, nil).Once()
				verificationSvc.On("IsExpired", verification).Return(false).Once()
				userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, code string, err error) {
				assert.EqualError(t, err, "user not found")
				assert.Empty(t, code)
			},
		},
		{
			name:  "email verification is marked verified and original token is replaced",
			token: "test-token",
			verification: func() *models.Verification {
				userID := "user-1"
				return &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}
			}(),
			setup: func(t *testing.T, uc *VerifyUseCaseImpl, userSvc *mockUserService, verificationSvc *mockVerificationService, tokenSvc *mockTokenService, verification *models.Verification) {
				tokenSvc.On("Hash", "test-token").Return("hashed-test-token").Once()
				verificationSvc.On("GetByToken", mock.Anything, "hashed-test-token").Return(verification, nil).Once()
				verificationSvc.On("IsExpired", verification).Return(false).Once()
				userSvc.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
				userSvc.On("UpdateFields", mock.Anything, "user-1", map[string]any{"email_verified": true}).Return(nil).Once()
				verificationSvc.On("Delete", mock.Anything, "verif-1").Return(nil).Once()
				tokenSvc.On("Generate").Return("new-token-456", nil).Once()
				tokenSvc.On("Hash", "new-token-456").Return("hashed-new-token-456").Once()
				verificationSvc.On("Create", mock.Anything, "user-1", "hashed-new-token-456", models.TypeMagicLinkExchangeCode, "test@example.com", uc.PluginConfig.ExpiresIn).
					Return(&models.Verification{ID: "verif-2"}, nil).Once()

				t.Cleanup(func() {
					userSvc.AssertExpectations(t)
					verificationSvc.AssertExpectations(t)
					tokenSvc.AssertExpectations(t)
				})
			},
			assertResult: func(t *testing.T, code string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "new-token-456", code)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			uc, userSvc, verificationSvc, tokenSvc := newVerifyTestUseCase()
			tt.setup(t, uc, userSvc, verificationSvc, tokenSvc, tt.verification)

			code, err := uc.Verify(context.Background(), tt.token, nil, nil)
			tt.assertResult(t, code, err)
		})
	}
}
