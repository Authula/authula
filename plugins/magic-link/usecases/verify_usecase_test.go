package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
)

func TestVerifyUseCase_Verify_ValidToken(t *testing.T) {
	uc, _, _, _ := newVerifyTestUseCase(t)

	code, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if code != "new-exchange-code" {
		t.Fatalf("expected code to be new-exchange-code, got %s", code)
	}
}

func TestVerifyUseCase_Verify_ExpiredToken(t *testing.T) {
	uc, _, verificationSvc, _ := newVerifyTestUseCase(t)
	verificationSvc.IsExpiredFn = func(verif *models.Verification) bool {
		return true
	}

	_, err := uc.Verify(context.Background(), "expired-token", nil, nil)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if err.Error() != "invalid or expired token" {
		t.Fatalf("expected 'invalid or expired token' error, got: %v", err)
	}
	if err.Error() != "invalid or expired token" {
		t.Fatalf("expected 'invalid or expired token' error, got: %v", err)
	}
}

func TestVerifyUseCase_Verify_MissingToken(t *testing.T) {
	uc, _, verificationSvc, _ := newVerifyTestUseCase(t)
	verificationSvc.GetByTokenFn = func(ctx context.Context, hashedToken string) (*models.Verification, error) {
		return nil, nil
	}

	_, err := uc.Verify(context.Background(), "invalid-token", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if err.Error() != "invalid or expired token" {
		t.Fatalf("expected 'invalid or expired token' error, got: %v", err)
	}
}

func TestVerifyUseCase_Verify_InvalidTokenType(t *testing.T) {
	uc, _, verificationSvc, _ := newVerifyTestUseCase(t)
	userID := "user-1"
	verificationSvc.GetByTokenFn = func(ctx context.Context, hashedToken string) (*models.Verification, error) {
		return &models.Verification{ID: "verif-1", UserID: &userID, Type: "password-reset"}, nil
	}

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid token type")
	}
	if err.Error() != "invalid token type" {
		t.Fatalf("expected 'invalid token type' error, got: %v", err)
	}
}

func TestVerifyUseCase_Verify_UserNotFound(t *testing.T) {
	uc, userSvc, _, _ := newVerifyTestUseCase(t)
	userGetByIDCalled := false
	userSvc.GetByIDFn = func(ctx context.Context, id string) (*models.User, error) {
		userGetByIDCalled = true
		return nil, nil
	}

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err == nil {
		t.Fatal("expected error for user not found")
	}
	if err.Error() != "user not found" {
		t.Fatalf("expected 'user not found' error, got: %v", err)
	}
	if !userGetByIDCalled {
		t.Fatal("expected GetByID to be called")
	}
}

func TestVerifyUseCase_Verify_UserEmailVerificationUpdated(t *testing.T) {
	uc, userSvc, _, _ := newVerifyTestUseCase(t)
	emailVerifiedUpdated := false
	userSvc.UpdateFieldsFn = func(ctx context.Context, id string, fields map[string]any) error {
		if verified, ok := fields["email_verified"]; ok && verified == true {
			emailVerifiedUpdated = true
		}
		return nil
	}

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !emailVerifiedUpdated {
		t.Fatal("expected email_verified to be updated to true")
	}
}
func TestVerifyUseCase_Verify_DeletesOriginalVerification(t *testing.T) {
	uc, _, verificationSvc, _ := newVerifyTestUseCase(t)
	deleteCalled := false
	verificationSvc.DeleteFn = func(ctx context.Context, id string) error {
		deleteCalled = true
		if id != "verif-1" {
			t.Fatalf("expected to delete verif-1, got %s", id)
		}
		return nil
	}

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Fatal("expected original verification to be deleted")
	}
}

func TestVerifyUseCase_Verify_CreatesNewExchangeCodeVerification(t *testing.T) {
	uc, _, verificationSvc, _ := newVerifyTestUseCase(t)
	createCalled := false
	verificationSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
		createCalled = true
		if userID != "user-1" {
			t.Fatalf("expected userID user-1, got %s", userID)
		}
		if vType != models.TypeMagicLinkExchangeCode {
			t.Fatalf("expected type TypeMagicLinkExchangeCode, got %s", vType)
		}
		if hashedToken != "hashed-new-exchange-code" {
			t.Fatalf("expected hashed new-exchange-code, got %s", hashedToken)
		}
		if value != "test@example.com" {
			t.Fatalf("expected identifier test@example.com, got %s", value)
		}
		return &models.Verification{ID: "verif-2", UserID: &userID, Type: vType}, nil
	}

	_, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !createCalled {
		t.Fatal("expected new exchange code verification to be created")
	}
}

func TestVerifyUseCase_Verify_GeneratesNewToken(t *testing.T) {
	uc, _, _, tokenSvc := newVerifyTestUseCase(t)
	generateCalled := false
	tokenSvc.GenerateFn = func() (string, error) {
		generateCalled = true
		return "new-token-456", nil
	}

	code, err := uc.Verify(context.Background(), "test-token", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !generateCalled {
		t.Fatal("expected TokenService.Generate to be called")
	}
	if code != "new-token-456" {
		t.Fatalf("expected code to be new-token-456, got %s", code)
	}
}
