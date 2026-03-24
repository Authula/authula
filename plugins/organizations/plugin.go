package organizations

import (
	"fmt"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
	rootservices "github.com/Authula/authula/services"
)

type OrganizationsPlugin struct {
	globalConfig        *models.Config
	pluginConfig        types.OrganizationsPluginConfig
	ctx                 *models.PluginContext
	logger              models.Logger
	Api                 *API
	organizationRepo    repositories.OrganizationRepository
	invitationRepo      repositories.OrganizationInvitationRepository
	memberRepo          repositories.OrganizationMemberRepository
	teamRepo            repositories.OrganizationTeamRepository
	teamMemberRepo      repositories.OrganizationTeamMemberRepository
	serviceUtils        *services.ServiceUtils
	organizationService *services.OrganizationService
	invitationService   *services.OrganizationInvitationService
	memberService       *services.OrganizationMemberService
	teamService         *services.OrganizationTeamService
	teamMemberService   *services.OrganizationTeamMemberService
	databaseHooks       *OrganizationsHookExecutor
}

func New(config types.OrganizationsPluginConfig) *OrganizationsPlugin {
	config.ApplyDefaults()
	return &OrganizationsPlugin{pluginConfig: config}
}

func (p *OrganizationsPlugin) Metadata() models.PluginMetadata {
	return models.PluginMetadata{
		ID:          models.PluginOrganizations.String(),
		Version:     "1.0.0",
		Description: "Provides organizations, invitations, members and teams.",
	}
}

func (p *OrganizationsPlugin) Config() any {
	return p.pluginConfig
}

func (p *OrganizationsPlugin) Init(ctx *models.PluginContext) error {
	p.ctx = ctx
	p.logger = ctx.Logger
	p.globalConfig = ctx.GetConfig()

	if err := util.LoadPluginConfig(p.globalConfig, p.Metadata().ID, &p.pluginConfig); err != nil {
		return err
	}

	userService, ok := ctx.ServiceRegistry.Get(models.ServiceUser.String()).(rootservices.UserService)
	if !ok {
		return fmt.Errorf("user service not available in service registry")
	}

	mailerService, ok := ctx.ServiceRegistry.Get(models.ServiceMailer.String()).(rootservices.MailerService)
	if !ok {
		return fmt.Errorf("mailer service not available in service registry")
	}

	accessControlService, ok := ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()).(rootservices.AccessControlService)
	if !ok {
		return fmt.Errorf("access control service not available in service registry")
	}

	p.databaseHooks = NewOrganizationsHookExecutor(p.pluginConfig.DatabaseHooks)
	p.organizationRepo = repositories.NewBunOrganizationRepository(ctx.DB, p.databaseHooks)
	p.invitationRepo = repositories.NewBunOrganizationInvitationRepository(ctx.DB, p.databaseHooks)
	p.memberRepo = repositories.NewBunOrganizationMemberRepository(ctx.DB, p.databaseHooks)
	p.teamRepo = repositories.NewBunOrganizationTeamRepository(ctx.DB, p.databaseHooks)
	p.teamMemberRepo = repositories.NewBunOrganizationTeamMemberRepository(ctx.DB, p.databaseHooks)

	p.serviceUtils = services.NewServiceUtils(p.organizationRepo, p.memberRepo, p.teamRepo, p.teamMemberRepo)
	p.organizationService = services.NewOrganizationService(p.organizationRepo, p.memberRepo, p.serviceUtils, accessControlService, p.pluginConfig.OrganizationsLimit, ctx.DB)
	p.invitationService = services.NewOrganizationInvitationService(ctx.DB, p.globalConfig, &p.pluginConfig, p.logger, ctx.EventBus, userService, mailerService, accessControlService, p.organizationRepo, p.invitationRepo, p.memberRepo, p.serviceUtils)
	p.memberService = services.NewOrganizationMemberService(userService, accessControlService, p.organizationRepo, p.memberRepo, p.pluginConfig.MembersLimit, ctx.DB, p.serviceUtils)
	p.teamService = services.NewOrganizationTeamService(p.organizationRepo, p.memberRepo, p.teamRepo, p.teamMemberRepo, p.serviceUtils, ctx.DB)
	p.teamMemberService = services.NewOrganizationTeamMemberService(p.organizationRepo, p.memberRepo, p.teamRepo, p.teamMemberRepo, p.serviceUtils)
	p.Api = BuildAPI(p)

	return nil
}

func (p *OrganizationsPlugin) Migrations(provider string) []migrations.Migration {
	return organizationsMigrationsForProvider(provider)
}

func (p *OrganizationsPlugin) DependsOn() []string {
	return []string{models.PluginAccessControl.String()}
}

func (p *OrganizationsPlugin) Routes() []models.Route {
	return Routes(p)
}

func (p *OrganizationsPlugin) Close() error {
	return nil
}
