package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/Wladimir/socket-service/connsock"
	"github.com/Wladimir/socket-service/svr"
)

const (
	MiniTickerURL = "wss://stream.binance.com:443/ws/!miniTicker@arr"
	Port          = ":50051"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	wg := new(sync.WaitGroup)

	ctx, cancel := context.WithCancel(context.Background())

	outputChan := make(chan []byte, 100)
	msgChan := make(chan error, 1)

	socketConn := connsock.NewSocketProduecer(outputChan, MiniTickerURL, msgChan)
	wg.Add(1)
	go socketConn.Start(ctx, wg)

	wg.Add(1)
	go svr.StartServer(wg, outputChan, ctx)

	slog.Info("🚀 Server started", "port", Port)

	<-c
	cancel()
	slog.Info("👾 Received Interruption signal")
	slog.Info("⏲️ Wait for finishing all the goroutines...")
	wg.Wait()
	slog.Info("🏁 It is over 😢")
}
