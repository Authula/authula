package env

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	// OAUTH2 PROVIDERS

	EnvGoogleClientID     = "GOOGLE_CLIENT_ID"
	EnvGoogleClientSecret = "GOOGLE_CLIENT_SECRET"

	EnvDiscordClientID     = "DISCORD_CLIENT_ID"
	EnvDiscordClientSecret = "DISCORD_CLIENT_SECRET"

	EnvGithubClientID     = "GITHUB_CLIENT_ID"
	EnvGithubClientSecret = "GITHUB_CLIENT_SECRET"

	// POSTGRES

	EnvPostgresURL = "POSTGRES_URL"

	// REDIS

	EnvRedisURL = "REDIS_URL"

	// KAFKA

	EnvKafkaBrokers = "KAFKA_BROKERS"

	// NATS

	EnvNatsURL = "NATS_URL"

	// RabbitMQ

	EnvRabbitMQURL = "RABBITMQ_URL"

	// EVENT BUS

	EnvEventBusConsumerGroup = "EVENT_BUS_CONSUMER_GROUP"

	// AUTHULA

	EnvOpenAPISpecVersion = "OPENAPI_SPEC_VERSION"
	EnvConfigPath         = "AUTHULA_CONFIG_PATH"
	EnvBaseURL            = "AUTHULA_BASE_URL"
	EnvSecret             = "AUTHULA_SECRET"
	EnvDatabaseURL        = "AUTHULA_DATABASE_URL"

	// ENVIRONMENT

	EnvGoEnvironment = "GO_ENV"
	EnvPort          = "PORT"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func LoadEnvConfig() {
	env := os.Getenv("GO_ENV")

	// Only attempt to load .env files in local development.
	// Production configurations should be explicitly injected into the environment.
	if env == "" || strings.ToLower(env) == "development" {
		if err := godotenv.Load(); err != nil {
			log.Println("Note: No .env file found, relying on system environment variables.")
		}
	}
}
