package constants

import "errors"

var (
	ErrConfigManagerNotAvailable = errors.New("config manager not available")
	ErrConfigNotAvailable        = errors.New("configuration not available")
	ErrPluginConfigNotFound      = errors.New("plugin configuration not found")
	ErrPluginAlreadyRegistered   = errors.New("plugin already registered as config watcher")
)
