package bootstrap

import (
	"fmt"

	"github.com/Authula/authula/internal/util"
	"github.com/Authula/authula/models"
	accesscontrolplugin "github.com/Authula/authula/plugins/access-control"
	accesscontrolplugintypes "github.com/Authula/authula/plugins/access-control/types"
	adminplugin "github.com/Authula/authula/plugins/admin"
	adminplugintypes "github.com/Authula/authula/plugins/admin/types"
	apikeyplugin "github.com/Authula/authula/plugins/api-key"
	apikeyplugintypes "github.com/Authula/authula/plugins/api-key/types"
	bearerplugin "github.com/Authula/authula/plugins/bearer"
	configmanagerplugin "github.com/Authula/authula/plugins/config-manager"
	configmanagerplugintypes "github.com/Authula/authula/plugins/config-manager/types"
	csrfplugin "github.com/Authula/authula/plugins/csrf"
	emailplugin "github.com/Authula/authula/plugins/email"
	emailpasswordplugin "github.com/Authula/authula/plugins/email-password"
	emailpasswordplugintypes "github.com/Authula/authula/plugins/email-password/types"
	emailplugintypes "github.com/Authula/authula/plugins/email/types"
	jwtplugin "github.com/Authula/authula/plugins/jwt"
	jwtplugintypes "github.com/Authula/authula/plugins/jwt/types"
	magiclinkplugin "github.com/Authula/authula/plugins/magic-link"
	magiclinkplugintypes "github.com/Authula/authula/plugins/magic-link/types"
	oauth2plugin "github.com/Authula/authula/plugins/oauth2"
	oauth2plugintypes "github.com/Authula/authula/plugins/oauth2/types"
	organizationsplugin "github.com/Authula/authula/plugins/organizations"
	organizationsplugintypes "github.com/Authula/authula/plugins/organizations/types"
	ratelimitplugin "github.com/Authula/authula/plugins/rate-limit"
	ratelimitplugintypes "github.com/Authula/authula/plugins/rate-limit/types"
	secondarystorageplugin "github.com/Authula/authula/plugins/secondary-storage"
	sessionplugin "github.com/Authula/authula/plugins/session"
	totpplugin "github.com/Authula/authula/plugins/totp"
	totplugintypes "github.com/Authula/authula/plugins/totp/types"
)

// PluginFactory defines a factory function for creating a plugin instance from typed config data.
type PluginFactory struct {
	ID                string
	ConfigParser      func(rawConfig any) (any, error)    // Parses raw config to typed config
	Constructor       func(typedConfig any) models.Plugin // Creates plugin from typed config
	RequiredByDefault bool                                // Whether this plugin must be enabled
}

// pluginFactories is an ordered list of registered plugin factories.
var pluginFactories = []PluginFactory{
	{
		ID:                models.PluginAdmin.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := adminplugintypes.AdminPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse admin plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return adminplugin.New(typedConfig.(adminplugintypes.AdminPluginConfig))
		},
	},
	{
		ID:                models.PluginConfigManager.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := configmanagerplugintypes.ConfigManagerPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse config_manager plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return configmanagerplugin.New(typedConfig.(configmanagerplugintypes.ConfigManagerPluginConfig))
		},
	},
	{
		ID:                models.PluginSecondaryStorage.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := secondarystorageplugin.SecondaryStoragePluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse secondary_storage plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return secondarystorageplugin.New(typedConfig.(secondarystorageplugin.SecondaryStoragePluginConfig))
		},
	},
	{
		ID:                models.PluginEmail.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := emailplugintypes.EmailPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse email plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return emailplugin.New(typedConfig.(emailplugintypes.EmailPluginConfig))
		},
	},
	{
		ID:                models.PluginCSRF.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := csrfplugin.CSRFPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse csrf plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return csrfplugin.New(typedConfig.(csrfplugin.CSRFPluginConfig))
		},
	},
	{
		ID:                models.PluginEmailPassword.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := emailpasswordplugintypes.EmailPasswordPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse email_password plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return emailpasswordplugin.New(typedConfig.(emailpasswordplugintypes.EmailPasswordPluginConfig))
		},
	},
	{
		ID:                models.PluginOAuth2.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := oauth2plugintypes.OAuth2PluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse oauth2 plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return oauth2plugin.New(typedConfig.(oauth2plugintypes.OAuth2PluginConfig))
		},
	},
	{
		ID:                models.PluginSession.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := sessionplugin.SessionPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse session plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return sessionplugin.New(typedConfig.(sessionplugin.SessionPluginConfig))
		},
	},
	{
		ID:                models.PluginJWT.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := jwtplugintypes.JWTPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse jwt plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return jwtplugin.New(typedConfig.(jwtplugintypes.JWTPluginConfig))
		},
	},
	{
		ID:                models.PluginBearer.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := bearerplugin.BearerPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse bearer plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return bearerplugin.New(typedConfig.(bearerplugin.BearerPluginConfig))
		},
	},
	{
		ID:                models.PluginRateLimit.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := ratelimitplugintypes.RateLimitPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse ratelimit plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return ratelimitplugin.New(typedConfig.(ratelimitplugintypes.RateLimitPluginConfig))
		},
	},
	{
		ID:                models.PluginMagicLink.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := magiclinkplugintypes.MagicLinkPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse magic link plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return magiclinkplugin.New(typedConfig.(magiclinkplugintypes.MagicLinkPluginConfig))
		},
	},
	{
		ID:                models.PluginAccessControl.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := accesscontrolplugintypes.AccessControlPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse access control plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return accesscontrolplugin.New(typedConfig.(accesscontrolplugintypes.AccessControlPluginConfig))
		},
	},
	{
		ID:                models.PluginOrganizations.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := organizationsplugintypes.OrganizationsPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse organizations plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return organizationsplugin.New(typedConfig.(organizationsplugintypes.OrganizationsPluginConfig))
		},
	},
	{
		ID:                models.PluginTOTP.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := totplugintypes.TOTPPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse totp plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return totpplugin.New(typedConfig.(totplugintypes.TOTPPluginConfig))
		},
	},
	{
		ID:                models.PluginApiKey.String(),
		RequiredByDefault: false,
		ConfigParser: func(rawConfig any) (any, error) {
			config := apikeyplugintypes.ApiKeyPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse api_key plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return apikeyplugin.New(typedConfig.(apikeyplugintypes.ApiKeyPluginConfig))
		},
	},
}

// isPluginEnabled checks if a plugin is enabled in the raw config
func isPluginEnabled(rawConfig any, isRequiredByDefault bool) bool {
	if pluginConfig, ok := rawConfig.(map[string]any); ok {
		if enabled, ok := pluginConfig["enabled"].(bool); ok {
			return enabled
		}
	}
	// If config doesn't explicitly set enabled:
	// - Required plugins default to true (enabled)
	// - Optional plugins default to true if config is present, false if nil
	if rawConfig == nil {
		return isRequiredByDefault
	}
	return true
}

// BuildPluginsFromConfig builds an ordered list of enabled plugins from the configuration.
// It validates that all plugins in the configuration are registered and instantiates them
// in the order defined by pluginFactories.
func BuildPluginsFromConfig(config *models.Config) []models.Plugin {
	// Validate that all plugins in config exist in the registry
	registeredIDs := make(map[string]bool)
	for _, factory := range pluginFactories {
		registeredIDs[factory.ID] = true
	}

	for pluginID := range config.Plugins {
		if !registeredIDs[pluginID] {
			panic(fmt.Errorf("unknown plugin in configuration: %q", pluginID))
		}
	}

	// Instantiate plugins in the registered order
	var plugins []models.Plugin
	for _, factory := range pluginFactories {
		rawConfig := config.Plugins[factory.ID]
		enabled := isPluginEnabled(rawConfig, factory.RequiredByDefault)

		if factory.RequiredByDefault && !enabled {
			panic(fmt.Errorf("%s is required but not enabled", factory.ID))
		}

		if !enabled {
			continue
		}

		// Parse the raw config to typed config
		typedConfig, err := factory.ConfigParser(rawConfig)
		if err != nil {
			panic(fmt.Errorf("failed to parse plugin %s config: %w", factory.ID, err))
		}

		// Create the plugin with typed config
		plugin := factory.Constructor(typedConfig)

		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}
