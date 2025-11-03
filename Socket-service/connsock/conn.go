package connsock

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type socketProducer struct {
	outputChan chan<- []byte

	// connection data
	urlConnection string
	readMsgError  chan error
	conn          *websocket.Conn
	mu            sync.RWMutex

	// ctx and wg for current goroutine readMessage
	readMsgCancel context.CancelFunc
	readMsgWg     sync.WaitGroup

	// flag that we take place in reconnection
	reconnecting bool
	reconnectMu  sync.Mutex
}

func NewSocketProduecer(outChan chan []byte, url string, msgChan chan error) *socketProducer {
	return &socketProducer{
		outputChan:    outChan,
		urlConnection: url,
		readMsgError:  msgChan,
	}
}

func (sp *socketProducer) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	reconnectTicker := time.NewTicker(23 * time.Hour)
	defer reconnectTicker.Stop()

	conn, _, err := websocket.DefaultDialer.Dial(sp.urlConnection, nil)
	if err != nil || conn == nil {
		slog.Error(
			"❌ Could not connect to binance",
			"connectiin",
			conn,
			"error",
			err,
			"URL",
			sp.urlConnection,
		)
		conn, err := sp.reconnect(ctx)
		if err != nil || conn == nil {
			return
		}
	}

	sp.mu.Lock()
	sp.conn = conn
	sp.mu.Unlock()

	defer func() {
		// stop goroutine readMessage before close connection
		if sp.readMsgCancel != nil {
			sp.readMsgCancel()
		}
		sp.readMsgWg.Wait()

		sp.mu.Lock()
		if sp.conn != nil {
			sp.conn.Close()
		}
		sp.mu.Unlock()
	}()

	sp.sendPong()
	sp.startReadMessage(ctx, conn)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Got interrupting singal, stop receiving binance api")
			return
		case <-reconnectTicker.C:
			sp.doReconnect(ctx)

		case err := <-sp.readMsgError:
			sp.reconnectMu.Lock()
			isReconnecting := sp.reconnecting
			sp.reconnectMu.Unlock()

			if isReconnecting {
				slog.Info("Ignoring error from readMessage during reconnection", "error", err)
				continue
			}

			slog.Warn("⚠️ Connection was broken, try to connect again", "error", err)
			sp.doReconnect(ctx)
		}
	}
}

func (sp *socketProducer) startReadMessage(parentCtx context.Context, conn *websocket.Conn) {
	readCtx, cancel := context.WithCancel(parentCtx)

	sp.mu.Lock()
	sp.readMsgCancel = cancel
	sp.mu.Unlock()

	sp.readMsgWg.Add(1)

	go sp.readMessage(readCtx, conn)
}

func (sp *socketProducer) stopReadMessage() {
	sp.mu.Lock()
	cancel := sp.readMsgCancel
	sp.mu.Unlock()

	if cancel != nil {
		cancel()
		sp.readMsgWg.Wait()

		slog.Info("Old goroutine readMessage stopped")
	}
}

func (sp *socketProducer) doReconnect(ctx context.Context) {
	sp.reconnectMu.Lock()
	sp.reconnecting = true
	sp.reconnectMu.Unlock()

	defer func() {
		sp.reconnectMu.Lock()
		sp.reconnecting = false
		sp.reconnectMu.Unlock()
	}()

	sp.stopReadMessage()

	sp.mu.Lock()
	if sp.conn != nil {
		sp.conn.Close()
		sp.conn = nil
	}
	sp.mu.Unlock()

	conn, err := sp.reconnect(ctx)
	if err != nil || conn == nil {
		slog.Error("Failed to reconnect")
		return
	}

	sp.mu.Lock()
	sp.conn = conn
	sp.mu.Unlock()

	sp.sendPong()
	sp.startReadMessage(ctx, conn)

	slog.Info("✅ Reconnection completed successfully")
}

func (sp *socketProducer) readMessage(ctx context.Context, conn *websocket.Conn) {
	slog.Info("📖 Entered function 'readMessage'")

	defer sp.readMsgWg.Done()

	if sp.conn == nil {
		slog.Error("'readMessage' got nil connection")
		return
	}

	go func() {
		<-ctx.Done()
		conn.SetReadDeadline(time.Now())
	}()

	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			select {
			case <-ctx.Done():
				slog.Info("'readMessage' stopped due to context cancellation ")
				return
			default:
				sp.readMsgError <- err
				return
			}
		}

		select {
		case <-ctx.Done():
			slog.Info("'readMessage' stopped due to context cancellation ")
			return
		default:
			sp.outputChan <- msg
		}
	}
}

func (sp *socketProducer) reconnect(ctx context.Context) (*websocket.Conn, error) {
	slog.Info("🔄 Entered function 'reconnect'")

	var conn *websocket.Conn
	var err error

	const maxRetries = 5
	var attempt int

	for ; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			slog.Info("Got interrupting singal during reconnect")
			return nil, ctx.Err()
		default:
			delay := 100 * time.Millisecond
			maxDelay := 5 * time.Second
			gen := rand.New(rand.NewSource(time.Now().UnixMicro()))

			conn, _, err = websocket.DefaultDialer.Dial(sp.urlConnection, nil)

			if err == nil && conn != nil {
				return conn, nil
			}

			backoffTime := delay * time.Duration(math.Pow(2, float64(attempt)))
			backoffTime = min(backoffTime, maxDelay)

			jitter := time.Duration(gen.Int63n(int64(backoffTime)))

			time.Sleep(jitter)
		}
	}

	slog.Error("❌ Could not reconnect to binance api after all the retries")
	return nil, err
}

func (sp *socketProducer) sendPong() {
	slog.Info("🎾 Entered function 'sendPong'")

	sp.mu.RLock()
	conn := sp.conn
	sp.mu.RUnlock()

	if conn == nil {
		slog.Error("'sendPong' got nil connection")
		return
	}

	conn.SetPingHandler(func(appData string) error {
		sp.mu.RLock()
		curConn := sp.conn
		sp.mu.RUnlock()

		err := curConn.WriteControl(
			websocket.PongMessage,
			[]byte(appData),
			time.Now().Add(3*time.Second),
		)

		if err != nil {
			slog.Error("❌ Could not send Pong to binance", "error", err)
			return err
		}

		slog.Info("✅ Send Pong to binance successfuly")
		return nil
	})
}
