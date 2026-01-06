package secondarystorage

import "github.com/GoBetterAuth/go-better-auth/models"

// secondaryStorageAPI is the concrete implementation of SecondaryStorageAPI
type secondaryStorageAPI struct {
	storage      models.SecondaryStorage
	providerName string
}

// GetStorage returns the configured SecondaryStorage backend
func (a *secondaryStorageAPI) GetStorage() models.SecondaryStorage {
	return a.storage
}

// GetProviderName returns the name of the currently active provider
func (a *secondaryStorageAPI) GetProviderName() string {
	return a.providerName
}
