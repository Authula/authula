package services

import (
	"context"

	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
)

type ServiceHookExecutor struct {
	serviceHooksConfig *types.EmailPasswordServiceHooksConfig
	logger             models.Logger
	pluginRegistry     models.PluginRegistry
	registry           models.ServiceRegistry
}

func NewServiceHookExecutor(serviceHooksConfig *types.EmailPasswordServiceHooksConfig, logger models.Logger, pluginRegistry models.PluginRegistry, registry models.ServiceRegistry) *ServiceHookExecutor {
	return &ServiceHookExecutor{serviceHooksConfig: serviceHooksConfig, logger: logger, pluginRegistry: pluginRegistry, registry: registry}
}

func (e *ServiceHookExecutor) BeforeSignUp(ctx context.Context, user *models.User) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.SignUp == nil || e.serviceHooksConfig.SignUp.BeforeSignUp == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.serviceHooksConfig.SignUp.BeforeSignUp(ctx, user)
}

func (e *ServiceHookExecutor) AfterSignUp(ctx context.Context, result *types.SignUpResult) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.SignUp == nil || e.serviceHooksConfig.SignUp.AfterSignUp == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.serviceHooksConfig.SignUp.AfterSignUp(ctx, result); err != nil {
		e.logger.Error("after sign up hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeSignIn(ctx context.Context, user *models.User) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.SignIn == nil || e.serviceHooksConfig.SignIn.BeforeSignIn == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.serviceHooksConfig.SignIn.BeforeSignIn(ctx, user)
}

func (e *ServiceHookExecutor) AfterSignIn(ctx context.Context, result *types.SignInResult) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.SignIn == nil || e.serviceHooksConfig.SignIn.AfterSignIn == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.serviceHooksConfig.SignIn.AfterSignIn(ctx, result); err != nil {
		e.logger.Error("after sign in hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) AfterVerifyEmail(ctx context.Context, user *models.User, verificationType models.VerificationType) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.EmailVerification == nil || e.serviceHooksConfig.EmailVerification.AfterVerifyEmail == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.serviceHooksConfig.EmailVerification.AfterVerifyEmail(ctx, user, verificationType); err != nil {
		e.logger.Error("after verify email hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeRequestPasswordReset(ctx context.Context, user *models.User) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.PasswordReset == nil || e.serviceHooksConfig.PasswordReset.BeforeRequestPasswordReset == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.serviceHooksConfig.PasswordReset.BeforeRequestPasswordReset(ctx, user)
}

func (e *ServiceHookExecutor) BeforeChangePassword(ctx context.Context, user *models.User, newPassword string) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.PasswordChange == nil || e.serviceHooksConfig.PasswordChange.BeforeChangePassword == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.serviceHooksConfig.PasswordChange.BeforeChangePassword(ctx, user, newPassword)
}

func (e *ServiceHookExecutor) AfterChangePassword(ctx context.Context, user *models.User) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.PasswordChange == nil || e.serviceHooksConfig.PasswordChange.AfterChangePassword == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.serviceHooksConfig.PasswordChange.AfterChangePassword(ctx, user); err != nil {
		e.logger.Error("after change password hook failed", "error", err.Error())
	}
	return nil
}

func (e *ServiceHookExecutor) BeforeRequestEmailChange(ctx context.Context, user *models.User) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.EmailChange == nil || e.serviceHooksConfig.EmailChange.BeforeRequestEmailChange == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	return e.serviceHooksConfig.EmailChange.BeforeRequestEmailChange(ctx, user)
}

func (e *ServiceHookExecutor) AfterEmailChanged(ctx context.Context, user *models.User, oldEmail, newEmail string) error {
	if e == nil || e.serviceHooksConfig == nil || e.serviceHooksConfig.EmailChange == nil || e.serviceHooksConfig.EmailChange.AfterEmailChanged == nil {
		return nil
	}
	ctx = models.NewContextWithPluginRegistry(ctx, e.pluginRegistry)
	ctx = models.NewContextWithServiceRegistry(ctx, e.registry)
	if err := e.serviceHooksConfig.EmailChange.AfterEmailChanged(ctx, user, oldEmail, newEmail); err != nil {
		e.logger.Error("after email changed hook failed", "error", err.Error())
	}
	return nil
}
