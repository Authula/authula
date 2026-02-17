package configloader

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"

	gobetterauthconfig "github.com/GoBetterAuth/go-better-auth/v2/config"
	gobetterauthmodels "github.com/GoBetterAuth/go-better-auth/v2/models"
)

// Load reads a TOML config file and returns the normalized runtime config.
// The exists flag indicates whether the file was present on disk.
func Load(path string) (*gobetterauthmodels.Config, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return gobetterauthconfig.NewConfig(), false, nil
		}
		return nil, false, fmt.Errorf("read config file: %w", err)
	}

	var loaded gobetterauthmodels.Config
	if err := toml.Unmarshal(data, &loaded); err != nil {
		return nil, true, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return gobetterauthconfig.NewConfig(
		gobetterauthconfig.WithAppName(loaded.AppName),
		gobetterauthconfig.WithBaseURL(loaded.BaseURL),
		gobetterauthconfig.WithBasePath(loaded.BasePath),
		gobetterauthconfig.WithDatabase(loaded.Database),
		gobetterauthconfig.WithLogger(loaded.Logger),
		gobetterauthconfig.WithSecret(loaded.Secret),
		gobetterauthconfig.WithSession(loaded.Session),
		gobetterauthconfig.WithSecurity(loaded.Security),
		gobetterauthconfig.WithEventBus(loaded.EventBus),
		gobetterauthconfig.WithPlugins(loaded.Plugins),
		gobetterauthconfig.WithRouteMappings(loaded.RouteMappings),
	), true, nil
}
