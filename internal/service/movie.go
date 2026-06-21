package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/internal/cache"
	"go-backend/internal/model"
	"go-backend/internal/repository"
	"log/slog"
)

type MovieService interface {
	CreateMovie(ctx context.Context, movie model.Movie) (model.Movie, error)
	GetMovie(ctx context.Context, id int64) (model.Movie, error)
	GetAllMovies(ctx context.Context) ([]model.Movie, error)
	UpdateMovie(ctx context.Context, id int64, patch model.MoviePatch) (model.Movie, error)
	DeleteMovie(ctx context.Context, id int64) error
}

type movieService struct {
	repo  repository.MovieRepository
	cache cache.Cache
}

func NewMovieService(repo repository.MovieRepository, cache cache.Cache) *movieService {
	return &movieService{repo: repo, cache: cache}
}

func (s *movieService) CreateMovie(ctx context.Context, movie model.Movie) (model.Movie, error) {
	if movie.Title == "" {
		return model.Movie{}, fmt.Errorf("title is required: %w", ErrValidation)
	}

	id, err := s.repo.Save(ctx, movie)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
	}

	movie.ID = id
	return movie, nil
}

func (s *movieService) GetMovie(ctx context.Context, id int64) (model.Movie, error) {
	cacheKey := fmt.Sprintf("movies:%d", id)

	cacheValue, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		var movie model.Movie
		if err := json.Unmarshal([]byte(cacheValue), &movie); err == nil {
			slog.Debug("cache hit", "key", cacheKey)
			return movie, nil
		}
	} else if !errors.Is(err, cache.ErrMiss) {
		slog.Warn("failed to read from cache", "key", cacheKey, "error", err)
	}

	slog.Info("Cache Miss")
	movie, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
	}

	data, err := json.Marshal(movie)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(data)); err != nil {
			slog.Warn("failed to write to cache", "key", cacheKey, "error", err)
		}
	}

	return movie, nil
}

func (s *movieService) GetAllMovies(ctx context.Context) ([]model.Movie, error) {
	movies, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, mapRepoError(err)
	}
	if movies == nil {
		movies = []model.Movie{}
	}
	return movies, nil
}

func (s *movieService) UpdateMovie(ctx context.Context, id int64, patch model.MoviePatch) (model.Movie, error) {
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
	}

	if patch.Title != nil {
		existing.Title = *patch.Title
	}
	if patch.DirectorID != nil {
		existing.DirectorID = *patch.DirectorID
	}
	if patch.ReleaseYear != nil {
		existing.ReleaseYear = *patch.ReleaseYear
	}
	if patch.RuntimeMinutes != nil {
		existing.RuntimeMinutes = patch.RuntimeMinutes
	}
	if patch.Genre != nil {
		existing.Genre = patch.Genre
	}
	if patch.Rating != nil {
		existing.Rating = patch.Rating
	}

	if existing.Title == "" {
		return model.Movie{}, fmt.Errorf("title is required: %w", ErrValidation)
	}

	updated, err := s.repo.Update(ctx, id, existing)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *movieService) DeleteMovie(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return mapRepoError(err)
	}
	return nil
}

func mapRepoError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
