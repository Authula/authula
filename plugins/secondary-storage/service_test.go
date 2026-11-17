package secondarystorage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSecondaryStorageService(t *testing.T) {
	t.Parallel()

	storage := newTestMemorySecondaryStorage()
	t.Cleanup(func() {
		assert.NoError(t, storage.Close())
	})

	tests := []struct {
		name         string
		providerName string
		storage      *MemorySecondaryStorage
	}{
		{name: "memory provider", providerName: "memory", storage: storage},
		{name: "nil storage is allowed", providerName: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewSecondaryStorageService(tt.providerName, tt.storage)

			assert.Equal(t, tt.providerName, service.GetProviderName())
			if tt.storage == nil {
				assert.Nil(t, service.GetStorage())
				return
			}
			assert.Same(t, tt.storage, service.GetStorage())
		})
	}
}
