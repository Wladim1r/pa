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

	// metrics
	messagesSent   int64
	messagesFailed int64
	batchesSent    int64
}

func NewProducer(cfg kafkaConfig) *producer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:          cfg.Brockers,
		Topic:            cfg.Topic,
		RequiredAcks:     1, // confirmation from one replic
		MaxAttempts:      cfg.MaxAttempts,
		BatchSize:        cfg.BatchSize,
		WriteTimeout:     cfg.WriteTimeOut,
		Balancer:         &kafka.Hash{},
		CompressionCodec: kafka.Snappy.Codec(),

		Async: false,

		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			slog.Debug("Kafka writer", "message", fmt.Sprintf(msg, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			slog.Error("Kafka writer error", "message", fmt.Sprintf(msg, args...))
		}),
	})

	return &producer{
		writer:      writer,
		config:      cfg,
		batchBuffer: make([]*models.KafkaMsg, 0, cfg.BatchSize),
		batchTimer:  time.NewTimer(cfg.BatchTimeOut),
	}
}

func (p *producer) Start(ctx context.Context, wg *sync.WaitGroup, inputChan chan models.KafkaMsg) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Got interruprion signal, stop to receiving messages from KafkaMsg chan")
			return
		case msg, ok := <-inputChan:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				slog.Info("Got interruprion signal, stop to receiving messages from KafkaMsg chan")
				return
			case <-p.batchTimer.C:
				if len(p.batchBuffer) > 0 {
					p.sendToKafka(ctx)
				}
				p.batchTimer.Reset(p.config.BatchTimeOut)
			default:
				if len(p.batchBuffer) >= p.config.BatchSize {
					p.sendToKafka(ctx)
					p.batchTimer.Reset(p.config.BatchTimeOut)
				} else {
					p.batchBuffer = append(p.batchBuffer, &msg)
				}

			}
		}
	}

}

func (p *producer) sendToKafka(ctx context.Context) {
	arrMsgs := make([]kafka.Message, 0, p.writer.BatchSize)

	select {
	case <-ctx.Done():
		slog.Info("Got interruption signal, stop to sending messages to Kafka")
		return
	default:
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
	}

	start := time.Now()
	err := p.writer.WriteMessages(ctx, arrMsgs...)
	finish := time.Since(start)

	if err != nil {
		p.messagesFailed += int64(len(arrMsgs))
		slog.Error("Failed to send batch to Kafka",
			"error", err,
			"time sending", finish,
			"batch size", len(arrMsgs))
	} else {
		p.messagesSent += int64(len(arrMsgs))
		p.batchesSent++
		slog.Debug("Sent batch to Kafka",
			"time sending", finish,
			"batch size", len(arrMsgs),
			"messages sent", p.messagesSent,
			"batches sent", p.batchesSent)
	}

	p.batchBuffer = p.batchBuffer[:0]
}
