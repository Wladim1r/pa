package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/Wladim1r/redboard/models"
	"github.com/Wladim1r/redboard/periferia/reddis"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())

	inChan := make(chan models.SecondStat, 5)
	wg := new(sync.WaitGroup)

	redisClient := reddis.NewClient()

	wg.Add(1)
	go func() {
		defer wg.Done()
		redisClient.Subscribe(ctx, inChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				slog.Info("got interruption signal")
				return
			case msg := <-inChan:
				slog.Info("GOT message from redis stream", "msg", msg)
			}
		}
	}()

	<-c
	cancel()
	wg.Wait()
}
