// Package db
package db

import (
	"log/slog"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func MustLoad() *gorm.DB {
	dsn := "host=postgres user=postgres password=postgres dbname=users port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		slog.Error("Could not connect to db", "error", err.Error())
		os.Exit(1)
	}

	return db
}
