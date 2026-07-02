package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/internal/service"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

const maxRequestBodyBytes = 1 << 20 // 1 MiB

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func respondRawJSON(w http.ResponseWriter, status int, raw string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprint(w, raw)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func mapServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrNotFound) {
		slog.Debug("not found", "error", err)
		respondError(w, http.StatusNotFound, "not found")
		return true
	}
	if errors.Is(err, service.ErrValidation) {
		slog.Debug("validation error", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return true
	}
	if errors.Is(err, service.ErrConflict) {
		slog.Debug("conflict", "error", err)
		respondError(w, http.StatusConflict, err.Error())
		return true
	}
	if errors.Is(err, service.ErrUnauthorized) {
		slog.Debug("unauthorized", "error", err)
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return true
	}
	if errors.Is(err, service.ErrUnavailable) {
		slog.Warn("service unavailable", "error", err)
		respondError(w, http.StatusServiceUnavailable, err.Error())
		return true
	}
	if errors.Is(err, context.Canceled) {
		// Client disconnected; nothing useful to send.
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		slog.Warn("request timed out", "error", err)
		respondError(w, http.StatusGatewayTimeout, "request timed out")
		return true
	}
	slog.Error("service error", "error", err)
	respondError(w, http.StatusInternalServerError, "internal server error")
	return true
}

func parseID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %w", err)
	}
	if n < 1 {
		return 0, errors.New("invalid id: must be positive")
	}
	return n, nil
}
