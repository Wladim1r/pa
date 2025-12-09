package converting

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

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

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"Got Interruption signal, stopping to receiving messages from MiniTicker chan",
			)
			return
		case msg, ok := <-inChan:
			if !ok {
				return
			}
			secondStat := models.SecondStat{
				Symbol: strings.ToLower(msg.Symbol),
				Price:  msg.PriceFloat(),
			}
			select {
			case <-ctx.Done():
				slog.Info(
					"Got Interruption signal, stopping to receiving messages from MiniTicker chan",
				)
				return
			default:
				outChan <- secondStat
			}
		}
	}
}
