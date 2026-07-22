package services

import (
	"context"
	"errors"
	"testing"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

func TestServiceHookExecutor_NilConfigIsNoop(t *testing.T) {
	t.Parallel()

	executor := NewServiceHookExecutor(nil, nil, nil, nil)
	ctx := context.Background()
	user := &models.User{ID: "user-1", Email: "test@example.com"}
	signUpResult := &types.SignUpResult{User: user}
	signInResult := &types.SignInResult{User: user}

	if err := executor.BeforeSignUp(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterSignUp(ctx, signUpResult); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeSignIn(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterSignIn(ctx, signInResult); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterVerifyEmail(ctx, user, models.TypeEmailVerification); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeRequestPasswordReset(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeChangePassword(ctx, user, "newpass"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterChangePassword(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeRequestEmailChange(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterEmailChanged(ctx, user, "old@test.com", "new@test.com"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestServiceHookExecutor_NilExecutorIsNoop(t *testing.T) {
	t.Parallel()

	var executor *ServiceHookExecutor
	ctx := context.Background()
	user := &models.User{ID: "user-1"}
	signUpResult := &types.SignUpResult{User: user}
	signInResult := &types.SignInResult{User: user}

	if err := executor.BeforeSignUp(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterSignUp(ctx, signUpResult); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.BeforeSignIn(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterSignIn(ctx, signInResult); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterVerifyEmail(ctx, user, models.TypeEmailVerification); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestServiceHookExecutor_SignUpHooks(t *testing.T) {
	t.Parallel()

	var beforeCalled bool
	var afterCalled bool

	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		SignUp: &types.SignUpServiceHooksConfig{
			BeforeSignUp: func(ctx context.Context, user *models.User) error {
				beforeCalled = true
				if user == nil || user.ID != "user-1" {
					t.Fatalf("unexpected user in before sign up hook: %+v", user)
				}
				return nil
			},
			AfterSignUp: func(ctx context.Context, result *types.SignUpResult) error {
				afterCalled = true
				if result == nil || result.User.ID != "user-1" {
					t.Fatalf("unexpected result in after sign up hook: %+v", result)
				}
				return nil
			},
		},
	}, nil, nil, nil)

	ctx := context.Background()
	user := &models.User{ID: "user-1", Email: "test@example.com"}
	result := &types.SignUpResult{User: user}

	if err := executor.BeforeSignUp(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterSignUp(ctx, result); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !beforeCalled {
		t.Fatal("expected BeforeSignUp hook to be called")
	}
	if !afterCalled {
		t.Fatal("expected AfterSignUp hook to be called")
	}
}

func TestServiceHookExecutor_SignInHooks(t *testing.T) {
	t.Parallel()

	var beforeCalled bool
	var afterCalled bool

	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		SignIn: &types.SignInServiceHooksConfig{
			BeforeSignIn: func(ctx context.Context, user *models.User) error {
				beforeCalled = true
				if user == nil || user.ID != "user-1" {
					t.Fatalf("unexpected user in before sign in hook: %+v", user)
				}
				return nil
			},
			AfterSignIn: func(ctx context.Context, result *types.SignInResult) error {
				afterCalled = true
				if result == nil || result.User.ID != "user-1" {
					t.Fatalf("unexpected result in after sign in hook: %+v", result)
				}
				return nil
			},
		},
	}, nil, nil, nil)

	ctx := context.Background()
	user := &models.User{ID: "user-1", Email: "test@example.com"}
	result := &types.SignInResult{User: user}

	if err := executor.BeforeSignIn(ctx, user); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := executor.AfterSignIn(ctx, result); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !beforeCalled {
		t.Fatal("expected BeforeSignIn hook to be called")
	}
	if !afterCalled {
		t.Fatal("expected AfterSignIn hook to be called")
	}
}

func TestServiceHookExecutor_AfterVerifyEmailHook(t *testing.T) {
	t.Parallel()

	var called bool
	var gotType models.VerificationType

	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		EmailVerification: &types.EmailVerificationServiceHooksConfig{
			AfterVerifyEmail: func(ctx context.Context, user *models.User, verificationType models.VerificationType) error {
				called = true
				gotType = verificationType
				return nil
			},
		},
	}, nil, nil, nil)

	ctx := context.Background()
	user := &models.User{ID: "user-1"}

	if err := executor.AfterVerifyEmail(ctx, user, models.TypeEmailVerification); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !called {
		t.Fatal("expected AfterVerifyEmail hook to be called")
	}
	if gotType != models.TypeEmailVerification {
		t.Fatalf("expected verification type %s, got %s", models.TypeEmailVerification, gotType)
	}
}

func TestServiceHookExecutor_BeforeChangePasswordHook(t *testing.T) {
	t.Parallel()

	var called bool
	var gotPassword string

	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		PasswordChange: &types.PasswordChangeServiceHooksConfig{
			BeforeChangePassword: func(ctx context.Context, user *models.User, newPassword string) error {
				called = true
				gotPassword = newPassword
				return nil
			},
		},
	}, nil, nil, nil)

	ctx := context.Background()
	user := &models.User{ID: "user-1"}

	if err := executor.BeforeChangePassword(ctx, user, "newpassword123"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !called {
		t.Fatal("expected BeforeChangePassword hook to be called")
	}
	if gotPassword != "newpassword123" {
		t.Fatalf("expected password 'newpassword123', got %s", gotPassword)
	}
}

func TestServiceHookExecutor_AfterEmailChangedHook(t *testing.T) {
	t.Parallel()

	var called bool
	var gotOldEmail string
	var gotNewEmail string

	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		EmailChange: &types.EmailChangeServiceHooksConfig{
			AfterEmailChanged: func(ctx context.Context, user *models.User, oldEmail, newEmail string) error {
				called = true
				gotOldEmail = oldEmail
				gotNewEmail = newEmail
				return nil
			},
		},
	}, nil, nil, nil)

	ctx := context.Background()
	user := &models.User{ID: "user-1"}

	if err := executor.AfterEmailChanged(ctx, user, "old@test.com", "new@test.com"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !called {
		t.Fatal("expected AfterEmailChanged hook to be called")
	}
	if gotOldEmail != "old@test.com" {
		t.Fatalf("expected old email 'old@test.com', got %s", gotOldEmail)
	}
	if gotNewEmail != "new@test.com" {
		t.Fatalf("expected new email 'new@test.com', got %s", gotNewEmail)
	}
}

func TestServiceHookExecutor_BeforeHookError(t *testing.T) {
	t.Parallel()

	someErr := errors.New("some error")
	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		SignUp: &types.SignUpServiceHooksConfig{
			BeforeSignUp: func(ctx context.Context, user *models.User) error {
				return someErr
			},
		},
	}, nil, nil, nil)

	err := executor.BeforeSignUp(context.Background(), &models.User{ID: "user-1"})
	if !errors.Is(err, someErr) {
		t.Fatalf("expected someErr error, got %v", err)
	}
}

func TestServiceHookExecutor_AfterHookErrorIsLoggedNotReturned(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		SignUp: &types.SignUpServiceHooksConfig{
			AfterSignUp: func(ctx context.Context, result *types.SignUpResult) error {
				return errors.New("hook error")
			},
		},
	}, logger, nil, nil)

	err := executor.AfterSignUp(context.Background(), &types.SignUpResult{User: &models.User{ID: "user-1"}})
	if err != nil {
		t.Fatalf("expected nil error to be returned, got %v", err)
	}
	if !logger.errorCalled {
		t.Fatal("expected logger.Error to be called")
	}
}

type testLogger struct {
	errorCalled bool
}

func (l *testLogger) Debug(msg string, args ...any) {}
func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Warn(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {
	l.errorCalled = true
}

func TestServiceHookExecutor_BothRegistriesAccessibleInHook(t *testing.T) {
	t.Parallel()

	var receivedPluginRegistry models.PluginRegistry
	var receivedService any
	pluginRegistry := &internaltests.TestPluginRegistry{}
	serviceRegistry := &internaltests.TestServiceRegistry{Services: map[string]any{
		models.ServiceUser.String(): "mock-user-service",
	}}

	executor := NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{
		SignUp: &types.SignUpServiceHooksConfig{
			BeforeSignUp: func(ctx context.Context, user *models.User) error {
				reg := models.GetPluginRegistryFromContext(ctx)
				if reg == nil {
					return errors.New("plugin registry not found in context")
				}
				receivedPluginRegistry = reg

				ok, svc := models.GetServiceFromContext[string](ctx, models.ServiceUser)
				if !ok {
					return errors.New("service not found in context")
				}
				receivedService = svc
				return nil
			},
		},
	}, nil, pluginRegistry, serviceRegistry)

	err := executor.BeforeSignUp(context.Background(), &models.User{ID: "user-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if receivedPluginRegistry != pluginRegistry {
		t.Fatal("expected the injected plugin registry to be accessible from context")
	}
	if receivedService != "mock-user-service" {
		t.Fatalf("expected 'mock-user-service', got %v", receivedService)
	}
}
