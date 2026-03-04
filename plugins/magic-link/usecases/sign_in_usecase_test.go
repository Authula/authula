package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func TestSignInUseCase_SignIn_ExistingUser(t *testing.T) {
	uc, userSvc, _, tokenSvc, verificationSvc, _ := newSignInTestUseCase()
	uc.PluginConfig.SendMagicLinkVerificationEmail = func(email, verificationURL, token string) error { return nil }

	userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
	tokenSvc.On("Generate").Return("token-123", nil).Once()
	tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
	verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
		Return(&models.Verification{ID: "verif-1"}, nil).Once()

	result, err := uc.SignIn(context.Background(), nil, "test@example.com", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "token-123", result.Token)
	userSvc.AssertExpectations(t)
	tokenSvc.AssertExpectations(t)
	verificationSvc.AssertExpectations(t)
}

func TestSignInUseCase_SignIn_NewUserSignUp(t *testing.T) {
	uc, userSvc, accountSvc, tokenSvc, verificationSvc, _ := newSignInTestUseCase()
	uc.PluginConfig.SendMagicLinkVerificationEmail = func(email, verificationURL, token string) error { return nil }

	name := "John Doe"
	userSvc.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, nil).Once()
	userSvc.On("Create", mock.Anything, "John Doe", "newuser@example.com", false, (*string)(nil), mock.Anything).
		Return(&models.User{ID: "user-1", Name: "John Doe", Email: "newuser@example.com"}, nil).Once()
	accountSvc.On("Create", mock.Anything, "user-1", "newuser@example.com", models.AuthProviderMagicLink.String(), (*string)(nil)).
		Return(&models.Account{ID: "account-1", UserID: "user-1"}, nil).Once()
	tokenSvc.On("Generate").Return("token-123", nil).Once()
	tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
	verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "newuser@example.com", uc.PluginConfig.ExpiresIn).
		Return(&models.Verification{ID: "verif-1"}, nil).Once()

	result, err := uc.SignIn(context.Background(), &name, "newuser@example.com", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "token-123", result.Token)
	userSvc.AssertExpectations(t)
	accountSvc.AssertExpectations(t)
	tokenSvc.AssertExpectations(t)
	verificationSvc.AssertExpectations(t)
}

func TestSignInUseCase_SignIn_NewUserSignUpDisabled(t *testing.T) {
	uc, userSvc, _, _, _, _ := newSignInTestUseCase()
	uc.PluginConfig.DisableSignUp = true

	userSvc.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, nil).Once()

	result, err := uc.SignIn(context.Background(), nil, "newuser@example.com", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "disabled")
	userSvc.AssertExpectations(t)
}

func TestSignInUseCase_SignIn_EmailNormalization(t *testing.T) {
	uc, userSvc, accountSvc, tokenSvc, verificationSvc, _ := newSignInTestUseCase()
	uc.PluginConfig.SendMagicLinkVerificationEmail = func(email, verificationURL, token string) error { return nil }

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

	result, err := uc.SignIn(context.Background(), nil, "TEST@EXAMPLE.COM", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test@example.com", capturedEmail)
}

func TestSignInUseCase_SignIn_GetByEmailError(t *testing.T) {
	uc, userSvc, _, _, _, _ := newSignInTestUseCase()

	userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database error")).Once()

	result, err := uc.SignIn(context.Background(), nil, "test@example.com", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

func TestSignInUseCase_SignIn_TokenGenerationError(t *testing.T) {
	uc, userSvc, _, tokenSvc, _, _ := newSignInTestUseCase()
	uc.PluginConfig.SendMagicLinkVerificationEmail = func(email, verificationURL, token string) error { return nil }

	userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
	tokenSvc.On("Generate").Return("", errors.New("token generation failed")).Once()

	result, err := uc.SignIn(context.Background(), nil, "test@example.com", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "token generation failed")
}

func TestSignInUseCase_SignIn_VerificationCreationError(t *testing.T) {
	uc, userSvc, _, tokenSvc, verificationSvc, _ := newSignInTestUseCase()
	uc.PluginConfig.SendMagicLinkVerificationEmail = func(email, verificationURL, token string) error { return nil }

	userSvc.On("GetByEmail", mock.Anything, "test@example.com").Return(&models.User{ID: "user-1", Email: "test@example.com"}, nil).Once()
	tokenSvc.On("Generate").Return("token-123", nil).Once()
	tokenSvc.On("Hash", "token-123").Return("hashed-token-123").Once()
	verificationSvc.On("Create", mock.Anything, "user-1", "hashed-token-123", models.TypeMagicLinkSignInRequest, "test@example.com", uc.PluginConfig.ExpiresIn).
		Return(nil, errors.New("verification creation failed")).Once()

	result, err := uc.SignIn(context.Background(), nil, "test@example.com", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "verification creation failed")
}

func TestSignInUseCase_SignIn_NewUserWithoutNameUsesEmptyString(t *testing.T) {
	uc, userSvc, accountSvc, tokenSvc, verificationSvc, _ := newSignInTestUseCase()
	uc.PluginConfig.SendMagicLinkVerificationEmail = func(email, verificationURL, token string) error { return nil }

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

	result, err := uc.SignIn(context.Background(), nil, strings.ToUpper("test@example.com"), nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "", capturedName)
}
