package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"go-backend/internal/model"
)

type UserRepository interface {
	Save(context.Context, model.User) (int, error)
	Get(context.Context, int) (model.User, error)
	GetAll(context.Context) ([]model.User, error)
	Update(context.Context, int, model.User) (model.User, error)
	Delete(context.Context, int) error
}

type MemoryUserStore struct {
	mu    sync.RWMutex
	users map[int]model.User
}

type PgUserStore struct {
	db *sql.DB
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{users: make(map[int]model.User)}
}

func NewPgUserStore(db *sql.DB) *PgUserStore {
	return &PgUserStore{db: db}
}

func (s *MemoryUserStore) Save(_ context.Context, u model.User) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := len(s.users) + 1
	u.ID = id
	s.users[id] = u

	return id, nil
}

func (s *PgUserStore) Save(ctx context.Context, u model.User) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `INSERT INTO users (name, active, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`

	var pk int
	err := s.db.QueryRowContext(ctx, query, u.Name, u.Active).Scan(&pk)
	if err != nil {
		return 0, fmt.Errorf("save user: %w", err)
	}
	return pk, nil
}

func (s *MemoryUserStore) Get(_ context.Context, id int) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return model.User{}, ErrNotFound
	}
	return user, nil
}

func (s *PgUserStore) Get(ctx context.Context, id int) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := "SELECT id, name, active, created_at, updated_at FROM users WHERE id = $1"
	user := model.User{}

	err := s.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return user, nil
}

func (s *MemoryUserStore) GetAll(_ context.Context) ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]model.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

func (s *PgUserStore) GetAll(ctx context.Context) ([]model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := "SELECT id, name, active, created_at, updated_at FROM users"
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (s *MemoryUserStore) Update(_ context.Context, id int, u model.User) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.users[id]
	if !ok {
		return model.User{}, ErrNotFound
	}
	u.ID = id
	u.CreatedAt = existing.CreatedAt
	u.UpdatedAt = time.Now()
	s.users[id] = u
	return u, nil
}

func (s *PgUserStore) Update(ctx context.Context, id int, u model.User) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `UPDATE users SET name = $1, active = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, active, created_at, updated_at`
	updated := model.User{}
	err := s.db.QueryRowContext(ctx, query, u.Name, u.Active, id).Scan(
		&updated.ID, &updated.Name, &updated.Active, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("update user %d: %w", id, err)
	}
	return updated, nil
}

func (s *MemoryUserStore) Delete(_ context.Context, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return ErrNotFound
	}
	delete(s.users, id)
	return nil
}

func (s *PgUserStore) Delete(ctx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := "DELETE FROM users WHERE id = $1"
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user %d rows affected: %w", id, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
