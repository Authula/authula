package secondarystorage

import (
	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/GoBetterAuth/go-better-auth/v2/services"
)

// SecondaryStorageServiceImpl implements the SecondaryStorageService interface
type SecondaryStorageServiceImpl struct {
	storage      models.SecondaryStorage
	providerName string
}

// NewSecondaryStorageService creates a new SecondaryStorageService instance
func NewSecondaryStorageService(providerName string, storage models.SecondaryStorage) services.SecondaryStorageService {
	return &SecondaryStorageServiceImpl{
		storage:      storage,
		providerName: providerName,
	}
}

// GetStorage returns the configured SecondaryStorage backend
func (s *SecondaryStorageServiceImpl) GetStorage() models.SecondaryStorage {
	return s.storage
}

// GetProviderName returns the name of the currently active provider
func (s *SecondaryStorageServiceImpl) GetProviderName() string {
	return s.providerName
}
