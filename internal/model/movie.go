package model

type Movie struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Director    string `json:"director"`
	ReleaseYear int    `json:"release_year"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
