package model

import "time"

type Movie struct {
	ID             int64     `json:"id"`
	Title          string    `json:"title"`
	DirectorID     int64     `json:"director_id"`
	ReleaseYear    int       `json:"release_year"`
	RuntimeMinutes *int      `json:"runtime_minutes,omitempty"`
	Genre          *string   `json:"genre,omitempty"`
	Rating         *float64  `json:"rating,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MoviePatch struct {
	Title          *string  `json:"title,omitempty"`
	DirectorID     *int64   `json:"director_id,omitempty"`
	ReleaseYear    *int     `json:"release_year,omitempty"`
	RuntimeMinutes *int     `json:"runtime_minutes,omitempty"`
	Genre          *string  `json:"genre,omitempty"`
	Rating         *float64 `json:"rating,omitempty"`
}
