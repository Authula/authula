package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func TestVerifyUseCase_Verify_ValidToken(t *testing.T) {
	uc, userSvc, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}

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

	code, err := uc.Verify(context.Background(), "test-token", nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, "new-exchange-code", code)
	userSvc.AssertExpectations(t)
	verificationSvc.AssertExpectations(t)
	tokenSvc.AssertExpectations(t)
}

func TestVerifyUseCase_Verify_ExpiredToken(t *testing.T) {
	uc, _, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}

	tokenSvc.On("Hash", "expired-token").Return("hashed-expired-token").Once()
	verificationSvc.On("GetByToken", mock.Anything, "hashed-expired-token").Return(verification, nil).Once()
	verificationSvc.On("IsExpired", verification).Return(true).Once()

	_, err := uc.Verify(context.Background(), "expired-token", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
}

func TestVerifyUseCase_Verify_MissingToken(t *testing.T) {
	uc, _, verificationSvc, tokenSvc := newVerifyTestUseCase()

	tokenSvc.On("Hash", "invalid-token").Return("hashed-invalid-token").Once()
	verificationSvc.On("GetByToken", mock.Anything, "hashed-invalid-token").Return(nil, nil).Once()

	_, err := uc.Verify(context.Background(), "invalid-token", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
}

func TestVerifyUseCase_Verify_InvalidTokenType(t *testing.T) {
	uc, _, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeEmailVerification}

	tokenSvc.On("Hash", "test-token").Return("hashed-test-token").Once()
	verificationSvc.On("GetByToken", mock.Anything, "hashed-test-token").Return(verification, nil).Once()
	verificationSvc.On("IsExpired", verification).Return(false).Once()

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, "invalid token type", err.Error())
}

func TestVerifyUseCase_Verify_UserNotFound(t *testing.T) {
	uc, userSvc, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}

	tokenSvc.On("Hash", "test-token").Return("hashed-test-token").Once()
	verificationSvc.On("GetByToken", mock.Anything, "hashed-test-token").Return(verification, nil).Once()
	verificationSvc.On("IsExpired", verification).Return(false).Once()
	userSvc.On("GetByID", mock.Anything, "user-1").Return(nil, nil).Once()

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)

	assert.Error(t, err)
	assert.Equal(t, "user not found", err.Error())
}

func TestVerifyUseCase_Verify_UserEmailVerificationUpdated(t *testing.T) {
	uc, userSvc, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}

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

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)

	assert.NoError(t, err)
	userSvc.AssertCalled(t, "UpdateFields", mock.Anything, "user-1", map[string]any{"email_verified": true})
}

func TestVerifyUseCase_Verify_DeletesOriginalVerification(t *testing.T) {
	uc, userSvc, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}

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

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)

	assert.NoError(t, err)
	verificationSvc.AssertCalled(t, "Delete", mock.Anything, "verif-1")
}

func TestVerifyUseCase_Verify_GeneratesNewToken(t *testing.T) {
	uc, userSvc, verificationSvc, tokenSvc := newVerifyTestUseCase()
	userID := "user-1"
	verification := &models.Verification{ID: "verif-1", UserID: &userID, Type: models.TypeMagicLinkSignInRequest}

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

	code, err := uc.Verify(context.Background(), "test-token", nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, "new-token-456", code)
}
