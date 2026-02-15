package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/plugins/magic-link/types"
)

func TestSignInUseCase_SignIn(t *testing.T) {
	userCreated := false
	accountCreated := false
	capturedEmail := ""
	capturedName := "__unset__"

	tests := []struct {
		name        string
		email       string
		inputName   *string
		wantErr     bool
		errContains string
		setup       func(
			t *testing.T,
			uc *SignInUseCaseImpl,
			userSvc *mockUserService,
			accountSvc *mockAccountService,
			tokenSvc *mockTokenService,
			verificationSvc *mockVerificationService,
		)
		assert func(t *testing.T, result *types.SignInResult)
	}{
		{
			name:  "existing user",
			email: "test@example.com",
		},
		{
			name:      "new user sign up",
			email:     "newuser@example.com",
			inputName: strPtr("John Doe"),
			setup: func(
				t *testing.T,
				_ *SignInUseCaseImpl,
				userSvc *mockUserService,
				accountSvc *mockAccountService,
				_ *mockTokenService,
				_ *mockVerificationService,
			) {
				t.Helper()
				userCreated = false
				accountCreated = false
				userSvc.GetByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
					return nil, nil
				}
				userSvc.CreateFn = func(ctx context.Context, name, email string, emailVerified bool, image *string) (*models.User, error) {
					userCreated = true
					return &models.User{ID: "user-1", Name: name, Email: email}, nil
				}
				accountSvc.CreateFn = func(ctx context.Context, userID, accountID, providerID string, password *string) (*models.Account, error) {
					accountCreated = true
					return &models.Account{ID: "account-1", UserID: userID}, nil
				}
			},
			assert: func(t *testing.T, _ *types.SignInResult) {
				t.Helper()
				if !userCreated {
					t.Fatal("expected user to be created")
				}
				if !accountCreated {
					t.Fatal("expected account to be created")
				}
			},
		},
		{
			name:        "new user sign up disabled",
			email:       "newuser@example.com",
			wantErr:     true,
			errContains: "disabled",
			setup: func(
				t *testing.T,
				uc *SignInUseCaseImpl,
				userSvc *mockUserService,
				_ *mockAccountService,
				_ *mockTokenService,
				_ *mockVerificationService,
			) {
				t.Helper()
				uc.PluginConfig.DisableSignUp = true
				userSvc.GetByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
					return nil, nil
				}
			},
		},
		{
			name:  "email normalization",
			email: "TEST@EXAMPLE.COM",
			setup: func(
				t *testing.T,
				_ *SignInUseCaseImpl,
				userSvc *mockUserService,
				_ *mockAccountService,
				_ *mockTokenService,
				_ *mockVerificationService,
			) {
				t.Helper()
				capturedEmail = ""
				userSvc.GetByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
					capturedEmail = email
					return nil, nil
				}
				userSvc.CreateFn = func(ctx context.Context, name, email string, emailVerified bool, image *string) (*models.User, error) {
					return &models.User{ID: "user-1", Email: email}, nil
				}
			},
			assert: func(t *testing.T, _ *types.SignInResult) {
				t.Helper()
				if capturedEmail != "test@example.com" {
					t.Fatalf("expected normalized email, got %s", capturedEmail)
				}
			},
		},
		{
			name:        "user service error",
			email:       "test@example.com",
			wantErr:     true,
			errContains: "database error",
			setup: func(
				t *testing.T,
				_ *SignInUseCaseImpl,
				userSvc *mockUserService,
				_ *mockAccountService,
				_ *mockTokenService,
				_ *mockVerificationService,
			) {
				t.Helper()
				userSvc.GetByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
					return nil, errors.New("database error")
				}
			},
		},
		{
			name:        "token generation error",
			email:       "test@example.com",
			wantErr:     true,
			errContains: "token generation failed",
			setup: func(
				t *testing.T,
				_ *SignInUseCaseImpl,
				_ *mockUserService,
				_ *mockAccountService,
				tokenSvc *mockTokenService,
				_ *mockVerificationService,
			) {
				t.Helper()
				tokenSvc.GenerateFn = func() (string, error) {
					return "", errors.New("token generation failed")
				}
			},
		},
		{
			name:        "verification creation error",
			email:       "test@example.com",
			wantErr:     true,
			errContains: "verification creation failed",
			setup: func(
				t *testing.T,
				_ *SignInUseCaseImpl,
				_ *mockUserService,
				_ *mockAccountService,
				_ *mockTokenService,
				verificationSvc *mockVerificationService,
			) {
				t.Helper()
				verificationSvc.CreateFn = func(ctx context.Context, userID, hashedToken string, vType models.VerificationType, value string, expiry time.Duration) (*models.Verification, error) {
					return nil, errors.New("verification creation failed")
				}
			},
		},
		{
			name:  "new user without name",
			email: "test@example.com",
			setup: func(
				t *testing.T,
				_ *SignInUseCaseImpl,
				userSvc *mockUserService,
				_ *mockAccountService,
				_ *mockTokenService,
				_ *mockVerificationService,
			) {
				t.Helper()
				capturedName = "__unset__"
				userSvc.GetByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
					return nil, nil
				}
				userSvc.CreateFn = func(ctx context.Context, name, email string, emailVerified bool, image *string) (*models.User, error) {
					capturedName = name
					return &models.User{ID: "user-1", Name: name, Email: email}, nil
				}
			},
			assert: func(t *testing.T, _ *types.SignInResult) {
				t.Helper()
				if capturedName != "" {
					t.Fatalf("expected empty name for user without name, got: %s", capturedName)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			uc, userSvc, accountSvc, tokenSvc, verificationSvc, _ := newSignInTestUseCase(t)
			if tt.setup != nil {
				tt.setup(t, uc, userSvc, accountSvc, tokenSvc, verificationSvc)
			}

			result, err := uc.SignIn(context.Background(), tt.inputName, tt.email, nil)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if result != nil {
					t.Fatalf("expected nil result")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error to contain %q, got %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result == nil || result.Token == "" {
				t.Fatalf("expected token in result")
			}

			if tt.assert != nil {
				tt.assert(t, result)
			}
		})
	}
}

func strPtr(value string) *string {
	return &value
}
