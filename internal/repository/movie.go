package repository

import (
	"context"
	"database/sql"
	"errors"
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

type MemoryMovieRepository struct {
	mu     sync.RWMutex
	movies map[int64]model.Movie
}

type PgMovieRepository struct {
	db *sql.DB
}

func NewMemoryMovieRepository() *MemoryMovieRepository {
	return &MemoryMovieRepository{movies: make(map[int64]model.Movie)}
}

func NewPgMovieRepository(db *sql.DB) *PgMovieRepository {
	return &PgMovieRepository{db: db}
}

func (s *MemoryMovieRepository) Save(_ context.Context, m model.Movie) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := int64(len(s.movies) + 1)
	m.ID = id
	s.movies[id] = m

	return id, nil
}

func (s *PgMovieRepository) Save(ctx context.Context, m model.Movie) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `INSERT INTO movies (title, director, release_year, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id`

	var id int64
	err := s.db.QueryRowContext(ctx, query, m.Title, m.Director, m.ReleaseYear).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to save movie: %w", err)
	}
	return id, nil
}

func (s *MemoryMovieRepository) Get(_ context.Context, id int64) (model.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	movie, exists := s.movies[id]
	if !exists {
		return model.Movie{}, fmt.Errorf("movie %d: %w", id, ErrNotFound)
	}
	return movie, nil
}

func (s *PgMovieRepository) Get(ctx context.Context, id int64) (model.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, title, director, release_year, created_at, updated_at FROM movies WHERE id = $1`
	var m model.Movie

	err := s.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.Title, &m.Director, &m.ReleaseYear, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Movie{}, fmt.Errorf("movie %d: %w", id, ErrNotFound)
		}
		return model.Movie{}, fmt.Errorf("failed to get movie %d: %w", id, err)
	}
	return m, nil
}

func (s *MemoryMovieRepository) GetAll(_ context.Context) ([]model.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	movies := make([]model.Movie, 0, len(s.movies))
	for _, movie := range s.movies {
		movies = append(movies, movie)
	}
	return movies, nil
}

func (s *PgMovieRepository) GetAll(ctx context.Context) ([]model.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `SELECT id, title, director, release_year, created_at, updated_at FROM movies`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all movies: %w", err)
	}
	defer rows.Close()

	movies := make([]model.Movie, 0)
	for rows.Next() {
		var m model.Movie
		if err := rows.Scan(&m.ID, &m.Title, &m.Director, &m.ReleaseYear, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("movie iteration error: %w", err)
	}

	return movies, nil
}

func (s *MemoryMovieRepository) Update(_ context.Context, id int64, m model.Movie) (model.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.movies[id]
	if !exists {
		return model.Movie{}, fmt.Errorf("movie %d: %w", id, ErrNotFound)
	}
	m.ID = id
	m.CreatedAt = existing.CreatedAt
	m.UpdatedAt = time.Now()
	s.movies[id] = m
	return m, nil
}

func (s *PgMovieRepository) Update(ctx context.Context, id int64, m model.Movie) (model.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `UPDATE movies SET title = $1, director = $2, release_year = $3, updated_at = NOW() 
		WHERE id = $4
		RETURNING created_at`
	updated := m

	err := s.db.QueryRowContext(ctx, query, m.Title, m.Director, m.ReleaseYear, id).Scan(&updated.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Movie{}, fmt.Errorf("movie %d: %w", id, ErrNotFound)
		}
		return model.Movie{}, fmt.Errorf("failed to update movie %d: %w", id, err)
	}
	return updated, nil
}

func (s *MemoryMovieRepository) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.movies[id]; !exists {
		return fmt.Errorf("movie %d: %w", id, ErrNotFound)
	}
	delete(s.movies, id)
	return nil
}

func (s *PgMovieRepository) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `DELETE FROM movies WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete movie %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie %d: %w", id, ErrNotFound)
	}

	return nil
}
