package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-backend/internal/model"
	"time"

	"github.com/lib/pq"
)

type UserRepository interface {
	Create(context.Context, model.User) (model.User, error)
	GetUsingEmail(context.Context, string) (model.User, error)
	GetUsingId(context.Context, int64) (model.User, error)
}

type PgUserRepository struct {
	db *sql.DB
}

func NewPgUserRepository(db *sql.DB) *PgUserRepository {
	return &PgUserRepository{db: db}
}

func (r *PgUserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `INSERT INTO users (email, name, password) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, user.Email, user.Name, user.Password).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return model.User{}, fmt.Errorf("%w: %s", ErrDuplicateEmail, user.Email)
		}
		return model.User{}, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}

func (r *PgUserRepository) GetUsingEmail(ctx context.Context, email string) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var user model.User
	query := `SELECT id, email, name, password, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, fmt.Errorf("%w: %s", ErrNotFound, email)
		}
		return model.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}
	return user, nil
}

func (r *PgUserRepository) GetUsingId(ctx context.Context, id int64) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var user model.User
	query := `SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return model.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}
	return user, nil
}
