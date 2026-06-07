package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-backend/internal/service"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func mapServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "user not found")
		return true
	}
	if errors.Is(err, service.ErrValidation) {
		respondError(w, http.StatusBadRequest, err.Error())
		return true
	}
	if errors.Is(err, context.Canceled) {
		// Client disconnected; nothing useful to send.
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		respondError(w, http.StatusGatewayTimeout, "request timed out")
		return true
	}
	slog.Error("service error", "error", err)
	respondError(w, http.StatusInternalServerError, "internal server error")
	return true
}

func RootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World! You requested: %s", r.URL.Path)
}

func HealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			slog.Error("health check: database ping failed", "error", err)
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func createMovieHandler(svc service.MovieService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req struct {
			Title       string `json:"title"`
			Director    string `json:"director"`
			ReleaseYear int    `json:"release_year"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		movie, err := svc.CreateMovie(r.Context(), req.Title, req.Director, req.ReleaseYear)
		if mapServiceError(w, err) {
			return
		}

		respondJSON(w, http.StatusCreated, movie)
	}
}

func GetUserHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idInt, err := parseUserID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		user, err := svc.GetUser(r.Context(), idInt)
		if mapServiceError(w, err) {
			return
		}

		respondJSON(w, http.StatusOK, user)
	}
}

func GetAllUsersHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := svc.GetAllUsers(r.Context())
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, users)
	}
}

func UpdateUserHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idInt, err := parseUserID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		var req struct {
			Name   string `json:"name"`
			Active *bool  `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		updated, err := svc.UpdateUser(r.Context(), idInt, req.Name, req.Active)
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, updated)
	}
}

func DeleteUserHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idInt, err := parseUserID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		if err := svc.DeleteUser(r.Context(), idInt); mapServiceError(w, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func parseUserID(id string) (int, error) {
	n, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %w", err)
	}
	if n < 1 {
		return 0, errors.New("invalid id: must be positive")
	}
	return n, nil
}
