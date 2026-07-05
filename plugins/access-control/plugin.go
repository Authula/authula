package accesscontrol

import (
	"context"
	"fmt"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	accesscontrolconstants "github.com/Authula/authula/plugins/access-control/constants"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
	rootservices "github.com/Authula/authula/services"
)

type AccessControlPlugin struct {
	config               types.AccessControlPluginConfig
	ctx                  *models.PluginContext
	logger               models.Logger
	accessControlService *services.AccessControlService
	Api                  *API
}

func New(config types.AccessControlPluginConfig) *AccessControlPlugin {
	config.ApplyDefaults()
	return &AccessControlPlugin{config: config}
}

func (p *AccessControlPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginAccessControl.String(),
		Version:     "1.0.0",
		Description: "Provides access control functionality.",
	}
}

func (p *AccessControlPlugin) Config() any {
	return p.config
}

func (p *AccessControlPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger

	if err := util.LoadPluginConfig(ctx.GetConfig(), p.Metadata().ID, &p.config); err != nil {
		return err
	}

	rolesRepo := repositories.NewBunRolesRepository(ctx.DB)
	permissionsRepo := repositories.NewBunPermissionsRepository(ctx.DB)
	rolePermissionsRepo := repositories.NewBunRolePermissionsRepository(ctx.DB)
	userRolesRepo := repositories.NewBunUserRolesRepository(ctx.DB)
	userPermissionsRepo := repositories.NewBunUserPermissionsRepository(ctx.DB)

	authorizer := rootservices.NewDefaultAuthorizer()

	rolesService := services.NewRolesService(rolesRepo, rolePermissionsRepo, userRolesRepo, authorizer)
	permissionsService := services.NewPermissionsService(permissionsRepo, rolePermissionsRepo, authorizer)
	rolePermissionsService := services.NewRolePermissionsService(rolesRepo, permissionsRepo, rolePermissionsRepo, authorizer)
	userRolesService := services.NewUserRolesService(userRolesRepo, rolesRepo, authorizer)
	userPermissionsService := services.NewUserPermissionsService(userPermissionsRepo, authorizer)

	accessControlService := services.NewAccessControlService(rolesService, userRolesService, permissionsRepo)
	p.accessControlService = accessControlService
	if err := p.ensurePermissions(); err != nil {
		return err
	}

	useCases := usecases.NewAccessControlUseCases(
		usecases.NewRolesUseCase(rolesService),
		usecases.NewPermissionsUseCase(permissionsService),
		usecases.NewRolePermissionsUseCase(rolePermissionsService),
		usecases.NewUserRolesUseCase(userRolesService),
		usecases.NewUserPermissionsUseCase(userPermissionsService),
	)
	p.Api = NewAPI(useCases)

	ctx.ServiceRegistry.Register(models.ServiceAccessControl.String(), accessControlService)

	return nil
}

func (p *AccessControlPlugin) Migrations(provider string) []migrations.Migration {
	return accessControlMigrationsForProvider(provider)
}

func (p *AccessControlPlugin) DependsOn() []string {
	return []string{}
}

func (p *AccessControlPlugin) Routes() []models.Route {
	return Routes(p.Api)
}

func (p *AccessControlPlugin) Close() error {
	return nil
}

func (p *AccessControlPlugin) ensurePermissions() error {
	if err := p.accessControlService.EnsurePermissions(context.Background(), []rootservices.PermissionDefinition{
		{Key: accesscontrolconstants.All, Description: "All access control permissions"},
		{Key: accesscontrolconstants.RolesCreatePermission, Description: "Create roles in the access control system"},
		{Key: accesscontrolconstants.RolesListPermission, Description: "List roles in the access control system"},
		{Key: accesscontrolconstants.RolesReadPermission, Description: "Read a role in the access control system"},
		{Key: accesscontrolconstants.RolesUpdatePermission, Description: "Update a role in the access control system"},
		{Key: accesscontrolconstants.RolesDeletePermission, Description: "Delete a role from the access control system"},
		{Key: accesscontrolconstants.PermissionsCreatePermission, Description: "Create permissions in the access control system"},
		{Key: accesscontrolconstants.PermissionsListPermission, Description: "List permissions in the access control system"},
		{Key: accesscontrolconstants.PermissionsReadPermission, Description: "Read a permission in the access control system"},
		{Key: accesscontrolconstants.PermissionsUpdatePermission, Description: "Update a permission in the access control system"},
		{Key: accesscontrolconstants.PermissionsDeletePermission, Description: "Delete a permission from the access control system"},
		{Key: accesscontrolconstants.RolePermissionsAssignPermission, Description: "Assign permissions to a role"},
		{Key: accesscontrolconstants.RolePermissionsReadPermission, Description: "Read permissions assigned to a role"},
		{Key: accesscontrolconstants.RolePermissionsRemovePermission, Description: "Remove permissions from a role"},
		{Key: accesscontrolconstants.UserRolesAssignPermission, Description: "Assign roles to a user"},
		{Key: accesscontrolconstants.UserRolesReadPermission, Description: "Read roles assigned to a user"},
		{Key: accesscontrolconstants.UserRolesRemovePermission, Description: "Remove roles from a user"},
		{Key: accesscontrolconstants.UserPermissionsReadPermission, Description: "Read permissions for a user"},
		{Key: accesscontrolconstants.UserPermissionsCheckPermission, Description: "Check permissions for a user"},
	}); err != nil {
		return fmt.Errorf("failed to ensure access control permissions: %w", err)
	}

	return nil
}
