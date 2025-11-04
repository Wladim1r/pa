package converting

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/Wladim1r/aggregator/models"
	"github.com/Wladim1r/proto-crypto/gen/socket"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	Address = "socket-service:50051"
)

func ReveiveMessage(ctx context.Context, wg *sync.WaitGroup, outChan chan []byte) {
	defer wg.Done()

	var conn *grpc.ClientConn
	var err error

	// Retry connection with exponential backoff
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		conn, err = grpc.NewClient(
			Address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			delay := time.Duration(i+1) * time.Second
			slog.Warn(
				"Failed to connect, retrying...",
				"attempt",
				i+1,
				"delay",
				delay,
				"error",
				err,
			)
			time.Sleep(delay)
		}
	}

	if err != nil {
		slog.Error("Failed to connect after retries", "address", Address, "error", err)
		return
	}
	defer conn.Close()

	client := socket.NewSocketServiceClient(conn)

	// Retry stream connection
	var stream socket.SocketService_ReceiveRawMsgClient
	for i := 0; i < maxRetries; i++ {
		stream, err = client.ReceiveRawMsg(ctx, &socket.RawMsgRequest{})
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			delay := time.Duration(i+1) * time.Second
			slog.Warn(
				"Could not set up stream, retrying...",
				"attempt",
				i+1,
				"delay",
				delay,
				"error",
				err,
			)
			time.Sleep(delay)
		}
	}

	if err != nil {
		slog.Error("Could not set up connection with ReceiveRawMsg after retries", "error", err)
		return
	}

	slog.Info("📞 Starting to receive messages from stream")
	for {
		select {
		case <-ctx.Done():
			slog.Info("Got Interruption signal, stopping to receive messages from stream")
		default:
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					slog.Info("Stream closed by server")
					return
				}
				slog.Error("Got error from stream during receiving", "error", err)
				return
			}
			// slog.Info("Got message", "msg", string(resp.GetData()))
			select {
			case <-ctx.Done():
				slog.Info("Got Interruption signal, stopping to receive messages from stream")
			default:
				outChan <- resp.Data
			}
		}
	}
}

func ConvertRawToArrDS(
	ctx context.Context,
	wg *sync.WaitGroup,
	inputChan chan []byte,
	outputChan chan models.DailyStat,
) {
	defer wg.Done()
	defer close(outputChan)

	numWorkers := 4
	workerChan := make(chan models.MiniTicker, 100)
	wgWorker := new(sync.WaitGroup)

	defer close(workerChan)

	wgWorker.Add(numWorkers)
	for range 4 {
		go ReceiveDailyStat(ctx, wgWorker, workerChan, outputChan)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Got Interruption signal, stopping to converting messages from stream")
			return
		case arrBytes := <-inputChan:
			var arrMsgs []models.MiniTicker
			if err := json.Unmarshal(arrBytes, &arrMsgs); err != nil {
				slog.Error("Could not parse from arrBytes into array of miniTickers", "error", err)
				continue
			}

			fmt.Println("=========================================")
			slog.Debug("Received batch", "size", len(arrMsgs))

			select {
			case <-ctx.Done():
				wgWorker.Wait()
				slog.Info("Got Interruption signal, stopping to converting messages from stream")
			default:
				for _, msg := range arrMsgs {
					select {
					case <-ctx.Done():
						wgWorker.Wait()
						slog.Info(
							"Got Interruption signal, stopping to converting messages from stream",
						)
						return
					case workerChan <- msg:
					}
				}
			}

		}
	}
}

func ReceiveDailyStat(
	ctx context.Context,
	wg *sync.WaitGroup,
	inputChan chan models.MiniTicker,
	outputChan chan models.DailyStat,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"Got Interruption signal, stopping to receiving messages from MiniTicker chan",
			)
			return
		case msg, ok := <-inputChan:
			if !ok {
				return
			}
			dailyStat := models.DailyStat{
				EventType:  msg.EventType,
				EventTime:  msg.EventTime,
				RecvTime:   time.Now().UnixMilli(),
				Symbol:     msg.Symbol,
				ClosePrice: msg.ClosePriceFloat(),
				OpenPrice:  msg.OpenPriceFloat(),
				HighPrice:  msg.HighPriceFloat(),
				LowPrice:   msg.LowPriceFloat(),
			}
			select {
			case <-ctx.Done():
				slog.Info(
					"Got Interruption signal, stopping to receiving messages from MiniTicker chan",
				)
				return
			default:
				outputChan <- dailyStat
			}
		}
	}
}

func ReceiveKafkaMsg(
	ctx context.Context,
	wg *sync.WaitGroup,
	inputChan chan models.DailyStat,
	outputChan chan models.KafkaMsg,
) {
	defer wg.Done()
	defer close(outputChan)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Got Interruption signal, stopping to sending messages from DailyStat chan")
			return
		case msg, ok := <-inputChan:
			if !ok {
				return
			}
			kafkaMsg := models.KafkaMsg{
				MessageID:     uuid.New().String(),
				EventType:     msg.EventType,
				EventTime:     msg.EventTime,
				RecvTime:      msg.RecvTime,
				Symbol:        msg.Symbol,
				ClosePrice:    decimal.NewFromFloat(msg.ClosePrice),
				OpenPrice:     decimal.NewFromFloat(msg.OpenPrice),
				HighPrice:     decimal.NewFromFloat(msg.HighPrice),
				LowPrice:      decimal.NewFromFloat(msg.LowPrice),
				ChangePrice:   msg.ChangeInPrice(),
				ChangePercent: msg.ChangeInPercent(),
			}
			select {
			case <-ctx.Done():
				slog.Info(
					"Got Interruption signal, stopping to sending messages from DailyStat chan",
				)
				return
			default:
				outputChan <- kafkaMsg
			}
		}
	}
}
