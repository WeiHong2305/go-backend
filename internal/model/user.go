package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name" validate:"required,min=1,max=100"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
