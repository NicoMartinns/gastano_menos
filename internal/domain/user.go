package domain

import "time"

type User struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	Role           string    `json:"role"`
	IsActive       bool      `json:"is_active"`
	SessionVersion int       `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

const (
	RoleUser  = "USER"
	RoleAdmin = "ADMIN"
)
