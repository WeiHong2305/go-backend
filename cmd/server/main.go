package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-backend/internal/api"
	"go-backend/internal/store"

	_ "github.com/lib/pq"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

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
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err = db.PingContext(ctx); err != nil {
		cancel()
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	cancel()

	if err := createUsersTable(db); err != nil {
		slog.Error("failed to create users table", "error", err)
		os.Exit(1)
	}

	users := store.NewPgUserStore(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/", api.RootHandler)
	mux.HandleFunc("/health", api.HealthHandler(db))
	mux.HandleFunc("POST /users", api.CreateUserHandler(users))
	mux.HandleFunc("GET /users/{id}", api.GetUserHandler(users))
	mux.HandleFunc("GET /users", api.GetAllUsersHandler(users))
	mux.HandleFunc("PUT /users/{id}", api.UpdateUserHandler(users))
	mux.HandleFunc("DELETE /users/{id}", api.DeleteUserHandler(users))

	handler := api.RequestLog(api.Recover(mux))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listen failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server exited")
}

func createUsersTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name varchar(100) NOT NULL,
		active boolean DEFAULT true,
		created_at timestamp DEFAULT NOW(),
		updated_at timestamp DEFAULT NOW()
	)`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("create users table: %w", err)
	}
	return nil
}
