package api

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World! You requested: %s", r.URL.Path)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ReadyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			slog.ErrorContext(r.Context(), "readiness check: database ping failed", "error", err)
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
