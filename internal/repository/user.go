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
	Create(ctx context.Context, user model.User) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetById(ctx context.Context, id int64) (model.User, error)
	GetAll(ctx context.Context) ([]model.User, error)
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

	query := `INSERT INTO users (email, name, password) VALUES ($1, $2, $3) RETURNING id, is_admin, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, user.Email, user.Name, user.Password).Scan(&user.ID, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return model.User{}, fmt.Errorf("%w: %s", ErrDuplicateEmail, user.Email)
		}
		return model.User{}, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}

func (r *PgUserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var user model.User
	query := `SELECT id, email, name, password, is_admin, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, fmt.Errorf("%w: %s", ErrNotFound, email)
		}
		return model.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}
	return user, nil
}

func (r *PgUserRepository) GetById(ctx context.Context, id int64) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var user model.User
	query := `SELECT id, email, name, is_admin, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Email, &user.Name, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, fmt.Errorf("%w: %d", ErrNotFound, id)
		}
		return model.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}
	return user, nil
}

func (r *PgUserRepository) GetAll(ctx context.Context) ([]model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, email, name, is_admin, created_at, updated_at FROM users`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan users: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}
