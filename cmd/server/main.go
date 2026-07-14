package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-backend/internal/api"
	"go-backend/internal/cache"
	"go-backend/internal/logging"
	"go-backend/internal/metrics"
	"go-backend/internal/model"
	"go-backend/internal/repository"
	"go-backend/internal/service"
	"go-backend/internal/tracing"
	"go-backend/internal/worker"
	"go-backend/internal/worker/handlers"

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
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: loglevel})
	slog.SetDefault(slog.New(logging.NewContextHandler(jsonHandler)))

	_, tracingShutdown, err := tracing.New(context.Background())
	if err != nil {
		slog.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	defer tracingShutdown(context.Background())

	m, err := metrics.New()
	if err != nil {
		slog.Error("failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	defer m.ShutDown(context.Background())

	db := newDatabase()
	defer db.Close()

	mq := newRabbitMQ()
	defer mq.Close()
	mq.HandleReconnect()

	redisCfg := newRedisClient()
	defer redisCfg.client.Close()

	movieRepo := repository.NewPgMovieRepository(db)
	movieCache := cache.NewMetricsCache(cache.NewRedisCache(redisCfg.client, redisCfg.cacheTTL), m, "movie")
	idempotencyCache := cache.NewMetricsCache(cache.NewRedisCache(redisCfg.client, 24*time.Hour), m, "idempotency")
	movieSvc := service.NewMovieService(movieRepo, movieCache)

	jwtSecret := os.Getenv("SECRET")
	if jwtSecret == "" {
		slog.Error("SECRET env var is required")
		os.Exit(1)
	}

	jobSvc := service.NewJobService(mq, mq.JobQ.Name)
	pool := worker.NewPool(mq.JobConCh, mq, mq.JobQ.Name, mq.RetryQ.Name, 4, m)
	pool.Register(model.JobTypeMovieImport, handlers.ImportMovies(movieSvc))
	pool.Start()

	userRepo := repository.NewPgUserRepository(db)
	userSvc := service.NewUserService(userRepo, []byte(jwtSecret))

	mux := http.NewServeMux()
	mux.HandleFunc("/", api.RootHandler)
	mux.HandleFunc("/health", api.HealthHandler)
	mux.HandleFunc("/ready", api.ReadyHandler(db))
	mux.Handle("GET /metrics", m.Handler())

	mux.HandleFunc("POST /movies", api.CreateMovieHandler(movieSvc))
	mux.HandleFunc("GET /movies/{id}", api.GetMovieHandler(movieSvc))
	mux.HandleFunc("PATCH /movies/{id}", api.UpdateMovieHandler(movieSvc))
	mux.HandleFunc("DELETE /movies/{id}", api.DeleteMovieHandler(movieSvc))
	mux.HandleFunc("GET /movies", api.GetAllMoviesHandler(movieSvc))
	mux.HandleFunc("POST /movies/import", api.ImportMovieHandler(jobSvc, idempotencyCache))

	mux.HandleFunc("POST /signup", api.SignUpHandler(userSvc))
	mux.HandleFunc("POST /login", api.LogInHandler(userSvc))
	mux.HandleFunc("GET /users/{id}", api.GetUserHandler(userSvc))
	mux.Handle("GET /users", api.RequireAdmin(http.HandlerFunc(api.GetAllUsersHandler(userSvc))))

	publicRoutes := map[string]struct{}{
		"GET /":        {},
		"GET /health":  {},
		"GET /metrics": {},
		"POST /signup": {},
		"POST /login":  {},
	}

	handler := api.RequestID(api.RequestLog(m, api.Recover(api.RequireAuth([]byte(jwtSecret), publicRoutes)(mux))))

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
	slog.Info("shutting down - send signal again to force exit")

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)

		httpCtx, httpCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer httpCancel()
		if err := server.Shutdown(httpCtx); err != nil {
			slog.Error("HTTP server forced shutdown", "error", err)
		}

		if err := pool.Stop(35 * time.Second); err != nil {
			slog.Error("worker pool forced shutdown", "error", err)
		}
	}()

	select {
	case <-shutdownDone:
		slog.Info("server exited cleanly")
	case <-quit:
		slog.Warn("second signal received, forcing exit")
		os.Exit(1)
	}
}
