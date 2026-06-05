package bearer

import (
	"testing"

	"github.com/stretchr/testify/require"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	bearertests "github.com/Authula/authula/plugins/bearer/tests"
)

func TestBearerPlugin_Metadata(t *testing.T) {
	t.Parallel()

	plugin := New(BearerPluginConfig{})
	metadata := plugin.Metadata()

	require.NotEmpty(t, metadata.ID)
	require.NotEmpty(t, metadata.Version)
	require.NotEmpty(t, metadata.Description)
}

func TestBearerPlugin_Config(t *testing.T) {
	t.Parallel()

	cfg := BearerPluginConfig{HeaderName: "Custom-Auth", Enabled: true}
	plugin := New(cfg)

	returnedCfg := plugin.Config()
	require.Equal(t, cfg, returnedCfg)
}

func TestBearerPlugin_Init(t *testing.T) {
	t.Parallel()

	t.Run("missing_jwt_service", func(t *testing.T) {
		t.Parallel()
		reg := &internaltests.MockServiceRegistry{}
		reg.On("Get", models.ServiceJWT.String()).Return(nil).Once()

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig:       func() *models.Config { return &models.Config{} },
		})
		require.Error(t, err)
		reg.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mockSvc := &bearertests.MockJWTService{}
		reg := &internaltests.MockServiceRegistry{}
		reg.On("Get", models.ServiceJWT.String()).Return(mockSvc).Once()

		plugin := New(BearerPluginConfig{})
		err := plugin.Init(&models.PluginContext{
			Logger:          &internaltests.MockLogger{},
			ServiceRegistry: reg,
			GetConfig:       func() *models.Config { return &models.Config{} },
		})
		require.NoError(t, err)
		require.Equal(t, mockSvc, plugin.jwtService)
		reg.AssertExpectations(t)
	})
}
