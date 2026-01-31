package services

import "github.com/GoBetterAuth/go-better-auth/v2/models"

// SecondaryStorageService provides access to the configured secondary storage backend
type SecondaryStorageService interface {
	// GetStorage returns the configured SecondaryStorage backend
	GetStorage() models.SecondaryStorage
	// GetProviderName returns the name of the currently active provider (e.g., "redis", "database", "memory")
	GetProviderName() string
}
