package organizations

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/core/pagination"
	"github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	orgtests "github.com/Authula/authula/plugins/organizations/tests"
	"github.com/Authula/authula/plugins/organizations/types"
	"github.com/Authula/authula/plugins/organizations/usecases"
)

func TestNew_AppliesDefaults(t *testing.T) {
	t.Parallel()

	plugin := New(types.OrganizationsPluginConfig{})

	require.NotNil(t, plugin)
	require.Equal(t, 100, *plugin.pluginConfig.MembersLimit)
	require.Equal(t, 100, *plugin.pluginConfig.InvitationsLimit)
	require.Equal(t, pagination.DefaultMaxLimit, *plugin.pluginConfig.MaxPageLimit)
	require.Equal(t, 24*time.Hour, plugin.pluginConfig.InvitationExpiresIn)
}

func TestNew_PreservesConfiguredValues(t *testing.T) {
	t.Parallel()

	plugin := New(types.OrganizationsPluginConfig{
		Enabled:             true,
		MembersLimit:        new(5),
		InvitationsLimit:    new(7),
		MaxPageLimit:        new(25),
		InvitationExpiresIn: 2 * time.Hour,
	})

	require.True(t, plugin.pluginConfig.Enabled)
	require.Equal(t, 5, *plugin.pluginConfig.MembersLimit)
	require.Equal(t, 7, *plugin.pluginConfig.InvitationsLimit)
	require.Equal(t, 25, *plugin.pluginConfig.MaxPageLimit)
	require.Equal(t, 2*time.Hour, plugin.pluginConfig.InvitationExpiresIn)
}

func TestPlugin_MetadataConfigDependsOnAndClose(t *testing.T) {
	t.Parallel()

	plugin := New(types.OrganizationsPluginConfig{Enabled: true})

	metadata := plugin.Metadata()
	require.Equal(t, models.PluginOrganizations.String(), metadata.ID)
	require.Equal(t, "1.0.0", metadata.Version)
	require.NotEmpty(t, metadata.Description)

	config, ok := plugin.Config().(types.OrganizationsPluginConfig)
	require.True(t, ok)
	require.True(t, config.Enabled)

	require.Equal(t, []string{models.PluginAccessControl.String()}, plugin.DependsOn())
	require.NoError(t, plugin.Close())

	migrations := plugin.Migrations("sqlite")
	require.NotEmpty(t, migrations)
	for _, migration := range migrations {
		require.NotEmpty(t, migration.Version)
		require.NotNil(t, migration.Up)
	}

	require.Empty(t, plugin.Migrations("unknown-provider"))
}

func TestPlugin_Routes(t *testing.T) {
	t.Parallel()

	plugin := &OrganizationsPlugin{
		useCases:     &usecases.UseCases{},
		globalConfig: &models.Config{},
	}

	routes := plugin.Routes()
	require.Len(t, routes, 26)

	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		require.NotNil(t, route.Handler, "%s %s must have a handler", route.Method, route.Path)
		require.NotEmpty(t, route.Middleware, "%s %s must be guarded", route.Method, route.Path)

		key := route.Method + " " + route.Path
		_, duplicate := seen[key]
		require.False(t, duplicate, "duplicate route %s", key)
		seen[key] = struct{}{}
	}

	expected := []string{
		http.MethodPost + " /organizations",
		http.MethodGet + " /organizations",
		http.MethodGet + " /organizations/{organization_id}",
		http.MethodPatch + " /organizations/{organization_id}",
		http.MethodDelete + " /organizations/{organization_id}",
		http.MethodPost + " /organizations/{organization_id}/invitations",
		http.MethodGet + " /organizations/{organization_id}/invitations",
		http.MethodGet + " /organizations/{organization_id}/invitations/{invitation_id}",
		http.MethodPatch + " /organizations/{organization_id}/invitations/{invitation_id}/revoke",
		http.MethodPatch + " /organizations/{organization_id}/invitations/{invitation_id}/accept",
		http.MethodPatch + " /organizations/{organization_id}/invitations/{invitation_id}/reject",
		http.MethodPost + " /organizations/{organization_id}/members",
		http.MethodGet + " /organizations/{organization_id}/members",
		http.MethodGet + " /organizations/{organization_id}/members/{member_id}",
		http.MethodGet + " /organizations/{organization_id}/members/by-user/{user_id}",
		http.MethodPatch + " /organizations/{organization_id}/members/{member_id}",
		http.MethodDelete + " /organizations/{organization_id}/members/{member_id}",
		http.MethodPost + " /organizations/{organization_id}/teams",
		http.MethodGet + " /organizations/{organization_id}/teams",
		http.MethodGet + " /organizations/{organization_id}/teams/{team_id}",
		http.MethodPatch + " /organizations/{organization_id}/teams/{team_id}",
		http.MethodDelete + " /organizations/{organization_id}/teams/{team_id}",
		http.MethodPost + " /organizations/{organization_id}/teams/{team_id}/members",
		http.MethodGet + " /organizations/{organization_id}/teams/{team_id}/members",
		http.MethodGet + " /organizations/{organization_id}/teams/{team_id}/members/{member_id}",
		http.MethodDelete + " /organizations/{organization_id}/teams/{team_id}/members/{member_id}",
	}
	for _, key := range expected {
		_, ok := seen[key]
		require.True(t, ok, "missing route %s", key)
	}
}

func newPluginInitContext(t *testing.T, includeUser, includeMailer bool, accessControl *orgtests.AccessControlServiceStub, config *models.Config) (*models.PluginContext, models.ServiceRegistry, *internaltests.LevelRecordingLogger) {
	t.Helper()

	db := orgtests.SetupRepoDB(t)
	serviceRegistry := plugins.NewServiceRegistry()
	if includeUser {
		serviceRegistry.Register(models.ServiceUser.String(), &internaltests.MockUserService{})
	}
	if includeMailer {
		serviceRegistry.Register(models.ServiceMailer.String(), &internaltests.MockMailerService{})
	}
	if accessControl != nil {
		serviceRegistry.Register(models.ServiceAccessControl.String(), accessControl)
	}
	logger := &internaltests.LevelRecordingLogger{}

	if config == nil {
		config = &models.Config{BaseURL: "https://example.com", BasePath: "/auth"}
	}

	return &models.PluginContext{
		DB:              db,
		Logger:          logger,
		EventBus:        internaltests.NewRecordingEventBus(4),
		ServiceRegistry: serviceRegistry,
		PluginRegistry:  &internaltests.TestPluginRegistry{},
		GetConfig:       func() *models.Config { return config },
	}, serviceRegistry, logger
}

func TestPlugin_Init(t *testing.T) {
	t.Parallel()

	wiredAccessControl := orgtests.NewAccessControlServiceStub()
	failingAccessControl := orgtests.NewAccessControlServiceStub()
	failingAccessControl.Err = context.DeadlineExceeded

	tests := []struct {
		name          string
		includeUser   bool
		includeMailer bool
		accessControl *orgtests.AccessControlServiceStub
		config        *models.Config
		assert        func(t *testing.T, plugin *OrganizationsPlugin, serviceRegistry models.ServiceRegistry, logger *internaltests.LevelRecordingLogger, err error)
	}{
		{
			name:          "wires every dependency and registers the lookup service",
			includeUser:   true,
			includeMailer: true,
			accessControl: wiredAccessControl,
			assert: func(t *testing.T, plugin *OrganizationsPlugin, serviceRegistry models.ServiceRegistry, logger *internaltests.LevelRecordingLogger, err error) {
				t.Helper()
				require.NoError(t, err)

				require.NotNil(t, plugin.Api)
				require.NotNil(t, plugin.useCases)
				require.NotNil(t, plugin.hooksExecutor)
				require.NotNil(t, plugin.emailTemplateManager)
				require.NotNil(t, plugin.organizationService)
				require.NotNil(t, plugin.invitationService)
				require.NotNil(t, plugin.memberService)
				require.NotNil(t, plugin.teamService)
				require.NotNil(t, plugin.teamMemberService)
				require.Equal(t, wiredAccessControl, plugin.accessControlService)

				lookup, ok := serviceRegistry.Get(models.ServiceOrganization.String()).(OrganizationLookupService)
				require.True(t, ok)
				exists, lookupErr := lookup.ExistsByID(context.Background(), "missing-org")
				require.NoError(t, lookupErr)
				require.False(t, exists)

				require.Empty(t, logger.Records())
			},
		},
		{
			name:          "missing mailer only logs a warning",
			includeUser:   true,
			accessControl: orgtests.NewAccessControlServiceStub(),
			assert: func(t *testing.T, plugin *OrganizationsPlugin, serviceRegistry models.ServiceRegistry, logger *internaltests.LevelRecordingLogger, err error) {
				t.Helper()
				require.NoError(t, err)
				require.NotNil(t, plugin.Api)

				records := logger.Records()
				require.Len(t, records, 1)
				require.Equal(t, "warn", records[0].Level)
				require.Contains(t, records[0].Message, "mailer service not available")

				_, registered := serviceRegistry.Get(models.ServiceOrganization.String()).(OrganizationLookupService)
				require.True(t, registered)
			},
		},
		{
			name:          "fails without a user service",
			accessControl: orgtests.NewAccessControlServiceStub(),
			assert: func(t *testing.T, _ *OrganizationsPlugin, _ models.ServiceRegistry, _ *internaltests.LevelRecordingLogger, err error) {
				t.Helper()
				require.EqualError(t, err, "user service not available in service registry")
			},
		},
		{
			name:        "fails without an access control service",
			includeUser: true,
			assert: func(t *testing.T, _ *OrganizationsPlugin, _ models.ServiceRegistry, _ *internaltests.LevelRecordingLogger, err error) {
				t.Helper()
				require.EqualError(t, err, "access control service not available in service registry")
			},
		},
		{
			name:          "fails when permissions cannot be ensured",
			includeUser:   true,
			accessControl: failingAccessControl,
			assert: func(t *testing.T, _ *OrganizationsPlugin, _ models.ServiceRegistry, _ *internaltests.LevelRecordingLogger, err error) {
				t.Helper()
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.ErrorContains(t, err, "failed to ensure organization permissions")
			},
		},
		{
			name:          "loads a pre-parsed plugin config",
			includeUser:   true,
			includeMailer: true,
			accessControl: orgtests.NewAccessControlServiceStub(),
			config: &models.Config{
				BaseURL:  "https://example.com",
				BasePath: "/auth",
				Plugins:  models.PluginsConfig{},
				PreParsedConfigs: map[string]any{
					models.PluginOrganizations.String(): types.OrganizationsPluginConfig{
						Enabled:             true,
						MembersLimit:        new(11),
						MaxPageLimit:        new(33),
						InvitationExpiresIn: time.Hour,
					},
				},
			},
			assert: func(t *testing.T, plugin *OrganizationsPlugin, _ models.ServiceRegistry, _ *internaltests.LevelRecordingLogger, err error) {
				t.Helper()
				require.NoError(t, err)
				require.True(t, plugin.pluginConfig.Enabled)
				require.Equal(t, 11, *plugin.pluginConfig.MembersLimit)
				require.Equal(t, 33, *plugin.pluginConfig.MaxPageLimit)
				require.Equal(t, time.Hour, plugin.pluginConfig.InvitationExpiresIn)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, serviceRegistry, logger := newPluginInitContext(t, tt.includeUser, tt.includeMailer, tt.accessControl, tt.config)
			plugin := New(types.OrganizationsPluginConfig{})

			err := plugin.Init(ctx)

			tt.assert(t, plugin, serviceRegistry, logger, err)
		})
	}
}

func TestBuildAPI_WiresServices(t *testing.T) {
	t.Parallel()

	plugin := New(types.OrganizationsPluginConfig{})
	ctx, _, _ := newPluginInitContext(t, true, true, orgtests.NewAccessControlServiceStub(), nil)
	require.NoError(t, plugin.Init(ctx))

	api := BuildAPI(plugin)
	require.NotNil(t, api)
	require.Equal(t, plugin.organizationService, api.organizationService)
	require.Equal(t, plugin.invitationService, api.invitationService)
	require.Equal(t, plugin.memberService, api.memberService)
	require.Equal(t, plugin.teamService, api.teamService)
	require.Equal(t, plugin.teamMemberService, api.teamMemberService)
	require.Equal(t, plugin.memberRepo, api.memberRepo)
	require.Equal(t, plugin.accessControlService, api.accessControlService)
}

func TestNewOrganizationEmailTemplateManager(t *testing.T) {
	t.Parallel()

	manager, err := newOrganizationEmailTemplateManager()
	require.NoError(t, err)
	require.NotNil(t, manager)

	_, _, _, err = manager.Render("organization_invitation", types.OrganizationInvitationContext{})
	require.NoError(t, err)
}
