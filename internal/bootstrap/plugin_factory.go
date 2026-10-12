package bootstrap

import (
	"fmt"

	"github.com/Authula/authula/models"
	accesscontrolplugin "github.com/Authula/authula/plugins/access-control"
	adminplugin "github.com/Authula/authula/plugins/admin"
	apikeyplugin "github.com/Authula/authula/plugins/api-key"
	bearerplugin "github.com/Authula/authula/plugins/bearer"
	csrfplugin "github.com/Authula/authula/plugins/csrf"
	emailplugin "github.com/Authula/authula/plugins/email"
	emailpasswordplugin "github.com/Authula/authula/plugins/email-password"
	jwtplugin "github.com/Authula/authula/plugins/jwt"
	magiclinkplugin "github.com/Authula/authula/plugins/magic-link"
	oauth2plugin "github.com/Authula/authula/plugins/oauth2"
	organizationsplugin "github.com/Authula/authula/plugins/organizations"
	ratelimitplugin "github.com/Authula/authula/plugins/rate-limit"
	secondarystorageplugin "github.com/Authula/authula/plugins/secondary-storage"
	sessionplugin "github.com/Authula/authula/plugins/session"
	totpplugin "github.com/Authula/authula/plugins/totp"
	"github.com/Authula/authula/util"
)

// pluginFactory defines how to parse a plugin's raw config into its typed config
// and how to construct the plugin from that typed config.
type pluginFactory struct {
	ID                string
	ConfigParser      func(rawConfig any) (any, error)    // Parses raw config to typed config
	Constructor       func(typedConfig any) models.Plugin // Creates plugin from typed config
	RequiredByDefault bool                                // Whether this plugin must be enabled
}

// newPluginFactory builds a pluginFactory for a plugin whose config type is T and
// whose constructor returns the concrete plugin type P. Go infers T and P from the
// constructor that is passed in, so no type assertions or config parsing are repeated
// at the registration site.
func newPluginFactory[T any, P models.Plugin](id string, requiredByDefault bool, constructor func(T) P) pluginFactory {
	return pluginFactory{
		ID:                id,
		RequiredByDefault: requiredByDefault,
		ConfigParser: func(rawConfig any) (any, error) {
			var config T
			if rawConfig != nil {
				if err := util.ParsePluginConfig(rawConfig, &config); err != nil {
					return nil, err
				}
			}
			return config, nil
		},
		Constructor: func(typedConfig any) models.Plugin {
			return constructor(typedConfig.(T))
		},
	}
}

// pluginFactories is an ordered list of registered plugin factories.
var pluginFactories = []pluginFactory{
	newPluginFactory(models.PluginSecondaryStorage.String(), false, secondarystorageplugin.New),
	newPluginFactory(models.PluginAccessControl.String(), false, accesscontrolplugin.New),
	newPluginFactory(models.PluginAdmin.String(), false, adminplugin.New),
	newPluginFactory(models.PluginEmail.String(), false, emailplugin.New),
	newPluginFactory(models.PluginCSRF.String(), false, csrfplugin.New),
	newPluginFactory(models.PluginEmailPassword.String(), false, emailpasswordplugin.New),
	newPluginFactory(models.PluginOAuth2.String(), false, oauth2plugin.New),
	newPluginFactory(models.PluginSession.String(), false, sessionplugin.New),
	newPluginFactory(models.PluginJWT.String(), false, jwtplugin.New),
	newPluginFactory(models.PluginBearer.String(), false, bearerplugin.New),
	newPluginFactory(models.PluginRateLimit.String(), false, ratelimitplugin.New),
	newPluginFactory(models.PluginMagicLink.String(), false, magiclinkplugin.New),
	newPluginFactory(models.PluginOrganizations.String(), false, organizationsplugin.New),
	newPluginFactory(models.PluginTOTP.String(), false, totpplugin.New),
	newPluginFactory(models.PluginApiKey.String(), false, apikeyplugin.New),
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
