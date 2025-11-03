package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/Wladim1r/aggregator/converting"
	"github.com/Wladim1r/aggregator/kaffka"
	"github.com/Wladim1r/aggregator/models"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	rawMsgsChan := make(chan []byte, 100)
	dailyStatChan := make(chan models.DailyStat, 500)
	kafkaMsgChan := make(chan models.KafkaMsg, 500)

	cfg := kaffka.LoadKafkaConfig()
	producer := kaffka.NewProducer(cfg)

	wg.Add(4)
	go converting.ReveiveMessage(ctx, wg, rawMsgsChan)
	go converting.ConvertRawToArrDS(ctx, wg, rawMsgsChan, dailyStatChan)
	go converting.ReceiveKafkaMsg(ctx, wg, dailyStatChan, kafkaMsgChan)
	go producer.Start(ctx, wg, kafkaMsgChan)

	// go func(ctx context.Context, wg *sync.WaitGroup) {
	// 	defer wg.Done()
	//
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case msg := <-kafkaMsgChan:
	// 			fmt.Println(msg)
	// 		}
	// 	}
	// }(ctx, wg)

	<-c
	cancel()
	slog.Info("👾 Received Interruption signal")
	slog.Info("⏲️ Wait for finishing all the goroutines...")
	wg.Wait()
	slog.Info("🏁 It is over 😢")
}
