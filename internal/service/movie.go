package service

import (
	"context"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type MovieService interface {
	CreateMovie(ctx context.Context, movie model.Movie) (model.Movie, error)
	GetMovie(ctx context.Context, id int64) (model.Movie, error)
	GetAllMovies(ctx context.Context) ([]model.Movie, error)
	UpdateMovie(ctx context.Context, id int64, movie model.Movie) (model.Movie, error)
	DeleteMovie(ctx context.Context, id int64) error
}

type movieService struct {
	repo repository.MovieRepository
}

func NewMovieService(repo repository.MovieRepository) *movieService {
	return &movieService{repo: repo}
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
	movie, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.Movie{}, mapRepoError(err)
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

func (s *movieService) UpdateMovie(ctx context.Context, id int64, movie model.Movie) (model.Movie, error) {
	if movie.Title == "" {
		return model.Movie{}, fmt.Errorf("title is required: %w", ErrValidation)
	}

	updated, err := s.repo.Update(ctx, id, movie)
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
