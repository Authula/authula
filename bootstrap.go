package gobetterauth

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/uptrace/bun"

	"github.com/GoBetterAuth/go-better-auth/events"
	internalbootstrap "github.com/GoBetterAuth/go-better-auth/internal/bootstrap"
	internalevents "github.com/GoBetterAuth/go-better-auth/internal/events"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// InitLogger initializes the logger based on configuration
func InitLogger(config *models.Config) models.Logger {
	return internalbootstrap.InitLogger(internalbootstrap.LoggerOptions{Level: config.Logger.Level})
}

// InitDatabase creates a Bun DB connection based on provider
func InitDatabase(config *models.Config, logger models.Logger, logLevel string) (bun.IDB, error) {
	return internalbootstrap.InitDatabase(
		internalbootstrap.DatabaseOptions{
			Provider:        config.Database.Provider,
			URL:             config.Database.URL,
			MaxOpenConns:    config.Database.MaxOpenConns,
			MaxIdleConns:    config.Database.MaxIdleConns,
			ConnMaxLifetime: config.Database.ConnMaxLifetime,
		},
		logger,
		logLevel,
	)
}

// InitEventBus creates an event bus based on the configuration
func InitEventBus(config *models.Config) (models.EventBus, error) {
	// Default to gochannel if not specified
	provider := config.EventBus.Provider
	if provider == "" {
		provider = events.ProviderGoChannel.String()
	}

	eventBusConfig := config.EventBus
	if provider == events.ProviderGoChannel.String() && eventBusConfig.GoChannel == nil {
		eventBusConfig.GoChannel = &models.GoChannelConfig{
			BufferSize: 100,
		}
	}

	logger := watermill.NewStdLogger(false, false)

	pubsub, err := internalevents.InitWatermillProvider(&eventBusConfig, logger)
	if err != nil {
		return nil, err
	}

	return internalevents.NewEventBus(config, logger, pubsub), nil
}
