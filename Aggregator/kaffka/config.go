package kaffka

import (
	"time"

	"github.com/Wladim1r/aggregator/lib/getenv"
)

type kafkaConfig struct {
	Brockers     []string // ["localhost:9092"]
	Topic        string   // binance.miniticker
	BatchSize    int
	BatchTimeOut time.Duration
	RequiredAcks int
	MaxAttempts  int
	WriteTimeOut time.Duration
}

func LoadKafkaConfig() kafkaConfig {
	return kafkaConfig{
		Brockers:     getenv.GetSlice("BROKERS", []string{"localhost:9092"}),
		Topic:        getenv.GetString("TOPIC", "binance.miniticker"),
		BatchSize:    getenv.GetInt("BATCH_SIZE", 120),
		BatchTimeOut: getenv.GetTime("BATCH_TIMEOUT", 2*time.Second),
		RequiredAcks: getenv.GetInt("ACK", 1),
		MaxAttempts:  getenv.GetInt("MAX_ATTEMPTS", 3),
		WriteTimeOut: getenv.GetTime("WRITE_TIMEOUT", 10*time.Second),
	}
}
