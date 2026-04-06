package accesscontrol

import (
	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/access-control/repositories"
	"github.com/Authula/authula/plugins/access-control/services"
	"github.com/Authula/authula/plugins/access-control/types"
	"github.com/Authula/authula/plugins/access-control/usecases"
)

type AccessControlPlugin struct {
	config types.AccessControlPluginConfig
	ctx    *models.PluginContext
	logger models.Logger
	Api    *API
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

	rolesService := services.NewRolesService(rolesRepo, rolePermissionsRepo, userRolesRepo)
	permissionsService := services.NewPermissionsService(permissionsRepo, rolePermissionsRepo)
	rolePermissionsService := services.NewRolePermissionsService(rolesRepo, permissionsRepo, rolePermissionsRepo)
	userRolesService := services.NewUserRolesService(userRolesRepo, rolesRepo)
	userPermissionsService := services.NewUserPermissionsService(userPermissionsRepo)

	useCases := usecases.NewAccessControlUseCases(
		usecases.NewRolesUseCase(rolesService),
		usecases.NewPermissionsUseCase(permissionsService),
		usecases.NewRolePermissionsUseCase(rolePermissionsService),
		usecases.NewUserRolesUseCase(userRolesService),
		usecases.NewUserPermissionsUseCase(userPermissionsService),
	)
	p.Api = NewAPI(useCases)

	accessControlService := NewAccessControlService(p.Api)
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
