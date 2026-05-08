package store

import (
	"database/sql"
	"fmt"
	"sync"

	"go-backend/internal/model"
)

type UserStore interface {
	Save(model.User) (int, error)
	Get(int) (model.User, error)
	GetAll() ([]model.User, error)
	Delete(int) error
}

type memoryUserStore struct {
	mu    sync.RWMutex
	users map[int]model.User
}

type pgUserStore struct {
	db *sql.DB
}

func NewMemoryUserStore() UserStore {
	return &memoryUserStore{users: make(map[int]model.User)}
}

func NewPgUserStore(db *sql.DB) UserStore {
	return &pgUserStore{db: db}
}

func (s *memoryUserStore) Save(u model.User) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := len(s.users) + 1
	u.ID = id
	s.users[id] = u
	return id, nil
}

func (s *pgUserStore) Save(u model.User) (int, error) {
	query := `INSERT INTO users (name, active, created_at, updated_at) VALUES ($1, $2, NOW(), NOW()) RETURNING id`

	var pk int
	err := s.db.QueryRow(query, u.Name, u.Active).Scan(&pk)
	return pk, err
}

func (s *memoryUserStore) Get(id int) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return model.User{}, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *pgUserStore) Get(id int) (model.User, error) {
	query := "SELECT id, name, active, created_at, updated_at FROM users WHERE id = $1"
	user := model.User{}

	err := s.db.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, fmt.Errorf("user not found")
		}
		return model.User{}, err
	}
	return user, nil
}

func (s *memoryUserStore) GetAll() ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]model.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

func (s *pgUserStore) GetAll() ([]model.User, error) {
	users := []model.User{}

	query := "SELECT id, name, active, created_at, updated_at FROM users"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Name, &user.Active, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (s *memoryUserStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(s.users, id)
	return nil
}

func (s *pgUserStore) Delete(id int) error {
	query := "DELETE FROM users WHERE id = $1"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
