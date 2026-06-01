package service

import (
	"context"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/store"
)

type UserService interface {
	CreateUser(ctx context.Context, name string, active *bool) (model.User, error)
	GetUser(ctx context.Context, id int) (model.User, error)
	GetAllUsers(ctx context.Context) ([]model.User, error)
	UpdateUser(ctx context.Context, id int, name string, active *bool) (model.User, error)
	DeleteUser(ctx context.Context, id int) error
}

type userService struct {
	repo store.UserRepository
}

func NewUserService(repo store.UserRepository) *userService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, name string, active *bool) (model.User, error) {
	if name == "" {
		return model.User{}, fmt.Errorf("name is required: %w", ErrValidation)
	}

	user := model.User{Name: name, Active: true}
	if active != nil {
		user.Active = *active
	}

	id, err := s.repo.Save(ctx, user)
	if err != nil {
		return model.User{}, mapRepoError(err)
	}

	user.ID = id
	return user, nil
}

func (s *userService) GetUser(ctx context.Context, id int) (model.User, error) {
	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.User{}, mapRepoError(err)
	}
	return user, nil
}

func (s *userService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, mapRepoError(err)
	}
	if users == nil {
		users = []model.User{}
	}
	return users, nil
}

func (s *userService) UpdateUser(ctx context.Context, id int, name string, active *bool) (model.User, error) {
	if name == "" {
		return model.User{}, fmt.Errorf("name is required: %w", ErrValidation)
	}

	patch := model.User{Name: name, Active: true}
	if active != nil {
		patch.Active = *active
	}

	updated, err := s.repo.Update(ctx, id, patch)
	if err != nil {
		return model.User{}, mapRepoError(err)
	}
	return updated, nil
}

func (s *userService) DeleteUser(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return mapRepoError(err)
	}
	return nil
}

func mapRepoError(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
