package configloader

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"

	authulaconfig "github.com/Authula/authula/config"
	authulamodels "github.com/Authula/authula/models"
)

// Load reads a TOML config file and returns the normalized runtime config.
// The exists flag indicates whether the file was present on disk.
func Load(path string) (*authulamodels.Config, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return authulaconfig.NewConfig(), false, nil
		}
		return nil, false, fmt.Errorf("read config file: %w", err)
	}

	var loaded authulamodels.Config
	if err := toml.Unmarshal(data, &loaded); err != nil {
		return nil, true, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return authulaconfig.NewConfig(
		authulaconfig.WithAppName(loaded.AppName),
		authulaconfig.WithBaseURL(loaded.BaseURL),
		authulaconfig.WithBasePath(loaded.BasePath),
		authulaconfig.WithDatabase(loaded.Database),
		authulaconfig.WithLogger(loaded.Logger),
		authulaconfig.WithSecret(loaded.Secret),
		authulaconfig.WithSession(loaded.Session),
		authulaconfig.WithVerification(loaded.Verification),
		authulaconfig.WithSecurity(loaded.Security),
		authulaconfig.WithEventBus(loaded.EventBus),
		authulaconfig.WithPlugins(loaded.Plugins),
		authulaconfig.WithRouteMappings(loaded.RouteMappings),
		authulaconfig.WithDisabledPaths(loaded.DisabledPaths),
	), true, nil
}
