package events

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill-sqlite/wmsqlitezombiezen"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/GoBetterAuth/go-better-auth/env"
	"github.com/GoBetterAuth/go-better-auth/events"
	"github.com/GoBetterAuth/go-better-auth/models"
)

// InitWatermillProvider initializes a Watermill PubSub based on the provider config
func InitWatermillProvider(config *models.EventBusConfig, logger watermill.LoggerAdapter) (models.PubSub, error) {
	if logger == nil {
		logger = watermill.NewStdLogger(false, false)
	}

	provider := events.EventBusProvider(config.Provider)

	switch provider {
	case events.ProviderGoChannel:
		return initGoChannel(logger, config.GoChannel)
	case events.ProviderRedis:
		return initRedis(logger, config.Redis)
	case events.ProviderRabbitMQ:
		return initRabbitMQ(logger, config.RabbitMQ)
	case events.ProviderKafka:
		return initKafka(logger, config.Kafka)
	case events.ProviderNATS:
		return initNATS(logger, config.NATS)
	case events.ProviderPostgres:
		return initPostgres(logger, config.PostgreSQL)
	case events.ProviderSQLite:
		return initSQLite(logger, config.SQLite)
	default:
		return nil, fmt.Errorf("unsupported event bus provider: %s", config.Provider)
	}
}

// initGoChannel initializes an in-memory GoChannel provider
func initGoChannel(logger watermill.LoggerAdapter, config *models.GoChannelConfig) (models.PubSub, error) {
	bufferSize := 100
	if config != nil && config.BufferSize > 0 {
		bufferSize = config.BufferSize
	}

	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            int64(bufferSize),
			Persistent:                     false,
			BlockPublishUntilSubscriberAck: false,
		},
		logger,
	)

	return NewWatermillPubSub(pubSub, pubSub), nil
}

// initSQLite initializes a SQLite provider using the ZombieZen driver
func initSQLite(logger watermill.LoggerAdapter, config *models.SQLiteConfig) (models.PubSub, error) {
	dbPath := "events.db"
	if config != nil && config.DBPath != "" {
		dbPath = config.DBPath
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for sqlite database: %w", err)
		}
	}

	// NewSubscriber expects a connection DSN string
	subscriber, err := wmsqlitezombiezen.NewSubscriber(
		dbPath,
		wmsqlitezombiezen.SubscriberOptions{
			InitializeSchema: true,
			Logger:           logger,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite subscriber: %w", err)
	}

	// For the publisher, we need to get a connection from the pool
	// Create a pool to get a connection for the publisher
	pool, err := sqlitex.NewPool("file:"+dbPath, sqlitex.PoolOptions{
		PoolSize: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite connection pool: %w", err)
	}

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get sqlite connection from pool: %w", err)
	}

	publisher, err := wmsqlitezombiezen.NewPublisher(
		conn,
		wmsqlitezombiezen.PublisherOptions{
			Logger: logger,
		},
	)
	if err != nil {
		pool.Put(conn)
		return nil, fmt.Errorf("failed to create sqlite publisher: %w", err)
	}

	return NewWatermillPubSub(publisher, subscriber), nil
}

// initPostgres initializes a Postgres SQL provider
func initPostgres(logger watermill.LoggerAdapter, config *models.PostgreSQLConfig) (models.PubSub, error) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" && config != nil {
		url = config.URL
	}
	if url == "" {
		return nil, fmt.Errorf("postgres config with url is required (set POSTGRES_URL env var or provide config)")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	subscriber, err := watermillSQL.NewSubscriber(
		db,
		watermillSQL.SubscriberConfig{
			SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
			InitializeSchema: true,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres subscriber: %w", err)
	}

	publisher, err := watermillSQL.NewPublisher(
		db,
		watermillSQL.PublisherConfig{
			SchemaAdapter: watermillSQL.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres publisher: %w", err)
	}

	return NewWatermillPubSub(publisher, subscriber), nil
}

// initRedis initializes a Redis Stream provider
func initRedis(logger watermill.LoggerAdapter, config *models.RedisConfig) (models.PubSub, error) {
	url := os.Getenv(env.EnvRedisURL)
	if url == "" && config != nil {
		url = config.URL
	}
	if url == "" {
		return nil, fmt.Errorf("redis config with url is required (set REDIS_URL env var or provide config)")
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	consumerGroup := os.Getenv(env.EnvEventBusConsumerGroup)
	if consumerGroup == "" {
		if config.ConsumerGroup != "" {
			consumerGroup = config.ConsumerGroup
		} else {
			consumerGroup = "gobetterauth_consumer_group"
		}
	}

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        client,
			ConsumerGroup: consumerGroup,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis subscriber: %w", err)
	}

	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client: client,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis publisher: %w", err)
	}

	return NewWatermillPubSub(publisher, subscriber), nil
}

// initKafka initializes a Kafka provider
func initKafka(logger watermill.LoggerAdapter, config *models.KafkaConfig) (models.PubSub, error) {
	brokersStr := os.Getenv(env.EnvKafkaBrokers)
	if brokersStr == "" && config != nil {
		brokersStr = config.Brokers
	}
	if brokersStr == "" {
		return nil, fmt.Errorf("kafka config with brokers is required")
	}

	brokerList := []string{}
	for b := range strings.SplitSeq(brokersStr, ",") {
		b = strings.TrimSpace(b)
		if b != "" {
			brokerList = append(brokerList, b)
		}
	}
	if len(brokerList) == 0 {
		return nil, fmt.Errorf("kafka config with brokers is required")
	}

	consumerGroup := os.Getenv(env.EnvEventBusConsumerGroup)
	if consumerGroup == "" {
		if config != nil && config.ConsumerGroup != "" {
			consumerGroup = config.ConsumerGroup
		} else {
			consumerGroup = "gobetterauth_consumer_group"
		}
	}

	logger.Debug("kafka init",
		watermill.LogFields{"brokers": strings.Join(brokerList, ","), "consumer_group": consumerGroup},
	)

	saramaSubscriberConfig := kafka.DefaultSaramaSubscriberConfig()
	saramaSubscriberConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	// Configure Kafka producer for optimized batching and throughput.
	// Watermill uses a SyncProducer which requires Return.Successes=true,
	// but we configure batching to reduce latency impact and broker load.
	// Event publishing itself is done asynchronously via goroutines in PublishEventAsync().
	saramaProducerConfig := sarama.NewConfig()
	saramaProducerConfig.Producer.Return.Successes = true                  // Required for SyncProducer
	saramaProducerConfig.Producer.Return.Errors = true                     // Track errors
	saramaProducerConfig.Producer.RequiredAcks = sarama.WaitForLocal       // Wait for local leader (balance durability/speed)
	saramaProducerConfig.Producer.Retry.Max = 3                            // Retry failed publishes up to 3 times
	saramaProducerConfig.Producer.Flush.Frequency = 100 * time.Millisecond // Batch messages every 100ms
	saramaProducerConfig.Producer.Flush.Messages = 100                     // Or batch 100 messages
	saramaProducerConfig.Producer.Flush.MaxMessages = 1000                 // Max batch size
	saramaProducerConfig.Version = sarama.V4_1_0_0                         // Use Kafka 4.1.0+ compatibility

	subscriber, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               brokerList,
			Unmarshaler:           kafka.DefaultMarshaler{},
			ConsumerGroup:         consumerGroup,
			OverwriteSaramaConfig: saramaSubscriberConfig,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka subscriber: %w", err)
	}

	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               brokerList,
			Marshaler:             kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaProducerConfig,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka publisher: %w", err)
	}

	return NewWatermillPubSub(publisher, subscriber), nil
}

// initNATS initializes a NATS provider
func initNATS(logger watermill.LoggerAdapter, config *models.NatsConfig) (models.PubSub, error) {
	url := os.Getenv(env.EnvNatsURL)
	if url == "" && config != nil {
		url = config.URL
	}
	if url == "" {
		return nil, fmt.Errorf("nats config with url is required (set NATS_URL env var or provide config)")
	}

	subscriberConfig := nats.SubscriberConfig{
		URL: url,
	}

	publisherConfig := nats.PublisherConfig{
		URL: url,
	}

	subscriber, err := nats.NewSubscriber(subscriberConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create nats subscriber: %w", err)
	}

	publisher, err := nats.NewPublisher(publisherConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create nats publisher: %w", err)
	}

	return NewWatermillPubSub(publisher, subscriber), nil
}

// initRabbitMQ initializes a RabbitMQ provider
func initRabbitMQ(logger watermill.LoggerAdapter, config *models.RabbitMQConfig) (models.PubSub, error) {
	url := os.Getenv(env.EnvRabbitMQURL)
	if url == "" && config != nil {
		url = config.URL
	}
	if url == "" {
		return nil, fmt.Errorf("rabbitmq config with url is required (set RABBITMQ_URL env var or provide config)")
	}

	rabbitmqConfig := amqp.NewDurableQueueConfig(url)

	subscriber, err := amqp.NewSubscriber(
		rabbitmqConfig,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rabbitmq subscriber: %w", err)
	}

	publisher, err := amqp.NewPublisher(
		rabbitmqConfig,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rabbitmq publisher: %w", err)
	}

	return NewWatermillPubSub(publisher, subscriber), nil
}
