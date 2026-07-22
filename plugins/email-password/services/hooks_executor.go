package services

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

type ServiceHookExecutor struct {
	config   *types.EmailPasswordServiceHooksConfig
	logger   models.Logger
	registry models.ServiceRegistry
}

func NewServiceHookExecutor(config *types.EmailPasswordServiceHooksConfig, logger models.Logger, registry models.ServiceRegistry) *ServiceHookExecutor {
	return &ServiceHookExecutor{config: config, logger: logger, registry: registry}
}

func (e *ServiceHookExecutor) BeforeSignUp(ctx context.Context, user *models.User) error {
	if e == nil || e.config == nil || e.config.SignUp == nil || e.config.SignUp.BeforeSignUp == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.SignUp.BeforeSignUp(ctx, user)
}

func (e *ServiceHookExecutor) AfterSignUp(ctx context.Context, result *types.SignUpResult) error {
	if e == nil || e.config == nil || e.config.SignUp == nil || e.config.SignUp.AfterSignUp == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.config.SignUp.AfterSignUp(ctx, result); err != nil {
		e.logger.Error("after sign up hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeSignIn(ctx context.Context, user *models.User) error {
	if e == nil || e.config == nil || e.config.SignIn == nil || e.config.SignIn.BeforeSignIn == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.SignIn.BeforeSignIn(ctx, user)
}

func (e *ServiceHookExecutor) AfterSignIn(ctx context.Context, result *types.SignInResult) error {
	if e == nil || e.config == nil || e.config.SignIn == nil || e.config.SignIn.AfterSignIn == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.config.SignIn.AfterSignIn(ctx, result); err != nil {
		e.logger.Error("after sign in hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) AfterVerifyEmail(ctx context.Context, user *models.User, verificationType models.VerificationType) error {
	if e == nil || e.config == nil || e.config.EmailVerification == nil || e.config.EmailVerification.AfterVerifyEmail == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.config.EmailVerification.AfterVerifyEmail(ctx, user, verificationType); err != nil {
		e.logger.Error("after verify email hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeRequestPasswordReset(ctx context.Context, user *models.User) error {
	if e == nil || e.config == nil || e.config.PasswordReset == nil || e.config.PasswordReset.BeforeRequestPasswordReset == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.PasswordReset.BeforeRequestPasswordReset(ctx, user)
}

func (e *ServiceHookExecutor) BeforeChangePassword(ctx context.Context, user *models.User, newPassword string) error {
	if e == nil || e.config == nil || e.config.PasswordChange == nil || e.config.PasswordChange.BeforeChangePassword == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.PasswordChange.BeforeChangePassword(ctx, user, newPassword)
}

func (e *ServiceHookExecutor) AfterChangePassword(ctx context.Context, user *models.User) error {
	if e == nil || e.config == nil || e.config.PasswordChange == nil || e.config.PasswordChange.AfterChangePassword == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.config.PasswordChange.AfterChangePassword(ctx, user); err != nil {
		e.logger.Error("after change password hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeRequestEmailChange(ctx context.Context, user *models.User) error {
	if e == nil || e.config == nil || e.config.EmailChange == nil || e.config.EmailChange.BeforeRequestEmailChange == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.config.EmailChange.BeforeRequestEmailChange(ctx, user)
}

func (e *ServiceHookExecutor) AfterEmailChanged(ctx context.Context, user *models.User, oldEmail, newEmail string) error {
	if e == nil || e.config == nil || e.config.EmailChange == nil || e.config.EmailChange.AfterEmailChanged == nil {
		return nil
	}
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.config.EmailChange.AfterEmailChanged(ctx, user, oldEmail, newEmail); err != nil {
		e.logger.Error("after email changed hook failed", "error", err.Error())
	}
	return nil
}
