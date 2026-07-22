package organizations

import (
	"context"
	"fmt"

	emailtmpl "github.com/Authula/authula/core/email/template"
	"github.com/Authula/authula/migrations"
	"github.com/Authula/authula/models"
	orgconstants "github.com/Authula/authula/plugins/organizations/constants"
	"github.com/Authula/authula/plugins/organizations/repositories"
	"github.com/Authula/authula/plugins/organizations/services"
	"github.com/Authula/authula/plugins/organizations/types"
	"github.com/Authula/authula/plugins/organizations/usecases"
	rootservices "github.com/Authula/authula/services"
	"github.com/Authula/authula/util"
)

type OrganizationsPlugin struct {
	globalConfig         *models.Config
	pluginConfig         types.OrganizationsPluginConfig
	ctx                  *models.PluginContext
	logger               models.Logger
	Api                  *API
	organizationRepo     repositories.OrganizationRepository
	invitationRepo       repositories.OrganizationInvitationRepository
	memberRepo           repositories.OrganizationMemberRepository
	teamRepo             repositories.OrganizationTeamRepository
	teamMemberRepo       repositories.OrganizationTeamMemberRepository
	serviceUtils         *services.ServiceUtils
	organizationService  services.OrganizationService
	invitationService    services.OrganizationInvitationService
	memberService        services.OrganizationMemberService
	teamService          services.OrganizationTeamService
	teamMemberService    services.OrganizationTeamMemberService
	accessControlService rootservices.AccessControlService
	hooksExecutor        *services.ServiceHookExecutor
	emailTemplateManager *emailtmpl.Manager
	useCases             *usecases.UseCases
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
		p.logger.Warn("mailer service not available in service registry: automatic email sending will be disabled for the organizations plugin")
	}

	accessControlService, ok := ctx.ServiceRegistry.Get(models.ServiceAccessControl.String()).(rootservices.AccessControlService)
	if !ok {
		return fmt.Errorf("access control service not available in service registry")
	}
	p.accessControlService = accessControlService
	if err := p.ensurePermissions(); err != nil {
		return err
	}

	p.hooksExecutor = services.NewServiceHookExecutor(p.pluginConfig.ServiceHooks, ctx.ServiceRegistry)
	p.organizationRepo = repositories.NewBunOrganizationRepository(ctx.DB)
	p.invitationRepo = repositories.NewBunOrganizationInvitationRepository(ctx.DB)
	p.memberRepo = repositories.NewBunOrganizationMemberRepository(ctx.DB)
	p.teamRepo = repositories.NewBunOrganizationTeamRepository(ctx.DB)
	p.teamMemberRepo = repositories.NewBunOrganizationTeamMemberRepository(ctx.DB)

	p.serviceUtils = services.NewServiceUtils(p.organizationRepo, p.memberRepo, p.teamRepo, p.teamMemberRepo)
	p.organizationService = services.NewOrganizationService(p.organizationRepo, p.memberRepo, p.serviceUtils, accessControlService, p.pluginConfig.OrganizationsLimit, ctx.DB, p.hooksExecutor)
	emailTemplateManager, err := newOrganizationEmailTemplateManager()
	if err != nil {
		return fmt.Errorf("failed to initialize organization email templates: %w", err)
	}
	p.emailTemplateManager = emailTemplateManager

	p.invitationService = services.NewOrganizationInvitationService(ctx.DB, p.globalConfig, &p.pluginConfig, p.logger, ctx.EventBus, userService, mailerService, accessControlService, p.organizationRepo, p.invitationRepo, p.memberRepo, p.serviceUtils, p.emailTemplateManager, p.hooksExecutor)
	p.memberService = services.NewOrganizationMemberService(userService, accessControlService, p.organizationRepo, p.memberRepo, p.pluginConfig.MembersLimit, ctx.DB, p.serviceUtils, p.hooksExecutor)
	p.teamService = services.NewOrganizationTeamService(p.organizationRepo, p.memberRepo, p.teamRepo, p.teamMemberRepo, p.serviceUtils, ctx.DB, p.hooksExecutor)
	p.teamMemberService = services.NewOrganizationTeamMemberService(p.organizationRepo, p.memberRepo, p.teamRepo, p.teamMemberRepo, p.serviceUtils, p.hooksExecutor)

	authorizer := rootservices.NewDefaultAuthorizer()
	p.useCases = usecases.NewUseCases(p.organizationService, p.invitationService, p.memberService, p.teamService, p.teamMemberService, authorizer)

	p.Api = BuildAPI(p)

	ctx.ServiceRegistry.Register(models.ServiceOrganization.String(), OrganizationLookupService(p.Api))

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

func (p *OrganizationsPlugin) ensurePermissions() error {
	if err := p.accessControlService.EnsurePermissions(context.Background(), []rootservices.PermissionDefinition{
		{Key: orgconstants.All, Description: "All organizations permissions"},
		{Key: orgconstants.OrganizationsReadPermission, Description: "Read organization details"},
		{Key: orgconstants.OrganizationsUpdatePermission, Description: "Update organization details"},
		{Key: orgconstants.OrganizationsDeletePermission, Description: "Delete organizations"},
		{Key: orgconstants.OrganizationsMembersAddPermission, Description: "Add members to an organization"},
		{Key: orgconstants.OrganizationsMembersListPermission, Description: "List organization members"},
		{Key: orgconstants.OrganizationsMembersReadPermission, Description: "Read organization member details"},
		{Key: orgconstants.OrganizationsMembersUpdatePermission, Description: "Update organization member details"},
		{Key: orgconstants.OrganizationsMembersRemovePermission, Description: "Remove members from an organization"},
		{Key: orgconstants.OrganizationsTeamsCreatePermission, Description: "Create organization teams"},
		{Key: orgconstants.OrganizationsTeamsListPermission, Description: "List organization teams"},
		{Key: orgconstants.OrganizationsTeamsReadPermission, Description: "Read organization team details"},
		{Key: orgconstants.OrganizationsTeamsUpdatePermission, Description: "Update organization team details"},
		{Key: orgconstants.OrganizationsTeamsDeletePermission, Description: "Delete organization teams"},
		{Key: orgconstants.OrganizationsTeamMembersAddPermission, Description: "Add members to an organization team"},
		{Key: orgconstants.OrganizationsTeamMembersListPermission, Description: "List organization team members"},
		{Key: orgconstants.OrganizationsTeamMembersReadPermission, Description: "Read organization team member details"},
		{Key: orgconstants.OrganizationsTeamMembersRemovePermission, Description: "Remove members from an organization team"},
		{Key: orgconstants.OrganizationsInvitationsCreatePermission, Description: "Create organization invitations"},
		{Key: orgconstants.OrganizationsInvitationsListPermission, Description: "List organization invitations"},
		{Key: orgconstants.OrganizationsInvitationsReadPermission, Description: "Read organization invitation details"},
		{Key: orgconstants.OrganizationsInvitationsRevokePermission, Description: "Revoke organization invitations"},
	}); err != nil {
		return fmt.Errorf("failed to ensure organization permissions: %w", err)
	}

	return nil
}
