package model

import "time"

type Movie struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Director    string    `json:"director"`
	ReleaseYear int       `json:"release_year"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
