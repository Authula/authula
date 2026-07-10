package admin

import (
	"context"
	"fmt"

	corerepos "github.com/Authula/authula/core/repositories"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	adminconstants "github.com/Authula/authula/plugins/admin/constants"
	"github.com/Authula/authula/plugins/admin/repositories"
	"github.com/Authula/authula/plugins/admin/types"
	"github.com/Authula/authula/plugins/admin/usecases"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type AdminPlugin struct {
	globalConfig         *models.Config
	config               types.AdminPluginConfig
	pluginCtx            *models.PluginContext
	logger               models.Logger
	Api                  *API
	sessionService       rootservices.SessionService
	tokenService         rootservices.TokenService
	accessControlService rootservices.AccessControlService
}

func New(config types.AdminPluginConfig) *AdminPlugin {
	config.ApplyDefaults()
	return &AdminPlugin{config: config}
}

func (p *AdminPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginAdmin.String(),
		Version:     "1.0.0",
		Description: "Provides admin operations for users, state, and impersonation.",
	}
}

func (p *AdminPlugin) Config() any {
	return p.config
}

func (p *AdminPlugin) Init(ctx *models.PluginContext) error {
	p.globalConfig = ctx.GetConfig()
	p.pluginCtx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	impersonationRepo := repositories.NewBunImpersonationRepository(ctx.DB)
	userStateRepo := repositories.NewBunUserStateRepository(ctx.DB)
	sessionStateRepo := repositories.NewBunSessionStateRepository(ctx.DB)

	coreUserRepo := corerepos.NewBunUserRepository(ctx.DB)
	coreAccountRepo := corerepos.NewBunAccountRepository(ctx.DB)

	sessionService, ok := ctx.ServiceRegistry.Get(models.ServiceSession.String()).(rootservices.SessionService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceSession.String())
	}
	p.sessionService = sessionService

	tokenService, ok := ctx.ServiceRegistry.Get(models.ServiceToken.String()).(rootservices.TokenService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceToken.String())
	}
	p.tokenService = tokenService

	passwordService, ok := ctx.ServiceRegistry.Get(models.ServicePassword.String()).(rootservices.PasswordService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServicePassword.String())
	}

	accessControlService, ok := ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()).(rootservices.AccessControlService)
	if !ok {
		return fmt.Errorf("required service %s is not registered", models.ServiceAccessControl.String())
	}
	p.accessControlService = accessControlService
	if err := p.ensurePermissions(); err != nil {
		return err
	}

	authorizer := rootservices.NewDefaultAuthorizer()

	adminUseCases := usecases.NewAdminUseCases(
		p.config,
		coreUserRepo,
		coreAccountRepo,
		sessionService,
		tokenService,
		passwordService,
		userStateRepo,
		sessionStateRepo,
		impersonationRepo,
		p.globalConfig.Session.ExpiresIn,
		authorizer,
	)
	p.Api = NewAPI(
		adminUseCases,
		impersonationRepo,
		userStateRepo,
		sessionStateRepo,
	)
	ctx.ServiceRegistry.Register(models.ServiceAdmin.String(), p.Api)

	return nil
}

func (p *AdminPlugin) Migrations(provider string) []migrations.Migration {
	return adminMigrationsForProvider(provider)
}

func (p *AdminPlugin) DependsOn() []string {
	return []string{models.PluginAccessControl.String()}
}

func (p *AdminPlugin) Routes() []models.Route {
	return p.buildRoutes(p.Api)
}

func (p *AdminPlugin) Close() error {
	return nil
}

func (p *AdminPlugin) ensurePermissions() error {
	if err := p.accessControlService.EnsurePermissions(context.Background(), []rootservices.PermissionDefinition{
		{Key: adminconstants.All, Description: "All admin permissions"},
		{Key: adminconstants.UsersCreatePermission, Description: "Create users"},
		{Key: adminconstants.UsersListPermission, Description: "List users"},
		{Key: adminconstants.UsersReadPermission, Description: "Read user details"},
		{Key: adminconstants.UsersUpdatePermission, Description: "Update user details"},
		{Key: adminconstants.UsersDeletePermission, Description: "Delete users"},
		{Key: adminconstants.AccountsCreatePermission, Description: "Create user accounts"},
		{Key: adminconstants.AccountsListPermission, Description: "List user accounts"},
		{Key: adminconstants.AccountsReadPermission, Description: "Read user account details"},
		{Key: adminconstants.AccountsUpdatePermission, Description: "Update user accounts"},
		{Key: adminconstants.AccountsDeletePermission, Description: "Delete user accounts"},
		{Key: adminconstants.UserStateReadPermission, Description: "Read user state"},
		{Key: adminconstants.UserStateCreatePermission, Description: "Create user state"},
		{Key: adminconstants.UserStateUpdatePermission, Description: "Update user state"},
		{Key: adminconstants.UserStateDeletePermission, Description: "Delete user state"},
		{Key: adminconstants.UserStateBanPermission, Description: "Ban users"},
		{Key: adminconstants.UserStateUnbanPermission, Description: "Unban users"},
		{Key: adminconstants.UserStateListBannedPermission, Description: "List banned users"},
		{Key: adminconstants.UserStateListSessionsPermission, Description: "List user sessions"},
		{Key: adminconstants.SessionStateReadPermission, Description: "Read session state"},
		{Key: adminconstants.SessionStateCreatePermission, Description: "Create session state"},
		{Key: adminconstants.SessionStateUpdatePermission, Description: "Update session state"},
		{Key: adminconstants.SessionStateDeletePermission, Description: "Delete session state"},
		{Key: adminconstants.SessionStateRevokePermission, Description: "Revoke sessions"},
		{Key: adminconstants.SessionStateListRevokedPermission, Description: "List revoked sessions"},
		{Key: adminconstants.ImpersonationsListPermission, Description: "List impersonations"},
		{Key: adminconstants.ImpersonationsReadPermission, Description: "Read impersonation details"},
		{Key: adminconstants.ImpersonationsStartPermission, Description: "Start impersonation"},
		{Key: adminconstants.ImpersonationsStopPermission, Description: "Stop impersonation"},
	}); err != nil {
		return fmt.Errorf("failed to ensure admin permissions: %w", err)
	}

	return nil
}
