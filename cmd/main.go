package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"

	gobetterauth "github.com/GoBetterAuth/go-better-auth"
	"github.com/GoBetterAuth/go-better-auth/config"
	"github.com/GoBetterAuth/go-better-auth/models"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Run GoBetterAuth in standalone mode
func main() {
	err := godotenv.Load()
	if err != nil {
		env := os.Getenv("GO_ENV")
		if env != "production" {
			// Production-specific logic
			log.Fatal("No .env file found", "error", err)
			return
		}
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Channel to signal restart
	restartChan := make(chan struct{})
	// Channel to signal shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Server loop with restart capability
	for {
		if err := runServer(port, restartChan, shutdownChan); err != nil {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}
}

// runServer starts the HTTP server and handles restarts
func runServer(port string, restartChan chan struct{}, shutdownChan chan os.Signal) error {
	logger := slog.Default()

	// Load configuration from TOML file if available
	tomlConfig := loadConfigFromFile()

	// Apply environment variable overrides and defaults
	applyConfigDefaults(&tomlConfig)

	// Build config using functional options pattern to ensure all fields are set
	authConfig := config.NewConfig(
		config.WithMode(models.ModeStandalone),
		config.WithAppName(tomlConfig.AppName),
		config.WithBaseURL(tomlConfig.BaseURL),
		config.WithBasePath(tomlConfig.BasePath),
		config.WithSecret(tomlConfig.Secret),
		config.WithLogger(tomlConfig.Logger),
		config.WithDatabase(tomlConfig.Database),
		config.WithEmailConfig(tomlConfig.Email),
		config.WithSecondaryStorage(tomlConfig.SecondaryStorage),
		config.WithEmailPassword(tomlConfig.EmailPassword),
		config.WithEmailVerification(tomlConfig.EmailVerification),
		config.WithUser(tomlConfig.User),
		config.WithSession(tomlConfig.Session),
		config.WithCSRF(tomlConfig.CSRF),
		config.WithSocialProviders(tomlConfig.SocialProviders),
		config.WithTrustedOrigins(tomlConfig.TrustedOrigins),
		config.WithRateLimit(tomlConfig.RateLimit),
		config.WithEventBus(tomlConfig.EventBus),
		config.WithEndpointHooks(tomlConfig.EndpointHooks),
		config.WithDatabaseHooks(tomlConfig.DatabaseHooks),
		config.WithEventHooks(tomlConfig.EventHooks),
		config.WithWebhooks(tomlConfig.Webhooks),
	)

	auth := gobetterauth.New(authConfig)

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

// loadConfigFromFile attempts to load configuration from TOML file if it exists
func loadConfigFromFile() models.Config {
	configPath := getEnv("GO_BETTER_AUTH_CONFIG_PATH", "config.toml")
	var config models.Config

	if _, err := os.Stat(configPath); err != nil {
		// File doesn't exist, return empty config - will use env vars and defaults
		return config
	}

	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		slog.Warn("Failed to parse TOML config file, will use environment variables and defaults", "path", configPath, "error", err)
	}

	return config
}

// applyConfigDefaults applies environment variable overrides and sensible defaults
func applyConfigDefaults(config *models.Config) {
	// Override other critical settings from environment variables
	if baseURL := os.Getenv("GO_BETTER_AUTH_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}
	if secret := os.Getenv("GO_BETTER_AUTH_SECRET"); secret != "" {
		config.Secret = secret
	}
}
