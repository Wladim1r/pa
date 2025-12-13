package converting

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wladim1r/aggregator/models"
)

func ConvertRawToSS(
	ctx context.Context,
	wg *sync.WaitGroup,
	inChan chan []byte,
	outChan chan models.SecondStat,
) {
	defer wg.Done()
	defer close(outChan)

	wgWorker := new(sync.WaitGroup)
	aggTrChan := make(chan models.AggTrade, 100)

	wgWorker.Add(1)
	go receiveSecondStat(ctx, wgWorker, aggTrChan, outChan)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Got Interruption signal, stopping to converting messages from stream")
			wgWorker.Wait()
			return
		case msg, ok := <-inChan:
			if !ok {
				return
			}
			var aggTrade models.AggTrade
			if err := json.Unmarshal(msg, &aggTrade); err != nil {
				slog.Error("Could not parse bytes into aggTrade struct", "error", err)
				continue
			}

			select {
			case <-ctx.Done():
				slog.Info("Got Interruption signal, stopping to converting messages from stream")
				wgWorker.Wait()
				return
			case aggTrChan <- aggTrade:
			}
		}
	}
}

func receiveSecondStat(
	ctx context.Context,
	wg *sync.WaitGroup,
	inChan chan models.AggTrade,
	outChan chan models.SecondStat,
) {
	defer wg.Done()

	// Map to store latest price per symbol
	latestPrices := make(map[string]float64)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"Got Interruption signal, stopping to receiving messages from aggTrade chan",
			)
			return
		case msg, ok := <-inChan:
			if !ok {
				return
			}
			// Update latest price for this symbol
			symbol := strings.ToLower(msg.Symbol)
			latestPrices[symbol] = msg.PriceFloat()
		case <-ticker.C:
			// Every second, send latest prices for all symbols
			for symbol, price := range latestPrices {
				secondStat := models.SecondStat{
					Symbol: symbol,
					Price:  price,
				}
				select {
				case <-ctx.Done():
					slog.Info(
						"Got Interruption signal, stopping to receiving messages from aggTrade chan",
					)
					return
				case outChan <- secondStat:
					// Successfully sent
				default:
					// Channel is full, but this should be rare now with 1-second aggregation
					// Log only if it happens multiple times
					slog.Debug("SecondStat channel is full, dropping message",
						"symbol", symbol,
						"price", price)
				}
			}
			// Clear the map after sending (optional - we could keep accumulating)
			// latestPrices = make(map[string]float64)
		}
	}
}
