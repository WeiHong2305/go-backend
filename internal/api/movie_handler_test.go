package api

import (
	"context"
	"encoding/json"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/service"
	"go-backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateMovieHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setupSvc   func(*testutil.MockMovieService)
		wantStatus int
	}{
		{
			name: "201 success",
			body: `{"title":"Inception","director_id":1,"release_year":2010}`,
			setupSvc: func(m *testutil.MockMovieService) {
				m.CreateMovieFunc = func(_ context.Context, movie model.Movie) (model.Movie, error) {
					movie.ID = 1
					return movie, nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "400 validation error",
			body: `{"title":"","director_id":1,"release_year":2010}`,
			setupSvc: func(m *testutil.MockMovieService) {
				m.CreateMovieFunc = func(_ context.Context, movie model.Movie) (model.Movie, error) {
					return model.Movie{}, fmt.Errorf("title is required: %w", service.ErrValidation)
				}
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &testutil.MockMovieService{}
			tc.setupSvc(svc)

			r := httptest.NewRequest(http.MethodPost, "/movies", strings.NewReader(tc.body))
			r.Header.Set("Context-Type", "application/json")
			w := httptest.NewRecorder()

			CreateMovieHandler(svc).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestGetMovieHandler(t *testing.T) {
	movie := model.Movie{ID: 1, Title: "Inception", DirectorID: 1, ReleaseYear: 2010}

	tests := []struct {
		name       string
		pathID     string
		setupSvc   func(*testutil.MockMovieService)
		wantStatus int
	}{
		{
			name:   "200 success",
			pathID: "1",
			setupSvc: func(m *testutil.MockMovieService) {
				m.GetMovieFunc = func(_ context.Context, _ int64) (model.Movie, error) {
					return movie, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "404 not found",
			pathID: "99",
			setupSvc: func(m *testutil.MockMovieService) {
				m.GetMovieFunc = func(_ context.Context, _ int64) (model.Movie, error) {
					return model.Movie{}, service.ErrNotFound
				}
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &testutil.MockMovieService{}
			tc.setupSvc(svc)

			r := httptest.NewRequest(http.MethodGet, "/movies/"+tc.pathID, nil)
			r.SetPathValue("id", tc.pathID)
			w := httptest.NewRecorder()

			GetMovieHandler(svc).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusOK {
				var got model.Movie
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if got.ID != movie.ID {
					t.Errorf("got movie ID %d, want %d", got.ID, movie.ID)
				}
			}
		})
	}
}

func TestGetAllMoviesHandler(t *testing.T) {
	movies := []model.Movie{
		{ID: 1, Title: "Inception", DirectorID: 1, ReleaseYear: 2010},
		{ID: 2, Title: "Interstellar", DirectorID: 2, ReleaseYear: 2014},
	}

	svc := &testutil.MockMovieService{
		GetAllMoviesFunc: func(_ context.Context) ([]model.Movie, error) {
			return movies, nil
		},
	}

	r := httptest.NewRequest(http.MethodGet, "/movies", nil)
	w := httptest.NewRecorder()

	GetAllMoviesHandler(svc).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var got []model.Movie
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != len(movies) {
		t.Errorf("got %d movies, want %d", len(got), len(movies))
	}
}

func TestUpdateMovieHandler(t *testing.T) {
	title := "New Title"
	updated := model.Movie{ID: 1, Title: title, DirectorID: 1, ReleaseYear: 2010}

	svc := &testutil.MockMovieService{
		UpdateMovieFunc: func(_ context.Context, _ int64, _ model.MoviePatch) (model.Movie, error) {
			return updated, nil
		},
	}

	body := `{"title":"New Title"}`
	r := httptest.NewRequest(http.MethodPatch, "/movies/1", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	UpdateMovieHandler(svc).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var got model.Movie
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.Title != title {
		t.Errorf("got title: %q, want %q", got.Title, title)
	}
}

func TestDeleteMovieHandler(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		setupSvc   func(*testutil.MockMovieService)
		wantStatus int
	}{
		{
			name:   "204 success",
			pathID: "1",
			setupSvc: func(m *testutil.MockMovieService) {
				m.DeleteMovieFunc = func(_ context.Context, _ int64) error {
					return nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "404 not found",
			pathID: "99",
			setupSvc: func(m *testutil.MockMovieService) {
				m.DeleteMovieFunc = func(_ context.Context, _ int64) error {
					return service.ErrNotFound
				}
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &testutil.MockMovieService{}
			tc.setupSvc(svc)

			r := httptest.NewRequest(http.MethodDelete, "/movies/"+tc.pathID, nil)
			r.SetPathValue("id", tc.pathID)
			w := httptest.NewRecorder()

			DeleteMovieHandler(svc).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, got %d", w.Code, tc.wantStatus)
			}
		})
	}
}
