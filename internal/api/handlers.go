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
	"go-backend/internal/service"

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

func CreateMovieHandler(svc service.MovieService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req struct {
			Title          string   `json:"title"`
			DirectorID     int64    `json:"director_id"`
			ReleaseYear    int      `json:"release_year"`
			RuntimeMinutes *int     `json:"runtime_minutes,omitempty"`
			Genre          *string  `json:"genre,omitempty"`
			Rating         *float64 `json:"rating,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		movie, err := svc.CreateMovie(r.Context(), model.Movie{
			Title:          req.Title,
			DirectorID:     req.DirectorID,
			ReleaseYear:    req.ReleaseYear,
			RuntimeMinutes: req.RuntimeMinutes,
			Genre:          req.Genre,
			Rating:         req.Rating,
		})
		if mapServiceError(w, err) {
			return
		}

		respondJSON(w, http.StatusCreated, movie)
	}
}

func GetMovieHandler(svc service.MovieService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid movie ID")
			return
		}

		movie, err := svc.GetMovie(r.Context(), id)
		if mapServiceError(w, err) {
			return
		}

		respondJSON(w, http.StatusOK, movie)
	}
}

func GetAllMoviesHandler(svc service.MovieService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		movies, err := svc.GetAllMovies(r.Context())
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, movies)
	}
}

func UpdateMovieHandler(svc service.MovieService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid movie ID")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		var req model.MoviePatch
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		updated, err := svc.UpdateMovie(r.Context(), id, req)
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, updated)
	}
}

func DeleteMovieHandler(svc service.MovieService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid movie ID")
			return
		}

		if err := svc.DeleteMovie(r.Context(), id); mapServiceError(w, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
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

func SignUpHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Name     string `json:"name" validate:"required"`
			Password string `json:"password" validate:"required,min=8,max=72"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if err := validate.Struct(req); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		user, err := svc.SignUp(r.Context(), model.User{
			Email:    req.Email,
			Name:     req.Name,
			Password: req.Password,
		})

		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusCreated, user)
	}
}

func LogInHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required,min=8,max=72"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if err := validate.Struct(req); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		tokenString, err := svc.LogIn(r.Context(), req.Email, req.Password)
		if mapServiceError(w, err) {
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   3600,
		})
		respondJSON(w, http.StatusOK, map[string]string{"message": "logged in"})
	}
}

func GetUserHandler(svc service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user ID")
			return
		}

		user, err := svc.GetUser(r.Context(), id)
		if mapServiceError(w, err) {
			return
		}
		respondJSON(w, http.StatusOK, user)
	}
}
