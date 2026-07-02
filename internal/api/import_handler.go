package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"go-backend/internal/cache"
	"go-backend/internal/model"
	"go-backend/internal/service"
)

func parseMoviesCSV(r io.Reader) ([]model.Movie, []string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = 6

	if _, err := reader.Read(); err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV header row: %s", err)
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CSV: %s", err)
	}
	if len(records) == 0 {
		return nil, nil, errors.New("CSV contains no data rows")
	}

	movies := make([]model.Movie, 0, len(records))
	var rowErrors []string

	for i, rec := range records {
		m, err := parseMovieRecord(rec)
		if err != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("row %d: %s", i+2, err))
			continue
		}
		movies = append(movies, m)
	}

	return movies, rowErrors, nil
}

func parseMovieRecord(rec []string) (model.Movie, error) {
	title := strings.TrimSpace(rec[0])
	if title == "" {
		return model.Movie{}, errors.New("title is required")
	}

	directorID, err := strconv.ParseInt(strings.TrimSpace(rec[1]), 10, 64)
	if err != nil || directorID < 1 {
		return model.Movie{}, errors.New("invalid director_id")
	}

	releaseYear, err := strconv.Atoi(strings.TrimSpace(rec[2]))
	if err != nil {
		return model.Movie{}, errors.New("invalid release_year")
	}

	m := model.Movie{
		Title:       title,
		DirectorID:  directorID,
		ReleaseYear: releaseYear,
	}

	if v := strings.TrimSpace(rec[3]); v != "" {
		rt, err := strconv.Atoi(v)
		if err != nil || rt < 1 {
			return model.Movie{}, errors.New("invalid runtime_minutes")
		}
		m.RuntimeMinutes = &rt
	}

	if v := strings.TrimSpace(rec[4]); v != "" {
		m.Genre = &v
	}

	if v := strings.TrimSpace(rec[5]); v != "" {
		rating, err := strconv.ParseFloat(v, 64)
		if err != nil || rating < 0 || rating > 10 {
			return model.Movie{}, errors.New("invalid rating (must be 0-10)")
		}
		m.Rating = &rating
	}

	return m, nil
}

const maxCSVBytes = 5 << 20 // 5 MiB

func ImportMovieHandler(svc service.JobService, idemCache cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idempotencyKey := r.Header.Get("Idempotency-Key")
		if idempotencyKey == "" {
			respondError(w, http.StatusBadRequest, "Idempotency-Key header is required")
			return
		}

		if served := serveFromCache(w, r, idemCache, idempotencyKey); served {
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxCSVBytes)

		if err := r.ParseMultipartForm(maxCSVBytes); err != nil {
			respondError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			respondError(w, http.StatusBadRequest, "missing file field")
			return
		}
		defer file.Close()

		movies, rowErrors, err := parseMoviesCSV(file)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		if len(rowErrors) > 0 {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":   "CSV validation failed",
				"details": rowErrors,
			})
			return
		}

		resp, err := svc.AddJob(model.JobTypeMovieImport, &model.MovieImportPayload{Movies: movies})
		if mapServiceError(w, err) {
			return
		}

		storeInCache(r, idemCache, idempotencyKey, resp)
		respondJSON(w, http.StatusAccepted, resp)
	}
}

func serveFromCache(w http.ResponseWriter, r *http.Request, c cache.Cache, key string) bool {
	cached, err := c.Get(r.Context(), key)
	if err == nil {
		respondRawJSON(w, http.StatusAccepted, cached)
		return true
	}
	if !errors.Is(err, cache.ErrMiss) {
		slog.Warn("idempotency cache error, proceeding without cache", "error", err)
	}
	return false
}

func storeInCache(r *http.Request, c cache.Cache, key string, resp model.JobRespond) {
	respJSON, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal idempotency response", "key", key, "error", err)
		return
	}
	if err := c.Set(r.Context(), key, string(respJSON)); err != nil {
		slog.Warn("failed to store idempotency key", "key", key, "error", err)
	}
}
