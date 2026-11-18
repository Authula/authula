package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

// hookRecord captures the arguments and invocation state of a service hook.
type hookRecord struct {
	called           bool
	user             *models.User
	verificationType models.VerificationType
	password         string
	oldEmail         string
	newEmail         string
}

func assertHookRecord(t *testing.T, want, got hookRecord) {
	t.Helper()

	assert.True(t, got.called, "expected hook to be called")
	assert.Equal(t, want.user, got.user)
	assert.Equal(t, want.verificationType, got.verificationType)
	assert.Equal(t, want.password, got.password)
	assert.Equal(t, want.oldEmail, got.oldEmail)
	assert.Equal(t, want.newEmail, got.newEmail)
}

func TestServiceHookExecutor_NoopWhenHooksNotConfigured(t *testing.T) {
	t.Parallel()

	user := &models.User{ID: "user-1", Email: "test@example.com"}
	signUpResult := &types.SignUpResult{User: user}
	signInResult := &types.SignInResult{User: user}

	calls := []struct {
		name string
		call func(executor *ServiceHookExecutor) error
	}{
		{
			name: "BeforeSignUp",
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeSignUp(context.Background(), user)
			},
		},
		{
			name: "AfterSignUp",
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterSignUp(context.Background(), signUpResult)
			},
		},
		{
			name: "BeforeSignIn",
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeSignIn(context.Background(), user)
			},
		},
		{
			name: "AfterSignIn",
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterSignIn(context.Background(), signInResult)
			},
		},
		{
			name: "AfterVerifyEmail",
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterVerifyEmail(context.Background(), user, models.TypeEmailVerification)
			},
		},
		{
			name: "BeforeRequestPasswordReset",
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeRequestPasswordReset(context.Background(), user)
			},
		},
		{
			name: "BeforeChangePassword",
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeChangePassword(context.Background(), user, "newpass")
			},
		},
		{
			name: "AfterChangePassword",
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterChangePassword(context.Background(), user)
			},
		},
		{
			name: "BeforeRequestEmailChange",
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeRequestEmailChange(context.Background(), user)
			},
		},
		{
			name: "AfterEmailChanged",
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterEmailChanged(context.Background(), user, "old@test.com", "new@test.com")
			},
		},
	}

	tests := []struct {
		name     string
		executor func() *ServiceHookExecutor
	}{
		{
			name: "nil hooks config",
			executor: func() *ServiceHookExecutor {
				return NewServiceHookExecutor(nil, nil, nil, nil)
			},
		},
		{
			name: "nil executor",
			executor: func() *ServiceHookExecutor {
				return nil
			},
		},
		{
			name: "empty hooks config",
			executor: func() *ServiceHookExecutor {
				return NewServiceHookExecutor(&types.EmailPasswordServiceHooksConfig{}, nil, nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := tt.executor()

			for _, call := range calls {
				assert.NoErrorf(t, call.call(executor), "%s should be a no-op", call.name)
			}
		})
	}
}

func TestServiceHookExecutor_BeforeHooks(t *testing.T) {
	t.Parallel()

	user := &models.User{ID: "user-1", Email: "test@example.com"}
	newPassword := "newpassword123"
	someErr := errors.New("some error")

	beforeSignUpHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			SignUp: &types.SignUpServiceHooksConfig{
				BeforeSignUp: func(ctx context.Context, u *models.User) error {
					record.called = true
					record.user = u
					return hookErr
				},
			},
		}
	}

	beforeSignInHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			SignIn: &types.SignInServiceHooksConfig{
				BeforeSignIn: func(ctx context.Context, u *models.User) error {
					record.called = true
					record.user = u
					return hookErr
				},
			},
		}
	}

	beforeRequestPasswordResetHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			PasswordReset: &types.PasswordResetServiceHooksConfig{
				BeforeRequestPasswordReset: func(ctx context.Context, u *models.User) error {
					record.called = true
					record.user = u
					return hookErr
				},
			},
		}
	}

	beforeChangePasswordHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			PasswordChange: &types.PasswordChangeServiceHooksConfig{
				BeforeChangePassword: func(ctx context.Context, u *models.User, password string) error {
					record.called = true
					record.user = u
					record.password = password
					return hookErr
				},
			},
		}
	}

	beforeRequestEmailChangeHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			EmailChange: &types.EmailChangeServiceHooksConfig{
				BeforeRequestEmailChange: func(ctx context.Context, u *models.User) error {
					record.called = true
					record.user = u
					return hookErr
				},
			},
		}
	}

	hookCases := []struct {
		name  string
		hooks func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig
		call  func(executor *ServiceHookExecutor) error
		want  hookRecord
	}{
		{
			name:  "before sign up",
			hooks: beforeSignUpHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeSignUp(context.Background(), user)
			},
			want: hookRecord{called: true, user: user},
		},
		{
			name:  "before sign in",
			hooks: beforeSignInHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeSignIn(context.Background(), user)
			},
			want: hookRecord{called: true, user: user},
		},
		{
			name:  "before request password reset",
			hooks: beforeRequestPasswordResetHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeRequestPasswordReset(context.Background(), user)
			},
			want: hookRecord{called: true, user: user},
		},
		{
			name:  "before change password",
			hooks: beforeChangePasswordHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeChangePassword(context.Background(), user, newPassword)
			},
			want: hookRecord{called: true, user: user, password: newPassword},
		},
		{
			name:  "before request email change",
			hooks: beforeRequestEmailChangeHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeRequestEmailChange(context.Background(), user)
			},
			want: hookRecord{called: true, user: user},
		},
	}

	errCases := []struct {
		name    string
		hookErr error
	}{
		{name: "succeeds"},
		{name: "returns hook error", hookErr: someErr},
	}

	for _, hookCase := range hookCases {
		for _, errCase := range errCases {
			t.Run(hookCase.name+"/"+errCase.name, func(t *testing.T) {
				t.Parallel()

				record := &hookRecord{}
				executor := NewServiceHookExecutor(hookCase.hooks(record, errCase.hookErr), nil, nil, nil)

				err := hookCase.call(executor)

				assert.ErrorIs(t, err, errCase.hookErr)
				assertHookRecord(t, hookCase.want, *record)
			})
		}
	}
}

func TestServiceHookExecutor_AfterHooks(t *testing.T) {
	t.Parallel()

	user := &models.User{ID: "user-1", Email: "test@example.com"}
	oldEmail := "old@test.com"
	newEmail := "new@test.com"
	someErr := errors.New("some error")

	afterSignUpHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			SignUp: &types.SignUpServiceHooksConfig{
				AfterSignUp: func(ctx context.Context, result *types.SignUpResult) error {
					record.called = true
					record.user = result.User
					return hookErr
				},
			},
		}
	}

	afterSignInHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			SignIn: &types.SignInServiceHooksConfig{
				AfterSignIn: func(ctx context.Context, result *types.SignInResult) error {
					record.called = true
					record.user = result.User
					return hookErr
				},
			},
		}
	}

	afterVerifyEmailHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			EmailVerification: &types.EmailVerificationServiceHooksConfig{
				AfterVerifyEmail: func(ctx context.Context, u *models.User, verificationType models.VerificationType) error {
					record.called = true
					record.user = u
					record.verificationType = verificationType
					return hookErr
				},
			},
		}
	}

	afterChangePasswordHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			PasswordChange: &types.PasswordChangeServiceHooksConfig{
				AfterChangePassword: func(ctx context.Context, u *models.User) error {
					record.called = true
					record.user = u
					return hookErr
				},
			},
		}
	}

	afterEmailChangedHooks := func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig {
		return &types.EmailPasswordServiceHooksConfig{
			EmailChange: &types.EmailChangeServiceHooksConfig{
				AfterEmailChanged: func(ctx context.Context, u *models.User, old, new string) error {
					record.called = true
					record.user = u
					record.oldEmail = old
					record.newEmail = new
					return hookErr
				},
			},
		}
	}

	hookCases := []struct {
		name    string
		hooks   func(record *hookRecord, hookErr error) *types.EmailPasswordServiceHooksConfig
		call    func(executor *ServiceHookExecutor) error
		want    hookRecord
		wantLog string
	}{
		{
			name:  "after sign up",
			hooks: afterSignUpHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterSignUp(context.Background(), &types.SignUpResult{User: user})
			},
			want:    hookRecord{called: true, user: user},
			wantLog: "after sign up hook failed",
		},
		{
			name:  "after sign in",
			hooks: afterSignInHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterSignIn(context.Background(), &types.SignInResult{User: user})
			},
			want:    hookRecord{called: true, user: user},
			wantLog: "after sign in hook failed",
		},
		{
			name:  "after verify email",
			hooks: afterVerifyEmailHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterVerifyEmail(context.Background(), user, models.TypeEmailVerification)
			},
			want: hookRecord{
				called:           true,
				user:             user,
				verificationType: models.TypeEmailVerification,
			},
			wantLog: "after verify email hook failed",
		},
		{
			name:  "after change password",
			hooks: afterChangePasswordHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterChangePassword(context.Background(), user)
			},
			want:    hookRecord{called: true, user: user},
			wantLog: "after change password hook failed",
		},
		{
			name:  "after email changed",
			hooks: afterEmailChangedHooks,
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterEmailChanged(context.Background(), user, oldEmail, newEmail)
			},
			want: hookRecord{
				called:   true,
				user:     user,
				oldEmail: oldEmail,
				newEmail: newEmail,
			},
			wantLog: "after email changed hook failed",
		},
	}

	errCases := []struct {
		name    string
		hookErr error
	}{
		{name: "succeeds"},
		{name: "logs hook error and returns nil", hookErr: someErr},
	}

	for _, hookCase := range hookCases {
		for _, errCase := range errCases {
			t.Run(hookCase.name+"/"+errCase.name, func(t *testing.T) {
				t.Parallel()

				record := &hookRecord{}
				logger := &internaltests.RecordingLogger{}
				executor := NewServiceHookExecutor(hookCase.hooks(record, errCase.hookErr), logger, nil, nil)

				err := hookCase.call(executor)

				assert.NoError(t, err)
				assertHookRecord(t, hookCase.want, *record)

				if errCase.hookErr == nil {
					assert.Empty(t, logger.ErrorCalls)
					return
				}
				assert.Equal(t, []string{hookCase.wantLog}, logger.ErrorCalls)
			})
		}
	}
}

func TestServiceHookExecutor_RegistriesAreInjectedIntoContext(t *testing.T) {
	t.Parallel()

	pluginRegistry := &internaltests.TestPluginRegistry{}
	serviceRegistry := &internaltests.TestServiceRegistry{Services: map[string]any{
		models.ServiceUser.String(): "mock-user-service",
	}}
	user := &models.User{ID: "user-1"}

	assertRegistries := func(t *testing.T, ctx context.Context) error {
		t.Helper()

		if got := models.GetPluginRegistryFromContext(ctx); got != pluginRegistry {
			return errors.New("plugin registry not found in context")
		}
		ok, svc := models.GetServiceFromContext[string](ctx, models.ServiceUser)
		if !ok {
			return errors.New("user service not found in context")
		}
		if svc != "mock-user-service" {
			return errors.New("unexpected user service in context")
		}
		return nil
	}

	tests := []struct {
		name  string
		hooks func(t *testing.T) *types.EmailPasswordServiceHooksConfig
		call  func(executor *ServiceHookExecutor) error
	}{
		{
			name: "before sign up",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					SignUp: &types.SignUpServiceHooksConfig{
						BeforeSignUp: func(ctx context.Context, _ *models.User) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeSignUp(context.Background(), user)
			},
		},
		{
			name: "after sign up",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					SignUp: &types.SignUpServiceHooksConfig{
						AfterSignUp: func(ctx context.Context, _ *types.SignUpResult) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterSignUp(context.Background(), &types.SignUpResult{User: user})
			},
		},
		{
			name: "before sign in",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					SignIn: &types.SignInServiceHooksConfig{
						BeforeSignIn: func(ctx context.Context, _ *models.User) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeSignIn(context.Background(), user)
			},
		},
		{
			name: "after sign in",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					SignIn: &types.SignInServiceHooksConfig{
						AfterSignIn: func(ctx context.Context, _ *types.SignInResult) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterSignIn(context.Background(), &types.SignInResult{User: user})
			},
		},
		{
			name: "after verify email",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					EmailVerification: &types.EmailVerificationServiceHooksConfig{
						AfterVerifyEmail: func(ctx context.Context, _ *models.User, _ models.VerificationType) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterVerifyEmail(context.Background(), user, models.TypeEmailVerification)
			},
		},
		{
			name: "before request password reset",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					PasswordReset: &types.PasswordResetServiceHooksConfig{
						BeforeRequestPasswordReset: func(ctx context.Context, _ *models.User) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeRequestPasswordReset(context.Background(), user)
			},
		},
		{
			name: "before change password",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					PasswordChange: &types.PasswordChangeServiceHooksConfig{
						BeforeChangePassword: func(ctx context.Context, _ *models.User, _ string) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeChangePassword(context.Background(), user, "newpass")
			},
		},
		{
			name: "after change password",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					PasswordChange: &types.PasswordChangeServiceHooksConfig{
						AfterChangePassword: func(ctx context.Context, _ *models.User) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterChangePassword(context.Background(), user)
			},
		},
		{
			name: "before request email change",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					EmailChange: &types.EmailChangeServiceHooksConfig{
						BeforeRequestEmailChange: func(ctx context.Context, _ *models.User) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.BeforeRequestEmailChange(context.Background(), user)
			},
		},
		{
			name: "after email changed",
			hooks: func(t *testing.T) *types.EmailPasswordServiceHooksConfig {
				return &types.EmailPasswordServiceHooksConfig{
					EmailChange: &types.EmailChangeServiceHooksConfig{
						AfterEmailChanged: func(ctx context.Context, _ *models.User, _, _ string) error {
							return assertRegistries(t, ctx)
						},
					},
				}
			},
			call: func(executor *ServiceHookExecutor) error {
				return executor.AfterEmailChanged(context.Background(), user, "old@test.com", "new@test.com")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := NewServiceHookExecutor(tt.hooks(t), nil, pluginRegistry, serviceRegistry)

			assert.NoError(t, tt.call(executor))
		})
	}
}
