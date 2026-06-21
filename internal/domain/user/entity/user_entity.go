package entity

import "github.com/google/uuid"

type User struct {
	ID       uuid.UUID
	Name     string
	Email    string
	Password string // hashed password
	Role     string // "USER" or "ADMIN"
}
