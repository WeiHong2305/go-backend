package service

import (
	"context"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type MovieService interface {
	CreateMovie(ctx context.Context, title, director string, releaseYear int) (model.Movie, error)
	GetMovie(ctx context.Context, id int64) (model.Movie, error)
	GetAllMovies(ctx context.Context) ([]model.Movie, error)
	UpdateMovie(ctx context.Context, id int64, title, director string, releaseYear int) (model.Movie, error)
	DeleteMovie(ctx context.Context, id int64) error
}

type movieService struct {
	repo repository.MovieRepository
}

func NewMovieService(repo repository.MovieRepository) *movieService {
	return &movieService{repo: repo}
}

func (s *movieService) CreateMovie(ctx context.Context, title string, director string, releaseYear int) (model.Movie, error) {
	if title == "" {
		return model.Movie{}, fmt.Errorf("title is required: %w", ErrValidation)
	}

	movie := model.Movie{Title: title, Director: director, ReleaseYear: releaseYear}
	id, err := s.repo.Save(ctx, movie)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
	}

	movie.ID = id
	return movie, nil
}
