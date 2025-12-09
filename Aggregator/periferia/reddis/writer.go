package reddis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Wladim1r/aggregator/gateway/strman"
	"github.com/Wladim1r/aggregator/models"
	"github.com/redis/go-redis/v9"
)

type saver struct {
	rdb *redis.Client
	cfg redisConfig
	sm  *strman.StreamManager
}

func NewSaver(cfg redisConfig, sm *strman.StreamManager) *saver {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DBnum,
	})

	return &saver{
		rdb: rdb,
		cfg: cfg,
		sm:  sm,
	}
}

func (s *saver) setPing(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}

func (s *saver) saveSecondStat(ctx context.Context, msg models.SecondStat) error {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Could not parse SecondStat struct into []bytes", "error", err)
		return err
	}

	cmd := s.rdb.Publish(ctx, "stream", data)
	if cmd.Err() != nil {
		slog.Error("Could not sent msg to Redis", "error", err)
		return err
	}

	return nil
}

func (s *saver) Start(ctx context.Context, wg *sync.WaitGroup, inChan chan models.SecondStat) {
	defer wg.Done()
	defer s.rdb.Close()

	if err := s.setPing(ctx); err != nil {
		slog.Error("Could not set up Ping Handler", "error", err)
		return
	}

	slog.Info("✍️ Starting Redis writer")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Got interruption signal, stopping Redis writer")
			return
		case stat, ok := <-inChan:
			if !ok {
				slog.Info("Input channel closed, stopping Redis writer")
				return
			}

			fmt.Println(stat.Symbol)
			fmt.Println(s.sm.Followers)
			followers := s.sm.GetFollowers(stat.Symbol)

			for _, id := range followers {
				stat.UserID = id
				if err := s.saveSecondStat(ctx, stat); err != nil {
					slog.Error("Failed to save SecondStat to Redis",
						"symbol", stat.Symbol,
						"userID", stat.UserID,
						"error", err,
					)
					continue
				}
			}

			slog.Debug(
				"Saved to Redis for followers",
				"symbol",
				stat.Symbol,
				"count",
				len(followers),
			)
		}
	}
}
