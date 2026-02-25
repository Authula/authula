package events

import (
	"os"
	"testing"

	"github.com/GoBetterAuth/go-better-auth/v2/models"
	"github.com/ThreeDotsLabs/watermill"
)

func TestInitWatermillProvider_GoChannel(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider:  "gochannel",
		GoChannel: &models.GoChannelConfig{BufferSize: 100},
	}

	pubsub, err := InitWatermillProvider(config, logger)
	if err != nil {
		t.Fatalf("failed to initialize gochannel provider: %v", err)
	}
	defer func() {
		if err := pubsub.Close(); err != nil {
			t.Errorf("failed to close pubsub: %v", err)
		}
	}()

	if pubsub == nil {
		t.Fatal("expected pubsub to be non-nil")
	}
}

func TestInitWatermillProvider_GoChannel_DefaultBufferSize(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider:  "gochannel",
		GoChannel: &models.GoChannelConfig{},
	}

	pubsub, err := InitWatermillProvider(config, logger)
	if err != nil {
		t.Fatalf("failed to initialize gochannel provider with default buffer: %v", err)
	}
	defer func() {
		if err := pubsub.Close(); err != nil {
			t.Errorf("failed to close pubsub: %v", err)
		}
	}()

	if pubsub == nil {
		t.Fatal("expected pubsub to be non-nil")
	}
}

func TestInitWatermillProvider_GoChannel_NilConfig(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider:  "gochannel",
		GoChannel: nil,
	}

	pubsub, err := InitWatermillProvider(config, logger)
	if err != nil {
		t.Fatalf("failed to initialize gochannel provider with nil config: %v", err)
	}
	defer func() {
		if err := pubsub.Close(); err != nil {
			t.Errorf("failed to close pubsub: %v", err)
		}
	}()

	if pubsub == nil {
		t.Fatal("expected pubsub to be non-nil")
	}
}

func TestInitWatermillProvider_UnsupportedProvider(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "unsupported",
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
}

func TestInitWatermillProvider_Redis_MissingURL(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "redis",
		Redis:    &models.RedisConfig{},
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing redis URL, got nil")
	}
}

func TestInitWatermillProvider_Redis_NilConfig(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "redis",
		Redis:    nil,
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing redis config, got nil")
	}
}

func TestInitWatermillProvider_Kafka_MissingBrokers(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "kafka",
		Kafka:    &models.KafkaConfig{},
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing kafka brokers, got nil")
	}
}

func TestInitWatermillProvider_Kafka_NilConfig(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "kafka",
		Kafka:    nil,
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing kafka config, got nil")
	}
}

func TestInitWatermillProvider_NATS_MissingURL(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "nats",
		NATS:     &models.NatsConfig{},
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing nats URL, got nil")
	}
}

func TestInitWatermillProvider_NATS_NilConfig(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "nats",
		NATS:     nil,
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing nats config, got nil")
	}
}

func TestInitWatermillProvider_Postgres_MissingURL(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider:   "postgres",
		PostgreSQL: &models.PostgreSQLConfig{},
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing postgres URL, got nil")
	}
}

func TestInitWatermillProvider_Postgres_NilConfig(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider:   "postgres",
		PostgreSQL: nil,
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing postgres config, got nil")
	}
}

func TestInitWatermillProvider_SQLite_DefaultPath(t *testing.T) {
	if os.Getenv("GO_BETTER_AUTH_TEST_DB") != "" && os.Getenv("GO_BETTER_AUTH_TEST_DB") != "sqlite" {
		t.Skip("skipping SQLite test when GO_BETTER_AUTH_TEST_DB is set to a different provider")
	}

	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "sqlite",
		SQLite:   nil, // should default to events.db
	}

	pubsub, err := InitWatermillProvider(config, logger)
	if err != nil {
		t.Fatalf("failed to initialize sqlite provider with default path: %v", err)
	}
	defer func() {
		if err := pubsub.Close(); err != nil {
			t.Errorf("failed to close pubsub: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove("events.db"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove events.db: %v", err)
		}
		if err := os.Remove("events.db-shm"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove events.db-shm: %v", err)
		}
		if err := os.Remove("events.db-wal"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove events.db-wal: %v", err)
		}
	}()

	if pubsub == nil {
		t.Fatal("expected pubsub to be non-nil")
	}
}

func TestInitWatermillProvider_SQLite_EmptyPath(t *testing.T) {
	if os.Getenv("GO_BETTER_AUTH_TEST_DB") != "" && os.Getenv("GO_BETTER_AUTH_TEST_DB") != "sqlite" {
		t.Skip("skipping SQLite test when GO_BETTER_AUTH_TEST_DB is set to a different provider")
	}

	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "sqlite",
		SQLite:   &models.SQLiteConfig{DBPath: ""}, // should default to events.db
	}

	pubsub, err := InitWatermillProvider(config, logger)
	if err != nil {
		t.Fatalf("failed to initialize sqlite provider with empty path: %v", err)
	}
	defer func() {
		if err := pubsub.Close(); err != nil {
			t.Errorf("failed to close pubsub: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove("events.db"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove events.db: %v", err)
		}
		if err := os.Remove("events.db-shm"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove events.db-shm: %v", err)
		}
		if err := os.Remove("events.db-wal"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove events.db-wal: %v", err)
		}
	}()

	if pubsub == nil {
		t.Fatal("expected pubsub to be non-nil")
	}
}

func TestInitWatermillProvider_SQLite_CustomPath(t *testing.T) {
	if os.Getenv("GO_BETTER_AUTH_TEST_DB") != "" && os.Getenv("GO_BETTER_AUTH_TEST_DB") != "sqlite" {
		t.Skip("skipping SQLite test when GO_BETTER_AUTH_TEST_DB is set to a different provider")
	}

	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "sqlite",
		SQLite:   &models.SQLiteConfig{DBPath: "/tmp/test_events.db"},
	}

	pubsub, err := InitWatermillProvider(config, logger)
	if err != nil {
		t.Fatalf("failed to initialize sqlite provider with custom path: %v", err)
	}
	defer func() {
		if err := pubsub.Close(); err != nil {
			t.Errorf("failed to close pubsub: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove("/tmp/test_events.db"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove /tmp/test_events.db: %v", err)
		}
		if err := os.Remove("/tmp/test_events.db-shm"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove /tmp/test_events.db-shm: %v", err)
		}
		if err := os.Remove("/tmp/test_events.db-wal"); err != nil && !os.IsNotExist(err) {
			t.Errorf("failed to remove /tmp/test_events.db-wal: %v", err)
		}
	}()

	if pubsub == nil {
		t.Fatal("expected pubsub to be non-nil")
	}
}

func TestInitWatermillProvider_RabbitMQ_MissingURL(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "rabbitmq",
		RabbitMQ: &models.RabbitMQConfig{},
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing rabbitmq URL, got nil")
	}
}

func TestInitWatermillProvider_RabbitMQ_NilConfig(t *testing.T) {
	logger := watermill.NewStdLogger(false, false)

	config := &models.EventBusConfig{
		Provider: "rabbitmq",
		RabbitMQ: nil,
	}

	_, err := InitWatermillProvider(config, logger)
	if err == nil {
		t.Fatal("expected error for missing rabbitmq config, got nil")
	}
}
