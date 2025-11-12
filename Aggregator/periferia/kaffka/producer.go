package kaffka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Wladim1r/aggregator/models"
	"github.com/segmentio/kafka-go"
)

type producer struct {
	writer      *kafka.Writer
	config      kafkaConfig
	batchBuffer []*models.KafkaMsg
	batchTimer  *time.Timer
}

func NewProducer(cfg kafkaConfig) *producer {
	slog.Info("🔄 Initializing Kafka producer", "brokers", cfg.Brockers, "topic", cfg.Topic)

	// Сначала проверим доступность Kafka
	maxRetries := 10
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		slog.Info("Checking Kafka availability", "attempt", i+1, "broker", cfg.Brockers[0])

		conn, err := kafka.DialContext(context.Background(), "tcp", cfg.Brockers[0])
		if err != nil {
			lastErr = err
			delay := time.Duration(i+1) * 2 * time.Second
			slog.Warn("Kafka not available yet",
				"attempt", i+1,
				"error", err,
				"delay", delay)
			time.Sleep(delay)
			continue
		}

		// Проверяем, что можем получить metadata
		brokers, err := conn.Brokers()
		if err != nil {
			conn.Close()
			lastErr = err
			delay := time.Duration(i+1) * 2 * time.Second
			slog.Warn("Failed to get Kafka brokers metadata",
				"attempt", i+1,
				"error", err,
				"delay", delay)
			time.Sleep(delay)
			continue
		}

		conn.Close()
		slog.Info("✅ Kafka is available", "brokers", len(brokers))
		break
	}

	if lastErr != nil {
		slog.Error("❌ Failed to verify Kafka availability after all retries", "error", lastErr)
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          cfg.Brockers,
		Topic:            cfg.Topic,
		RequiredAcks:     cfg.RequiredAcks,
		MaxAttempts:      cfg.MaxAttempts,
		BatchSize:        cfg.BatchSize,
		WriteTimeout:     cfg.WriteTimeOut,
		Balancer:         &kafka.Hash{},
		CompressionCodec: kafka.Snappy.Codec(),
		Async:            false,
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			slog.Debug("Kafka writer", "message", fmt.Sprintf(msg, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			slog.Error("Kafka writer error", "message", fmt.Sprintf(msg, args...))
		}),
	})

	slog.Info("✅ Kafka producer initialized successfully")

	return &producer{
		writer:      writer,
		config:      cfg,
		batchBuffer: make([]*models.KafkaMsg, 0, cfg.BatchSize),
		batchTimer:  time.NewTimer(cfg.BatchTimeOut),
	}
}

func (p *producer) Start(ctx context.Context, wg *sync.WaitGroup, inputChan chan models.KafkaMsg) {
	defer wg.Done()
	defer p.writer.Close() // Добавьте закрытие writer

	slog.Info("🚀 Kafka producer started")

	for {
		select {
		case <-ctx.Done():
			// Отправляем оставшиеся сообщения перед выходом
			if len(p.batchBuffer) > 0 {
				slog.Info("Sending remaining messages before shutdown", "count", len(p.batchBuffer))
				p.sendToKafka(ctx)
			}
			slog.Info("Got interruption signal, stop to receiving messages from KafkaMsg chan")
			return

		case <-p.batchTimer.C:
			if len(p.batchBuffer) > 0 {
				slog.Debug("Timer triggered, sending batch", "size", len(p.batchBuffer))
				p.sendToKafka(ctx)
			}
			p.batchTimer.Reset(p.config.BatchTimeOut)

		case msg, ok := <-inputChan:
			if !ok {
				// Канал закрыт, отправляем оставшиеся сообщения
				if len(p.batchBuffer) > 0 {
					slog.Info(
						"Input channel closed, sending remaining messages",
						"count",
						len(p.batchBuffer),
					)
					p.sendToKafka(ctx)
				}
				return
			}

			// Добавляем сообщение в буфер
			p.batchBuffer = append(p.batchBuffer, &msg)

			// Если буфер заполнен, отправляем
			if len(p.batchBuffer) >= p.config.BatchSize {
				slog.Debug("Batch full, sending", "size", len(p.batchBuffer))
				p.sendToKafka(ctx)
				p.batchTimer.Reset(p.config.BatchTimeOut)
			}
		}
	}
}

func (p *producer) sendToKafka(ctx context.Context) {
	if len(p.batchBuffer) == 0 {
		return
	}

	arrMsgs := make([]kafka.Message, 0, len(p.batchBuffer))

	for _, msg := range p.batchBuffer {
		jsonData, err := json.Marshal(msg)
		if err != nil {
			slog.Error("Could not parse msg into JSON", "error", err)
			continue
		}
		arrMsgs = append(arrMsgs, kafka.Message{
			Key:   []byte(msg.Symbol),
			Value: jsonData,
			Time:  time.UnixMilli(msg.RecvTime),
			Headers: []kafka.Header{
				{Key: "message_id", Value: []byte(msg.MessageID)},
			},
		})
	}

	if len(arrMsgs) == 0 {
		p.batchBuffer = p.batchBuffer[:0]
		return
	}

	start := time.Now()
	err := p.writer.WriteMessages(ctx, arrMsgs...)
	finish := time.Since(start)

	if err != nil {
		slog.Error("❌ Failed to send batch to Kafka",
			"error", err,
			"time_sending", finish,
			"batch_size", len(arrMsgs))
	} else {
		slog.Info("✅ Sent batch to Kafka",
			"time_sending", finish,
			"batch_size", len(arrMsgs))
	}

	p.batchBuffer = p.batchBuffer[:0]
}
