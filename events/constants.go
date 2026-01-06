package events

type EventBusProvider string

const (
	ProviderGoChannel EventBusProvider = "gochannel"
	ProviderSQLite    EventBusProvider = "sqlite"
	ProviderPostgres  EventBusProvider = "postgres"
	ProviderRedis     EventBusProvider = "redis"
	ProviderKafka     EventBusProvider = "kafka"
	ProviderNATS      EventBusProvider = "nats"
	ProviderRabbitMQ  EventBusProvider = "rabbitmq"
)

func (p EventBusProvider) String() string {
	return string(p)
}

func (p EventBusProvider) Valid() bool {
	switch p {
	case ProviderGoChannel, ProviderSQLite, ProviderPostgres, ProviderRedis, ProviderKafka, ProviderNATS, ProviderRabbitMQ:
		return true
	}
	return false
}
