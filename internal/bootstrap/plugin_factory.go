package bootstrap

import (
	"fmt"

	"github.com/GoBetterAuth/go-better-auth/v2/internal/util"
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	bearerplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/bearer"
	configmanagerplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager"
	configmanagerplugintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/config-manager/types"
	csrfplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/csrf"
	emailplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/email"
	emailpasswordplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password"
	emailpasswordplugintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/email-password/types"
	emailplugintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/email/types"
	jwtplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt"
	jwtplugintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/jwt/types"
	oauth2plugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2"
	oauth2plugintypes "github.com/GoBetterAuth/go-better-auth/v2/plugins/oauth2/types"
	ratelimitplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/rate-limit"
	secondarystorageplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/secondary-storage"
	sessionplugin "github.com/GoBetterAuth/go-better-auth/v2/plugins/session"
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
			config := ratelimitplugin.RateLimitPluginConfig{}
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, fmt.Errorf("failed to parse ratelimit plugin config: %w", err)
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return ratelimitplugin.New(typedConfig.(ratelimitplugin.RateLimitPluginConfig))
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
