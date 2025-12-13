package converting

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/Wladim1r/aggregator/lib/getenv"
	"github.com/Wladim1r/proto-crypto/gen/socket-aggregator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	Address    = getenv.GetString("SOCKET_SERVICE_ADDR", "socket-service:50051")
	MaxRetries = getenv.GetInt("SOCKET_SERVICE_MAX_RETRIES", 10)
)

type StreamReceiver interface {
	Recv() (*socket.RawResponse, error)
}

func DistributeMessages(
	ctx context.Context,
	wg *sync.WaitGroup,
	rawChan, aggTradeChan, miniTickerChan chan []byte,
) {
	defer wg.Done()
	defer close(aggTradeChan)
	defer close(miniTickerChan)

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-rawChan:
			if !ok {
				return
			}
			if len(msg) == 0 {
				continue
			}

			if msg[0] == '{' {
				select {
				case <-ctx.Done():
					return
				case aggTradeChan <- msg:
				}
			}

			if msg[0] == '[' {
				select {
				case <-ctx.Done():
					return
				case miniTickerChan <- msg:
				}
			}
		}
	}
}

func ReceiveMiniTickerMessage(ctx context.Context, wg *sync.WaitGroup, outChan chan []byte) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		slog.Info("Attempting to connect to miniTicker stream...")
		conn, err := createClientConn(ctx)
		if err != nil {
			slog.Error("Failed to create gRPC client connection for miniTicker", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		client := socket.NewSocketServiceClient(conn)
		stream, err := client.ReceiveRawMiniTicker(ctx, &socket.RawMiniTickerRequest{})
		if err != nil {
			slog.Error("Could not set up miniTicker stream, will retry...", "error", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		slog.Info("Connection to miniTicker stream established")

		err = receiveMessages(ctx, stream, outChan)
		conn.Close()

		if err != nil {
			slog.Warn("miniTicker stream connection broken", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(5 * time.Second)
		}
	}
}

func ReceiveAggTradeMessage(
	ctx context.Context,
	symbol string,
	outChan chan []byte,
) {
	slog.Info("ReceiveAggTradeMessage started", "symbol", symbol)
	defer slog.Info("ReceiveAggTradeMessage ended", "symbol", symbol)

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"ReceiveAggTradeMessage context cancelled",
				"symbol",
				symbol,
				"error",
				ctx.Err(),
			)
			return
		default:
		}

		slog.Info("Attempting to connect to aggTrade stream...", "symbol", symbol)
		conn, err := createClientConn(ctx)
		if err != nil {
			slog.Error(
				"Failed to create gRPC client connection for aggTrade",
				"symbol",
				symbol,
				"error",
				err,
			)
			time.Sleep(5 * time.Second)
			continue
		}

		client := socket.NewSocketServiceClient(conn)
		stream, err := client.ReceiveRawAggTrade(ctx, &socket.RawAggTradeRequest{Symbol: symbol})
		if err != nil {
			slog.Error(
				"Could not set up aggTrade stream, will retry...",
				"symbol",
				symbol,
				"error",
				err,
			)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		slog.Info("Connection to aggTrade stream established", "symbol", symbol)

		err = receiveMessages(ctx, stream, outChan)
		conn.Close()

		if err != nil {
			slog.Warn("aggTrade stream connection broken", "symbol", symbol, "error", err)
		} else {
			slog.Info("aggTrade stream ended normally", "symbol", symbol)
		}

		select {
		case <-ctx.Done():
			slog.Info("ReceiveAggTradeMessage context cancelled after stream end", "symbol", symbol)
			return
		default:
			slog.Info("Will retry connection in 5 seconds", "symbol", symbol)
			time.Sleep(5 * time.Second)
		}
	}
}

func createClientConn(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func receiveMessages(ctx context.Context, stream StreamReceiver, outChan chan<- []byte) error {
	slog.Info("📞 Starting to receive messages from gRPC stream")
	messageCount := 0
	droppedCount := 0
	lastLogTime := time.Now()
	
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				slog.Info("gRPC stream closed by server", "messages_received", messageCount, "dropped", droppedCount)
				return nil
			}
			slog.Error("Got error from gRPC stream during receiving",
				"error", err,
				"messages_received", messageCount,
				"dropped", droppedCount)
			return err
		}
		messageCount++
		if messageCount%1000 == 0 {
			slog.Debug("Received messages from gRPC stream", "count", messageCount, "dropped", droppedCount)
		}
		select {
		case <-ctx.Done():
			slog.Info(
				"Context cancelled in receiveMessages",
				"messages_received",
				messageCount,
				"dropped",
				droppedCount,
				"error",
				ctx.Err(),
			)
			return ctx.Err()
		case outChan <- resp.Data:
			// Message successfully sent to channel
		default:
			// Channel is full, drop message but log only periodically to avoid log spam
			droppedCount++
			now := time.Now()
			if now.Sub(lastLogTime) >= 5*time.Second {
				dropRate := float64(droppedCount) / float64(messageCount) * 100
				slog.Warn("Output channel is full, dropping messages",
					"channel_size", len(outChan),
					"channel_capacity", cap(outChan),
					"messages_received", messageCount,
					"total_dropped", droppedCount,
					"drop_rate_percent", dropRate)
				lastLogTime = now
			}
		}
	}
}
