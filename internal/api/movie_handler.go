package api

import (
	"encoding/json"
	"errors"
	"go-backend/internal/model"
	"go-backend/internal/service"
	"net/http"
)

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
