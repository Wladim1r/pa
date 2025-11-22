// Package reddis
package reddis

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Wladim1r/redboard/models"
	"github.com/redis/go-redis/v9"
)

type rDB struct {
	rdb *redis.Client
}

func NewClient() *rDB {
	return &rDB{
		rdb: redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
			Password: "",
			DB:       0,
		}),
	}
}

func (r *rDB) Subscribe(ctx context.Context, outChan chan models.SecondStat) {
	subscriber := r.rdb.Subscribe(ctx, "stream")

	secondStat := new(models.SecondStat)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := subscriber.ReceiveMessage(ctx)
			if err != nil {
				slog.Error("Could not receive message from redis stream channel", "error", err)
				continue
			}

			if err := json.Unmarshal([]byte(msg.Payload), secondStat); err != nil {
				slog.Error("Failed to parsing bytes into SecondStat struct", "error", err)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case outChan <- *secondStat:
				slog.Info("Msg from redis stream has just send to channel")
			}
		}
	}
}
