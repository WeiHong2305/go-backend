package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"
)

func newDatabase() *sql.DB {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:secret@localhost:5433/gopgtest?sslmode=disable"
		slog.Warn("DATABASE_URL not set, using default local connection string")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	return db
}
