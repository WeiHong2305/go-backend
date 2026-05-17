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

	"go-backend/internal/model"
	"go-backend/internal/store"
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

// mapStoreError translates store/DB errors into HTTP responses.
// Returns true if a response was written.
func mapStoreError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "user not found")
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
	slog.Error("store error", "error", err)
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

func CreateUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		if req.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required")
			return
		}

		user := model.User{Name: req.Name}
		if req.Active != nil {
			user.Active = *req.Active
		} else {
			user.Active = true
		}

		id, err := userStore.Save(r.Context(), user)
		if err != nil {
			mapStoreError(w, err)
			return
		}

		user.ID = id
		respondJSON(w, http.StatusCreated, user)
	}
}

func GetUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idInt, err := parseUserID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		user, err := userStore.Get(r.Context(), idInt)
		if mapStoreError(w, err) {
			return
		}

		respondJSON(w, http.StatusOK, user)
	}
}

func GetAllUsersHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := userStore.GetAll(r.Context())
		if mapStoreError(w, err) {
			return
		}
		if users == nil {
			users = []model.User{}
		}
		respondJSON(w, http.StatusOK, users)
	}
}

func UpdateUserHandler(userStore store.UserStore) http.HandlerFunc {
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

		if req.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required")
			return
		}

		patch := model.User{Name: req.Name}
		if req.Active != nil {
			patch.Active = *req.Active
		} else {
			patch.Active = true
		}

		updated, err := userStore.Update(r.Context(), idInt, patch)
		if mapStoreError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, updated)
	}
}

func DeleteUserHandler(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idInt, err := parseUserID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		if err := userStore.Delete(r.Context(), idInt); mapStoreError(w, err) {
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
