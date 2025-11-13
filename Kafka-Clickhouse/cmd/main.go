package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/Wladim1r/kafclick/internal/repository"
	"github.com/Wladim1r/kafclick/models"
	"github.com/Wladim1r/kafclick/periferia/clkhouse"
	"github.com/Wladim1r/kafclick/periferia/kaffka"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := new(sync.WaitGroup)

	clickHouseCfg := clkhouse.LoadClickHouseConfig()
	kafkaCfg := kaffka.LoadKafkaConfig()

	chClient := clkhouse.NewClient(ctx, clickHouseCfg)
	defer chClient.Close()

	repo := repository.NewRepository(chClient, clickHouseCfg)

	if err := repo.CreateTable(ctx); err != nil {
		slog.Error("Failed to create table", "error", err)
		os.Exit(1)
	}

	kafkaMsgs := make(chan models.KafkaMsg, 500)

	cons := kaffka.NewConsumer(ctx, kafkaCfg)

	wg.Add(3)
	go cons.Start(ctx, wg, kafkaMsgs)
	go repo.BatchInsert(ctx, wg, kafkaMsgs)

	go func() {
		defer wg.Done()

		count := 0
		for {
			select {
			case <-ctx.Done():
				slog.Info("Message reader stopped", "total_processed", count)
				return
			case msg, ok := <-kafkaMsgs:
				if !ok {
					slog.Info("KafkaMsgs channel closed", "total_processed", count)
					return
				}
				count++
				if count%100 == 0 {
					slog.Info("Processing messages",
						"count", count,
						"last_symbol", msg.Symbol)
				}
			}
		}
	}()

	<-c
	cancel()
	slog.Info("👾 Received shutdown signal")
	slog.Info("⏲️  Waiting for goroutines to finish...")
	wg.Wait()
	slog.Info("🏁 Shutdown complete")
}
