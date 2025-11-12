package strman

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Wladim1r/aggregator/gateway/converting"
)

type streamManager struct {
	streams map[string]context.CancelFunc
	mu      sync.Mutex
}

func NewStreamManager() *streamManager {
	return &streamManager{
		streams: make(map[string]context.CancelFunc),
	}
}

func (sm *streamManager) Start(
	ctxParent context.Context,
	wg *sync.WaitGroup,
	symbol string,
	outChan chan []byte,
) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.streams[symbol]; exists {
		slog.Info("Stream already exists", "symbol", symbol)
		return false
	}

	ctx, cancel := context.WithCancel(ctxParent)

	sm.streams[symbol] = cancel

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			sm.mu.Lock()
			delete(sm.streams, symbol)
			sm.mu.Unlock()
		}()
		converting.ReceiveAggTradeMessage(ctx, symbol, outChan)
	}()

	return true
}

func (sm *streamManager) Stop(symbol string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cancel, exists := sm.streams[symbol]
	if exists {
		cancel()
		delete(sm.streams, symbol)
		slog.Info("Stream deleted", "symbol", symbol)
	}
}
