package strman

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/Wladim1r/aggregator/gateway/converting"
)

type StreamManager struct {
	streams   map[string]context.CancelFunc
	Followers map[string][]int
	mu        sync.RWMutex
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams:   make(map[string]context.CancelFunc),
		Followers: make(map[string][]int),
	}
}

func (sm *StreamManager) AddCoin(
	ctxParent context.Context,
	wg *sync.WaitGroup,
	symbol string,
	userID int,
	outChan chan []byte,
) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	symbol = strings.ToLower(symbol)

	if slices.Contains(sm.Followers[symbol], userID) {
		return false
	}

	if len(sm.Followers[symbol]) == 0 {
		sm.start(ctxParent, wg, symbol, outChan)
		slog.Info("Stream started", "symbol", symbol)
	}

	sm.Followers[symbol] = append(sm.Followers[symbol], userID)
	slog.Info("User added to coin", "symbol", symbol, "userID", userID)
	fmt.Println("StreamManager state after AddCoin:", sm.Followers)
	return true
}

func (sm *StreamManager) start(
	ctxParent context.Context,
	wg *sync.WaitGroup,
	symbol string,
	outChan chan []byte,
) {
	ctx, cancel := context.WithCancel(ctxParent)
	sm.streams[symbol] = cancel

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			sm.mu.Lock()
			delete(sm.streams, symbol)
			delete(sm.Followers, symbol)
			sm.mu.Unlock()
			slog.Warn("Stream processing stopped and cleaned up", "symbol", symbol)
		}()
		converting.ReceiveAggTradeMessage(ctx, symbol, outChan)
	}()
}

func (sm *StreamManager) DeleteCoin(symbol string, userID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	symbol = strings.ToLower(symbol)

	userList, ok := sm.Followers[symbol]
	if !ok {
		return
	}

	for i, id := range userList {
		if id == userID {
			sm.Followers[symbol] = append(userList[:i], userList[i+1:]...)
			slog.Info("User removed from coin", "symbol", symbol, "userID", userID)
			break
		}
	}

	if len(sm.Followers[symbol]) == 0 {
		if cancel, exists := sm.streams[symbol]; exists {
			cancel()
			delete(sm.streams, symbol)
			delete(sm.Followers, symbol)
			slog.Info("Stream stopped and deleted due to no followers", "symbol", symbol)
		}
	}
}

func (sm *StreamManager) GetFollowers(symbol string) []int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	symbol = strings.ToLower(symbol)
	fmt.Println("StreamManager state before GetFollowers:", sm.Followers, "for symbol:", symbol)

	followersCopy := make([]int, len(sm.Followers[symbol]))
	copy(followersCopy, sm.Followers[symbol])
	return followersCopy
}
