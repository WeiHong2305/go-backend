package service

import (
	"context"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"go-backend/internal/repository"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	SignUp(ctx context.Context, user model.User) (model.User, error)
	LogIn(ctx context.Context, email, password string) (string, error)
	GetUser(ctx context.Context, id int64) (model.User, error)
	GetAllUsers(ctx context.Context) ([]model.User, error)
}

type userService struct {
	repo      repository.UserRepository
	jwtSecret []byte
}

func NewUserService(repo repository.UserRepository, jwtSecret []byte) *userService {
	return &userService{repo: repo, jwtSecret: jwtSecret}
}

func (s *userService) SignUp(ctx context.Context, user model.User) (model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("hash password: %w", err)
	}

	created, err := s.repo.Create(ctx, model.User{
		Email:    user.Email,
		Name:     user.Name,
		Password: string(hash),
	})
	if err != nil {
		return model.User{}, mapUserRepoError(err)
	}

	created.Password = ""
	return created, nil
}

func (s *userService) LogIn(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", fmt.Errorf("invalid email or password: %w", ErrUnauthorized)
		}
		return "", fmt.Errorf("get user: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", fmt.Errorf("invalid email or password: %w", ErrUnauthorized)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (s *userService) GetUser(ctx context.Context, id int64) (model.User, error) {
	user, err := s.repo.GetById(ctx, id)
	if err != nil {
		return model.User{}, mapUserRepoError(err)
	}
	return user, nil
}

func (s *userService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all users: %w", err)
	}
	return users, nil
}

func mapUserRepoError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrDuplicateEmail):
		return fmt.Errorf("email already taken: %w", ErrConflict)
	default:
		return fmt.Errorf("repository: %w", err)
	}
}
