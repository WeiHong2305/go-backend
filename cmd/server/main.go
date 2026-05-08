package main

import (
	"database/sql"
	"fmt"
	"go-backend/internal/api"
	"go-backend/internal/store"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgres://postgres:secret@localhost:5433/gopgtest?sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	createProductTable(db)

	// users := store.NewMemoryUserStore()
	users := store.NewPgUserStore(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/", api.RootHandler)
	mux.HandleFunc("/health", api.HealthHandler)
	mux.HandleFunc("POST /users", api.CreateUserHandler(users))
	mux.HandleFunc("GET /users/{id}", api.GetUserHandler(users))
	mux.HandleFunc("GET /users", api.GetAllUsersHandler(users))
	mux.HandleFunc("DELETE /users/{id}", api.DeleteUserHandler(users))

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func createProductTable(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name varchar(100) NOT NULL,
		active boolean DEFAULT true,
		created timestamp DEFAULT NOW()
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}
