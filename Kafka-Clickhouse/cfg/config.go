package cfg

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Kafka      KafkaConfig
	ClickHouse ClickHouseConfig
}

type KafkaConfig struct {
	Brokers           []string
	Topic             string
	GroupID           string
	MaxRetries        int
	RetryDelay        time.Duration
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration
}

type ClickHouseConfig struct {
	Addr         string
	Database     string
	Table        string
	Username     string
	Password     string
	BatchSize    int
	BatchTimeout time.Duration
	MaxRetries   int
	DialTimeout  time.Duration
}

func Load() Config {
	return Config{
		Kafka: KafkaConfig{
			Brokers:           getSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:             getString("KAFKA_TOPIC", "binance.miniticker"),
			GroupID:           getString("KAFKA_GROUP_ID", "clickhouse-consumer-group"),
			MaxRetries:        getInt("KAFKA_MAX_RETRIES", 5),
			RetryDelay:        getDuration("KAFKA_RETRY_DELAY", 2*time.Second),
			SessionTimeout:    getDuration("KAFKA_SESSION_TIMEOUT", 10*time.Second),
			HeartbeatInterval: getDuration("KAFKA_HEARTBEAT_INTERVAL", 3*time.Second),
		},
		ClickHouse: ClickHouseConfig{
			Addr:         getString("CLICKHOUSE_ADDR", "localhost:9000"),
			Database:     getString("CLICKHOUSE_DATABASE", "crypto"),
			Table:        getString("CLICKHOUSE_TABLE", "market_tickers"),
			Username:     getString("CLICKHOUSE_USERNAME", "default"),
			Password:     getString("CLICKHOUSE_PASSWORD", ""),
			BatchSize:    getInt("BATCH_SIZE", 1000),
			BatchTimeout: getDuration("BATCH_TIMEOUT", 5*time.Second),
			MaxRetries:   getInt("CLICKHOUSE_MAX_RETRIES", 3),
			DialTimeout:  getDuration("CLICKHOUSE_DIAL_TIMEOUT", 10*time.Second),
		},
	}
}

func getString(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultVal
}

func getSlice(key string, defaultVal []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultVal
}
