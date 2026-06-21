package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/repository"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type MovieService interface {
	CreateMovie(ctx context.Context, movie model.Movie) (model.Movie, error)
	GetMovie(ctx context.Context, id int64) (model.Movie, error)
	GetAllMovies(ctx context.Context) ([]model.Movie, error)
	UpdateMovie(ctx context.Context, id int64, patch model.MoviePatch) (model.Movie, error)
	DeleteMovie(ctx context.Context, id int64) error
}

type movieService struct {
	repo        repository.MovieRepository
	redisClient *redis.Client
	cacheTTL    time.Duration
}

func NewMovieService(repo repository.MovieRepository, redisClient *redis.Client, cacheTTL time.Duration) *movieService {
	return &movieService{repo: repo, redisClient: redisClient, cacheTTL: cacheTTL}
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
	cacheKey := fmt.Sprintf("movies:%s", strconv.FormatInt(id, 10))

	// 1. Attempt to fetch from Redis
	cacheValue, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache Hit
		slog.Info("Cache Hit")
		var movie model.Movie
		if err := json.Unmarshal([]byte(cacheValue), &movie); err == nil {
			return movie, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		slog.Warn("Failed to read from Redis", "error", err)
	}

	// 2. Cache Miss: Fetch from the primary database
	slog.Info("Cache Miss")
	movie, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
	}

	// 3. Serialize and save back to Redis
	data, err := json.Marshal(movie)
	if err != nil {
		slog.Warn("Failed to GetMovie: Json marshal error", "error", err)
	}
	if err := s.redisClient.Set(ctx, cacheKey, data, s.cacheTTL).Err(); err != nil {
		slog.Warn("Failed to write to Redis", "error", err)
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
