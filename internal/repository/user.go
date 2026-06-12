package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/internal/model"
	"time"
)

type UserRepository interface {
	Create(context.Context, model.User) (int64, error)
}

type PgUserRepository struct {
	db *sql.DB
}

func NewPgUserRepository(db *sql.DB) *PgUserRepository {
	return &PgUserRepository{db: db}
}

func (r *PgUserRepository) Create(ctx context.Context, user model.User) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `INSERT INTO users (email, name, password) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, query, user.Email, user.Name, user.Password).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to save user: %w", err)
	}
	return id, nil
}
