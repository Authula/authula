package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"

	gobetterauth "github.com/GoBetterAuth/go-better-auth"
	gobetterauthconfig "github.com/GoBetterAuth/go-better-auth/config"
	gobetterauthmodels "github.com/GoBetterAuth/go-better-auth/models"
)

func getEnv(key, fallback string) string {
	// First check viper (includes .env file values)
	if value := viper.GetString(key); value != "" {
		return value
	}
	// Fall back to direct OS env lookup
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// initViper initializes viper for configuration management
func initViper() error {
	// Set up viper for .env file loading
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	// Enable automatic environment variable binding with GBA_ prefix
	// ENV vars use format: GBA_DATABASE__PROVIDER, GBA_SOCIAL_PROVIDERS__PROVIDERS__GITHUB__CLIENT_SECRET
	viper.SetEnvPrefix("GBA")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))

	// Try to read .env file, but don't fail if it doesn't exist in production
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// .env file not found - this is okay in production
			return nil
		}
		// Config file was found but another error occurred
		return err
	}
	return nil
}

// Run GoBetterAuth in standalone mode
func main() {
	err := initViper()
	if err != nil {
		env := os.Getenv("GO_ENV")
		if env != "production" {
			log.Fatal("Failed to initialize configuration", "error", err)
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
	authConfig := gobetterauthconfig.NewConfig(
		gobetterauthconfig.WithMode(gobetterauthmodels.ModeStandalone),
		gobetterauthconfig.WithAppName(tomlConfig.AppName),
		gobetterauthconfig.WithBaseURL(tomlConfig.BaseURL),
		gobetterauthconfig.WithBasePath(tomlConfig.BasePath),
		gobetterauthconfig.WithSecret(tomlConfig.Secret),
		gobetterauthconfig.WithLogger(tomlConfig.Logger),
		gobetterauthconfig.WithDatabase(tomlConfig.Database),
		gobetterauthconfig.WithEmailConfig(tomlConfig.Email),
		gobetterauthconfig.WithSecondaryStorage(tomlConfig.SecondaryStorage),
		gobetterauthconfig.WithEmailPassword(tomlConfig.EmailPassword),
		gobetterauthconfig.WithEmailVerification(tomlConfig.EmailVerification),
		gobetterauthconfig.WithUser(tomlConfig.User),
		gobetterauthconfig.WithSession(tomlConfig.Session),
		gobetterauthconfig.WithCSRF(tomlConfig.CSRF),
		gobetterauthconfig.WithSocialProviders(tomlConfig.SocialProviders),
		gobetterauthconfig.WithTrustedOrigins(tomlConfig.TrustedOrigins),
		gobetterauthconfig.WithRateLimit(tomlConfig.RateLimit),
		gobetterauthconfig.WithEventBus(tomlConfig.EventBus),
		gobetterauthconfig.WithEndpointHooks(tomlConfig.EndpointHooks),
		gobetterauthconfig.WithDatabaseHooks(tomlConfig.DatabaseHooks),
		gobetterauthconfig.WithEventHooks(tomlConfig.EventHooks),
		gobetterauthconfig.WithWebhooks(tomlConfig.Webhooks),
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
func loadConfigFromFile() gobetterauthmodels.Config {
	configPath := getEnv("GO_BETTER_AUTH_CONFIG_PATH", "config.toml")
	var config gobetterauthmodels.Config

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
func applyConfigDefaults(config *gobetterauthmodels.Config) {
	// Override critical settings from environment variables (via viper with GBA_ prefix)
	// e.g., GBA_BASE_URL, GBA_SECRET
	if baseURL := viper.GetString("base_url"); baseURL != "" {
		config.BaseURL = baseURL
	}
	if secret := viper.GetString("secret"); secret != "" {
		config.Secret = secret
	}
}
