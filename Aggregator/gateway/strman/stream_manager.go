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

	// Check if user is already subscribed
	if slices.Contains(sm.Followers[symbol], userID) {
		slog.Info(
			"User already subscribed to coin, allowing re-subscription",
			"symbol",
			symbol,
			"userID",
			userID,
		)
		// Allow re-subscription - user might have disconnected and reconnected
		return true
	}

	existingFollowers := len(sm.Followers[symbol])
	sm.Followers[symbol] = append(sm.Followers[symbol], userID)

	// Check if stream exists and is active
	_, streamExists := sm.streams[symbol]

	if !streamExists {
		// Stream doesn't exist - need to start it
		if existingFollowers == 0 {
			slog.Info("First subscriber, starting new stream", "symbol", symbol, "userID", userID)
		} else {
			slog.Info("Stream was stopped, restarting for existing followers",
				"symbol", symbol,
				"userID", userID,
				"total_followers", len(sm.Followers[symbol]))
		}
		sm.start(ctxParent, wg, symbol, outChan)
		slog.Info("Stream started", "symbol", symbol)
	} else {
		// Stream already exists - just add subscriber
		slog.Info("Adding subscriber to existing stream",
			"symbol", symbol,
			"userID", userID,
			"existing_followers", existingFollowers,
			"total_followers", len(sm.Followers[symbol]),
			"stream_exists", true)
	}

	slog.Info(
		"User added to coin",
		"symbol",
		symbol,
		"userID",
		userID,
		"total_followers",
		len(sm.Followers[symbol]),
	)
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
			// Only clean up if there are no followers left
			// This prevents race condition where stream restarts but followers list is cleared
			if len(sm.Followers[symbol]) == 0 {
				delete(sm.streams, symbol)
				delete(sm.Followers, symbol)
				slog.Info(
					"Stream processing stopped and cleaned up (no followers)",
					"symbol",
					symbol,
				)
			} else {
				// Stream stopped but there are still followers - remove cancel func but keep followers
				// This allows stream to restart if needed
				delete(sm.streams, symbol)
				slog.Warn("Stream processing stopped but followers remain, will restart on next message",
					"symbol", symbol,
					"followers", sm.Followers[symbol])
			}
			sm.mu.Unlock()
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
			slog.Info("Stream cancellation signal sent", "symbol", symbol)
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
