package main

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/Wladim1r/kafclick/cfg"
	"github.com/Wladim1r/kafclick/consumer"
	"github.com/Wladim1r/kafclick/models"
)

func main() {
	c := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	kafkaMsgs := make(chan models.KafkaMsg, 500)

	cfg := cfg.Load()
	cons := consumer.NewConsumer(ctx, cfg.Kafka)

	wg.Add(2)
	go cons.Start(ctx, wg, kafkaMsgs)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-kafkaMsgs:
				slog.Info("Got msg from kafkaMsgs chan", "data", msg)
			}
		}
	}()

	<-c
	cancel()
	wg.Wait()
}
