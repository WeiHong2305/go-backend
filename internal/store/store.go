package store

import (
	"database/sql"
	"log"
	"sync"

	"go-backend/internal/model"
)

type UserStore interface {
	Save(model.User) int
	Get(int) (model.User, bool)
	GetAll() []model.User
	Delete(int)
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

func (s *memoryUserStore) Save(u model.User) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := len(s.users) + 1
	s.users[id] = u
	return id
}

func (s *pgUserStore) Save(u model.User) int {
	query := `INSERT INTO users (name, active) VALUES ($1, $2) RETURNING id`

	var pk int
	err := s.db.QueryRow(query, u.Name, u.Active).Scan(&pk)
	if err != nil {
		log.Fatal(err)
	}
	return pk
}

func (s *memoryUserStore) Get(id int) (model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	return user, ok
}

func (s *pgUserStore) Get(id int) (model.User, bool) {
	query := "SELECT name, active FROM users WHERE id = $1"
	user := model.User{}

	err := s.db.QueryRow(query, id).Scan(&user.Name, &user.Active)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, false
		} else {
			log.Fatal(err)
		}
	}
	return user, true
}

func (s *memoryUserStore) GetAll() []model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]model.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

func (s *pgUserStore) GetAll() []model.User {
	users := []model.User{}

	query := "SELECT name, active FROM users"
	rows, err := s.db.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	var user model.User

	for rows.Next() {
		err := rows.Scan(&user.Name, &user.Active)
		if err != nil {
			log.Fatal(err)
		}
		users = append(users, user)
	}

	return users
}

func (s *memoryUserStore) Delete(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, id)
}

func (s *pgUserStore) Delete(id int) {
	query := "DELETE FROM users WHERE id = $1"
	_, err := s.db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
}
