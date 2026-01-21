package env

const (
	// OAUTH2 PROVIDERS

	EnvGoogleClientID     = "GOOGLE_CLIENT_ID"
	EnvGoogleClientSecret = "GOOGLE_CLIENT_SECRET"

	EnvDiscordClientID     = "DISCORD_CLIENT_ID"
	EnvDiscordClientSecret = "DISCORD_CLIENT_SECRET"

	EnvGithubClientID     = "GITHUB_CLIENT_ID"
	EnvGithubClientSecret = "GITHUB_CLIENT_SECRET"

	// EMAIL / SMTP

	EnvResendApiKey = "RESEND_API_KEY"

	EnvSMTPHost  = "SMTP_HOST"
	EnvSMTPPort  = "SMTP_PORT"
	EnvSMTPUser  = "SMTP_USER"
	EnvSMTPPass  = "SMTP_PASS"
	EnvEmailFrom = "FROM_ADDRESS"

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

	// GO BETTER AUTH

	EnvConfigPath  = "GO_BETTER_AUTH_CONFIG_PATH"
	EnvAdminApiKey = "GO_BETTER_AUTH_ADMIN_API_KEY"
	EnvBaseURL     = "GO_BETTER_AUTH_BASE_URL"
	EnvSecret      = "GO_BETTER_AUTH_SECRET"
	EnvDatabaseURL = "GO_BETTER_AUTH_DATABASE_URL"

	// ENVIRONMENT

	EnvGoEnvironment = "GO_ENV"
	EnvPort          = "PORT"
)
