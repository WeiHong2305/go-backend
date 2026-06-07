package store

import (
	"context"
	"fmt"
	"go-backend/internal/model"
	"sync"
	"time"
)

type MovieRepository interface {
	Save(context.Context, model.Movie) (int64, error)
	Get(context.Context, int64) (model.Movie, error)
	GetAll(context.Context) ([]model.Movie, error)
	Update(context.Context, int64, model.Movie) (model.Movie, error)
	Delete(context.Context, int64) error
}

type MemoryMovieStore struct {
	mu     sync.RWMutex
	movies map[int64]model.Movie
}

func NewMemoryMovieStore() *MemoryMovieStore {
	return &MemoryMovieStore{movies: make(map[int64]model.Movie)}
}

func (s *MemoryMovieStore) Save(_ context.Context, m model.Movie) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := int64(len(s.movies) + 1)
	m.ID = id
	s.movies[id] = m

	return id, nil
}

func (s *MemoryMovieStore) Get(_ context.Context, id int64) (model.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	movie, exists := s.movies[id]
	if !exists {
		return model.Movie{}, fmt.Errorf("movie with ID %d not found", id)
	}
	return movie, nil
}

func (s *MemoryMovieStore) GetAll(_ context.Context) ([]model.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	movies := make([]model.Movie, 0, len(s.movies))
	for _, movie := range s.movies {
		movies = append(movies, movie)
	}
	return movies, nil
}

func (s *MemoryMovieStore) Update(_ context.Context, id int64, m model.Movie) (model.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.movies[id]
	if !exists {
		return model.Movie{}, fmt.Errorf("movie with ID %d not found", id)
	}
	m.ID = id
	m.CreatedAt = existing.CreatedAt
	m.UpdatedAt = time.Now()
	s.movies[id] = m
	return m, nil
}

func (s *MemoryMovieStore) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.movies[id]; !exists {
		return fmt.Errorf("movie with ID %d not found", id)
	}
	delete(s.movies, id)
	return nil
}
