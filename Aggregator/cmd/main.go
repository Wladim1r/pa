package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/Wladim1r/aggregator/gateway/converting"
	"github.com/Wladim1r/aggregator/gateway/strman"
	"github.com/Wladim1r/aggregator/lib/getenv"
	"github.com/Wladim1r/aggregator/periferia/reddis"
	"github.com/gin-gonic/gin"

	"github.com/Wladim1r/aggregator/models"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	rawMsgsChan := make(chan []byte, 300)

	rawAggTradeChan := make(chan []byte, 100)
	rawMiniTickerChan := make(chan []byte, 100)

	secondStatChan := make(chan models.SecondStat, 100)
	dailyStatChan := make(chan models.DailyStat, 500)
	// kafkaMsgChan := make(chan models.KafkaMsg, 500)

	streamManager := strman.NewStreamManager()

	r := gin.Default()

	// start stream
	r.GET("/coin", func(c *gin.Context) {
		// now query like /symbol?name=btcusdt&id=3

		symbol := c.Query("symbol")
		idStr := c.Query("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to parse 'id' into int: " + err.Error(),
			})
			return
		}

		started := streamManager.AddCoin(ctx, wg, symbol, id, rawMsgsChan)

		if started {
			c.JSON(http.StatusOK, gin.H{
				"status": "started",
				"symbol": symbol,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"status": fmt.Sprintf("for user number %d already active", id),
				"symbol": symbol,
			})
		}
	})

	// stop stream
	r.DELETE("/coin", func(c *gin.Context) {
		symbol := c.Query("symbol")
		idStr := c.Query("id")

		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "failed to parse 'id' into int: " + err.Error(),
			})
			return
		}

		streamManager.DeleteCoin(symbol, id)

		c.JSON(http.StatusOK, gin.H{
			"status": fmt.Sprintf("for user number %d stopped", id),
			"symbol": symbol,
		})
	})

	server := http.Server{
		Addr:    getenv.GetString("SERVER_ADDR", ":8088"),
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Failed to run server", "error", err)
			c <- os.Interrupt
		}
	}()

	// cfgKafka := kaffka.LoadKafkaConfig()
	// producer := kaffka.NewProducer(cfgKafka)

	cfgRedis := reddis.LoadRedisConfig()
	saver := reddis.NewSaver(cfgRedis, streamManager)

	wg.Add(7)

	go converting.DistributeMessages(ctx, wg, rawMsgsChan, rawAggTradeChan, rawMiniTickerChan)

	go converting.ReceiveMiniTickerMessage(ctx, wg, rawMsgsChan)

	go converting.ConvertRawToArrDS(ctx, wg, rawMiniTickerChan, dailyStatChan)
	go converting.ConvertRawToSS(ctx, wg, rawAggTradeChan, secondStatChan)

	// go converting.ReceiveKafkaMsg(ctx, wg, dailyStatChan, kafkaMsgChan)
	// go producer.Start(ctx, wg, kafkaMsgChan)

	go saver.Start(ctx, wg, secondStatChan)

	<-c
	slog.Info("👾 Received Interruption signal")

	// Отменяем контекст для всех горутин
	cancel()

	// Останавливаем HTTP сервер с таймаутом
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to gracefully shutdown HTTP server", "error", err)
	} else {
		slog.Info("✅ HTTP server stopped gracefully")
	}

	slog.Info("⏲️ Wait for finishing all the goroutines...")
	wg.Wait()
	slog.Info("🏁 It is over 😢")
}
