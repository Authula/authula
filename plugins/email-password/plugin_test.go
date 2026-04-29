package email_password

import (
	"testing"

	"github.com/Authula/authula/internal/plugins"
	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/stretchr/testify/require"
)

type pluginInitTestCase struct {
	name                 string
	includeUserService   bool
	includeMailerService bool
	wantErr              string
	wantMailerService    bool
	wantAPI              bool
}

func TestEmailPasswordPlugin_Init(t *testing.T) {
	t.Parallel()

	tests := []pluginInitTestCase{
		{
			name:                 "allows missing mailer service",
			includeUserService:   true,
			includeMailerService: false,
			wantMailerService:    false,
			wantAPI:              true,
		},
		{
			name:                 "fails when user service is missing",
			includeUserService:   false,
			includeMailerService: false,
			wantErr:              "user service not available in service registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(types.EmailPasswordPluginConfig{})
			serviceRegistry := plugins.NewServiceRegistry()

			if tt.includeUserService {
				serviceRegistry.Register(models.ServiceUser.String(), &inttests.MockUserService{})
			}
			serviceRegistry.Register(models.ServiceAccount.String(), &inttests.MockAccountService{})
			serviceRegistry.Register(models.ServiceSession.String(), &inttests.MockSessionService{})
			serviceRegistry.Register(models.ServiceVerification.String(), &inttests.MockVerificationService{})
			serviceRegistry.Register(models.ServiceToken.String(), &inttests.MockTokenService{})
			serviceRegistry.Register(models.ServicePassword.String(), &inttests.MockPasswordService{})
			if tt.includeMailerService {
				serviceRegistry.Register(models.ServiceMailer.String(), &inttests.MockMailerService{})
			}

			ctx := &models.PluginContext{
				Logger:          &inttests.MockLogger{},
				ServiceRegistry: serviceRegistry,
				GetConfig: func() *models.Config {
					return &models.Config{BaseURL: "http://localhost", BasePath: "/auth"}
				},
			}

			err := plugin.Init(ctx)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantMailerService, plugin.mailerService != nil)
			require.Equal(t, tt.wantAPI, plugin.Api != nil)
			if tt.includeMailerService {
				require.NotNil(t, plugin.mailerService)
			}
			if !tt.includeMailerService {
				require.Nil(t, plugin.mailerService)
			}
		})
	}
}
