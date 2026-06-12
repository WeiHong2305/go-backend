package service

import (
	"context"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	SignUp(ctx context.Context, user model.User) (model.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *userService {
	return &userService{repo: repo}
}

func (s *userService) SignUp(ctx context.Context, user model.User) (model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("Failed to hash password: %w", err)
	}

	created, err := s.repo.Create(ctx, model.User{
		Email:    user.Email,
		Name:     user.Name,
		Password: string(hash),
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return model.User{}, fmt.Errorf("email already taken: %w", ErrConflict)
		}
		return model.User{}, err
	}

	created.Password = ""
	return created, nil
}
