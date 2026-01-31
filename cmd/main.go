package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"

	gobetterauth "github.com/GoBetterAuth/go-better-auth/v2"
	gobetterauthconfig "github.com/GoBetterAuth/go-better-auth/v2/config"
	gobetterauthenv "github.com/GoBetterAuth/go-better-auth/v2/env"
	"github.com/GoBetterAuth/go-better-auth/v2/internal/bootstrap"
	gobetterauthmodels "github.com/GoBetterAuth/go-better-auth/v2/models"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Run GoBetterAuth with plugins built from config file
// This demonstrates the unified architecture where both library and standalone modes
// use identical runtime behavior - they only differ in how plugins are instantiated
func main() {
	_ = godotenv.Load(".env")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	config := loadConfig()

	pluginsList := bootstrap.BuildPluginsFromConfig(config)

	auth := gobetterauth.New(&gobetterauth.AuthConfig{
		Config:  config,
		Plugins: pluginsList,
	})

	// Channel to signal restart
	restartChan := make(chan struct{})
	// Channel to signal shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Server loop with restart capability
	for {
		runServer(logger, auth, restartChan, shutdownChan)
	}
}

// runServer starts the HTTP server and handles restarts
func runServer(logger gobetterauthmodels.Logger, auth *gobetterauth.Auth, restartChan chan struct{}, shutdownChan chan os.Signal) {
	port := getEnv(gobetterauthenv.EnvPort, "8080")

	// Create HTTP server with graceful shutdown support
	server := &http.Server{
		Addr:    ":" + port,
		Handler: auth.Handler(),
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
			panic(fmt.Errorf("server error: %w", err))
		}
		return

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
		return

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
}

// loadConfig loads configuration with proper precedence:
func loadConfig() *gobetterauthmodels.Config {
	configPath := getEnv(gobetterauthenv.EnvConfigPath, "config.toml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		slog.Debug("No config file found, continuing", "path", configPath, "error", err)
		return gobetterauthconfig.NewConfig()
	}

	var loadedConfig gobetterauthmodels.Config
	if err := toml.Unmarshal(data, &loadedConfig); err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %w", err))
	}

	return gobetterauthconfig.NewConfig(
		gobetterauthconfig.WithAppName(loadedConfig.AppName),
		gobetterauthconfig.WithBaseURL(loadedConfig.BaseURL),
		gobetterauthconfig.WithBasePath(loadedConfig.BasePath),
		gobetterauthconfig.WithDatabase(loadedConfig.Database),
		gobetterauthconfig.WithLogger(loadedConfig.Logger),
		gobetterauthconfig.WithSecret(loadedConfig.Secret),
		gobetterauthconfig.WithSession(loadedConfig.Session),
		gobetterauthconfig.WithSecurity(loadedConfig.Security),
		gobetterauthconfig.WithEventBus(loadedConfig.EventBus),
		gobetterauthconfig.WithPlugins(loadedConfig.Plugins),
		gobetterauthconfig.WithRouteMappings(loadedConfig.RouteMappings),
	)
}
