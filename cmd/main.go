package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	gobetterauth "github.com/GoBetterAuth/go-better-auth"
	gobetterauthconfig "github.com/GoBetterAuth/go-better-auth/config"
	gobetterauthenv "github.com/GoBetterAuth/go-better-auth/env"
	gobetterauthmodels "github.com/GoBetterAuth/go-better-auth/models"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Run GoBetterAuth in standalone mode
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Channel to signal restart
	restartChan := make(chan struct{})
	// Channel to signal shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Server loop with restart capability
	for {
		if err := runServer(logger, restartChan, shutdownChan); err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}
}

// loadConfig loads configuration with proper precedence:
func loadConfig() (*gobetterauthmodels.Config, error) {
	_ = godotenv.Load(".env")

	v := viper.New()
	configPath := getEnv(gobetterauthenv.ConfigPath, "config.toml")
	v.SetConfigFile(configPath)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		slog.Debug("No config.toml found, continuing", "path", configPath)
	}

	var loadedConfig gobetterauthmodels.Config
	if err := v.Unmarshal(&loadedConfig, func(c *mapstructure.DecoderConfig) {
		c.TagName = "toml"
		c.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)
	}); err != nil {
		return &loadedConfig, err
	}

	authConfig := gobetterauthconfig.NewConfig(
		gobetterauthconfig.WithMode(gobetterauthmodels.ModeStandalone),
		gobetterauthconfig.WithAppName(loadedConfig.AppName),
		gobetterauthconfig.WithBaseURL(loadedConfig.BaseURL),
		gobetterauthconfig.WithBasePath(loadedConfig.BasePath),
		gobetterauthconfig.WithSecret(loadedConfig.Secret),
		gobetterauthconfig.WithLogger(loadedConfig.Logger),
		gobetterauthconfig.WithDatabase(loadedConfig.Database),
		gobetterauthconfig.WithEmailConfig(loadedConfig.Email),
		gobetterauthconfig.WithSecondaryStorage(loadedConfig.SecondaryStorage),
		gobetterauthconfig.WithEmailPassword(loadedConfig.EmailPassword),
		gobetterauthconfig.WithEmailVerification(loadedConfig.EmailVerification),
		gobetterauthconfig.WithUser(loadedConfig.User),
		gobetterauthconfig.WithSession(loadedConfig.Session),
		gobetterauthconfig.WithCSRF(loadedConfig.CSRF),
		gobetterauthconfig.WithSocialProviders(loadedConfig.SocialProviders),
		gobetterauthconfig.WithTrustedOrigins(loadedConfig.TrustedOrigins),
		gobetterauthconfig.WithRateLimit(loadedConfig.RateLimit),
		gobetterauthconfig.WithEventBus(loadedConfig.EventBus),
		gobetterauthconfig.WithEndpointHooks(loadedConfig.EndpointHooks),
		gobetterauthconfig.WithDatabaseHooks(loadedConfig.DatabaseHooks),
		gobetterauthconfig.WithEventHooks(loadedConfig.EventHooks),
		gobetterauthconfig.WithWebhooks(loadedConfig.Webhooks),
	)

	return authConfig, nil
}

// runServer starts the HTTP server and handles restarts
func runServer(logger gobetterauthmodels.Logger, restartChan chan struct{}, shutdownChan chan os.Signal) error {
	port := getEnv("PORT", "8080")

	config, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		return err
	}

	auth := gobetterauth.New(config)

	// Set the restart handler - called when config changes require restart
	var mu sync.Mutex
	restartRequested := false
	auth.OnRestartRequired = func() error {
		mu.Lock()
		defer mu.Unlock()
		if restartRequested {
			return nil // Already requested
		}
		restartRequested = true
		logger.Info("Restart handler triggered - gracefully shutting down server")
		// Send restart signal in a goroutine to avoid deadlock
		go func() {
			restartChan <- struct{}{}
		}()
		return nil
	}

	// Create HTTP server with graceful shutdown support
	server := &http.Server{
		Addr: ":" + port,
		Handler: auth.CorsAuthMiddleware()(
			auth.OptionalAuthMiddleware()(
				auth.Handler(),
			),
		),
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Starting GoBetterAuth standalone server", "port", port)
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for shutdown, restart, or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			return err
		}
		return nil

	case <-restartChan:
		logger.Info("Restarting server due to configuration change")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error", "error", err)
		}
		if err := auth.ClosePlugins(); err != nil {
			logger.Error("Failed to close plugins", "error", err)
		}
		return nil

	case sig := <-shutdownChan:
		logger.Info("Shutdown signal received", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error", "error", err)
		}
		if err := auth.ClosePlugins(); err != nil {
			logger.Error("Failed to close plugins", "error", err)
		}
		os.Exit(0)
	}

	return nil
}
