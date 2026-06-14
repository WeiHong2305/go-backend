package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-backend/internal/api"
	"go-backend/internal/repository"
	"go-backend/internal/service"

	_ "github.com/lib/pq"
)

func main() {
	var loglevel slog.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		loglevel = slog.LevelDebug
	case "warn":
		loglevel = slog.LevelWarn
	case "error":
		loglevel = slog.LevelError
	default:
		loglevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: loglevel})))

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
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err = db.PingContext(ctx); err != nil {
		cancel()
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	cancel()

	movieRepo := repository.NewPgMovieRepository(db)
	movieSvc := service.NewMovieService(movieRepo)

	jwtSecret := os.Getenv("SECRET")
	if jwtSecret == "" {
		slog.Error("SECRET env var is required")
		os.Exit(1)
	}

	userRepo := repository.NewPgUserRepository(db)
	userSvc := service.NewUserService(userRepo, []byte(jwtSecret))

	mux := http.NewServeMux()
	mux.HandleFunc("/", api.RootHandler)
	mux.HandleFunc("/health", api.HealthHandler(db))

	mux.HandleFunc("POST /movies", api.CreateMovieHandler(movieSvc))
	mux.HandleFunc("GET /movies/{id}", api.GetMovieHandler(movieSvc))
	mux.HandleFunc("PATCH /movies/{id}", api.UpdateMovieHandler(movieSvc))
	mux.HandleFunc("DELETE /movies/{id}", api.DeleteMovieHandler(movieSvc))
	mux.HandleFunc("GET /movies", api.GetAllMoviesHandler(movieSvc))

	mux.HandleFunc("POST /signup", api.SignUpHandler(userSvc))
	mux.HandleFunc("POST /login", api.LogInHandler(userSvc))
	mux.HandleFunc("GET /users/{id}", api.GetUserHandler(userSvc))
	mux.Handle("GET /users", api.RequireAdmin(http.HandlerFunc(api.GetAllUsersHandler(userSvc))))

	publicRoutes := map[string]struct{}{
		"GET /":        {},
		"GET /health":  {},
		"POST /signup": {},
		"POST /login":  {},
	}

	handler := api.RequestLog(api.Recover(api.RequireAuth([]byte(jwtSecret), publicRoutes)(mux)))

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
