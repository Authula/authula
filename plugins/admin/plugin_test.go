package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalplugins "github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	admintests "github.com/Authula/authula/plugins/admin/tests"
	"github.com/Authula/authula/plugins/admin/types"
)

func TestAdminPlugin_Metadata(t *testing.T) {
	t.Parallel()

	plugin := New(types.AdminPluginConfig{})
	metadata := plugin.Metadata()

	assert.Equal(t, models.PluginAdmin.String(), metadata.ID)
	assert.NotEmpty(t, metadata.Version)
	assert.NotEmpty(t, metadata.Description)
}

func TestAdminPlugin_ConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   types.AdminPluginConfig
		wantMax  time.Duration
		wantName string
	}{
		{
			name:     "empty config uses default max expiry",
			config:   types.AdminPluginConfig{},
			wantMax:  15 * time.Minute,
			wantName: "admin",
		},
		{
			name:     "explicit max expiry is preserved",
			config:   types.AdminPluginConfig{Enabled: true, ImpersonationMaxExpiresIn: time.Hour},
			wantMax:  time.Hour,
			wantName: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tt.config)
			cfg, ok := plugin.Config().(types.AdminPluginConfig)
			require.True(t, ok)
			assert.Equal(t, tt.wantMax, cfg.ImpersonationMaxExpiresIn)
			assert.Equal(t, tt.wantName, plugin.Metadata().ID)
		})
	}
}

func TestAdminPlugin_DependsOnAndClose(t *testing.T) {
	t.Parallel()

	plugin := New(types.AdminPluginConfig{})
	assert.Equal(t, []string{models.PluginAccessControl.String()}, plugin.DependsOn())
	assert.NoError(t, plugin.Close())
}

func TestAdminPlugin_Migrations(t *testing.T) {
	t.Parallel()

	plugin := New(types.AdminPluginConfig{})

	tests := []struct {
		provider string
		wantLen  int
	}{
		{provider: "sqlite", wantLen: 1},
		{provider: "postgres", wantLen: 1},
		{provider: "mysql", wantLen: 1},
		{provider: "unknown", wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Parallel()

			migrations := plugin.Migrations(tt.provider)
			require.Len(t, migrations, tt.wantLen)
			if tt.wantLen > 0 {
				assert.NotEmpty(t, migrations[0].Version)
				assert.NotNil(t, migrations[0].Up)
				assert.NotNil(t, migrations[0].Down)
			}
		})
	}
}

func TestAdminPlugin_Hooks(t *testing.T) {
	t.Parallel()

	plugin := New(types.AdminPluginConfig{})
	hooks := plugin.Hooks()

	require.Len(t, hooks, 3)
	wantOrders := []int{15, 17, 25}
	for i, hook := range hooks {
		assert.Equal(t, models.HookBefore, hook.Stage)
		assert.Equal(t, wantOrders[i], hook.Order)
		assert.NotNil(t, hook.Handler)
	}
}

func TestAdminPlugin_Init_Success(t *testing.T) {
	t.Parallel()

	db := internaltests.NewSQLiteIntegrationDB(t)
	registry := internalplugins.NewServiceRegistry()

	sessionSvc := &internaltests.MockSessionService{}
	tokenSvc := &internaltests.MockTokenService{}
	passwordSvc := &admintests.MockPasswordService{}
	accessControlSvc := admintests.NewAccessControlServiceFixture()
	accessControlSvc.On("EnsurePermissions", mock.Anything, mock.Anything).Return(nil).Once()

	registry.Register(models.ServiceSession.String(), sessionSvc)
	registry.Register(models.ServiceToken.String(), tokenSvc)
	registry.Register(models.ServicePassword.String(), passwordSvc)
	registry.Register(models.ServiceAccessControl.String(), accessControlSvc)

	plugin := New(types.AdminPluginConfig{})
	err := plugin.Init(&models.PluginContext{
		DB:              db,
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return &models.Config{} },
	})
	require.NoError(t, err)

	require.NotNil(t, plugin.Api)
	assert.Equal(t, plugin.Api, registry.Get(models.ServiceAdmin.String()))
	assert.Equal(t, sessionSvc, plugin.sessionService)
	assert.Equal(t, tokenSvc, plugin.tokenService)
	assert.Equal(t, accessControlSvc, plugin.accessControlService)

	accessControlSvc.AssertExpectations(t)
}

func TestAdminPlugin_Init_MissingRequiredService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		omit    string
		wantErr string
	}{
		{name: "session", omit: models.ServiceSession.String(), wantErr: models.ServiceSession.String()},
		{name: "token", omit: models.ServiceToken.String(), wantErr: models.ServiceToken.String()},
		{name: "password", omit: models.ServicePassword.String(), wantErr: models.ServicePassword.String()},
		{name: "access control", omit: models.ServiceAccessControl.String(), wantErr: models.ServiceAccessControl.String()},
	}

	services := []struct {
		name  string
		value any
	}{
		{name: models.ServiceSession.String(), value: &internaltests.MockSessionService{}},
		{name: models.ServiceToken.String(), value: &internaltests.MockTokenService{}},
		{name: models.ServicePassword.String(), value: &admintests.MockPasswordService{}},
		{name: models.ServiceAccessControl.String(), value: admintests.NewAccessControlServiceFixture()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := internalplugins.NewServiceRegistry()
			for _, svc := range services {
				if svc.name != tt.omit {
					registry.Register(svc.name, svc.value)
				}
			}

			plugin := New(types.AdminPluginConfig{})
			err := plugin.Init(&models.PluginContext{
				DB:              internaltests.NewSQLiteIntegrationDB(t),
				Logger:          &internaltests.MockLogger{},
				ServiceRegistry: registry,
				GetConfig:       func() *models.Config { return &models.Config{} },
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestAdminPlugin_Init_EnsurePermissionsError(t *testing.T) {
	t.Parallel()

	registry := internalplugins.NewServiceRegistry()
	accessControlSvc := admintests.NewAccessControlServiceFixture()
	accessControlSvc.On("EnsurePermissions", mock.Anything, mock.Anything).Return(assert.AnError).Once()

	registry.Register(models.ServiceSession.String(), &internaltests.MockSessionService{})
	registry.Register(models.ServiceToken.String(), &internaltests.MockTokenService{})
	registry.Register(models.ServicePassword.String(), &admintests.MockPasswordService{})
	registry.Register(models.ServiceAccessControl.String(), accessControlSvc)

	plugin := New(types.AdminPluginConfig{})
	err := plugin.Init(&models.PluginContext{
		DB:              internaltests.NewSQLiteIntegrationDB(t),
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return &models.Config{} },
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "failed to ensure admin permissions")

	accessControlSvc.AssertExpectations(t)
}
