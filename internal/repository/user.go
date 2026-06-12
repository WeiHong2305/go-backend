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
