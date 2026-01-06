package secondarystorage

import "github.com/GoBetterAuth/go-better-auth/models"

// SecondaryStorageAPI is the API exposed by the secondary storage plugin
type SecondaryStorageAPI interface {
	// GetStorage returns the configured SecondaryStorage backend
	GetStorage() models.SecondaryStorage
	// GetProviderName returns the name of the currently active provider
	GetProviderName() string
}
