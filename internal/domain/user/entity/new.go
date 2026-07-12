package entity

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidName       = errors.New("Name cannot be empty")
	ErrInvalidEmail      = errors.New("Email cannot be empty")
	ErrInvalidPassword   = errors.New("Password cannot be empty")
	ErrInvalidEnterprise = errors.New("Enterprise code must be greater than zero")
)

func NewUser(id uuid.UUID, name, email, password string) (*User, error) {
	switch {
	case name == "":
		return nil, ErrInvalidName
	case email == "":
		return nil, ErrInvalidEmail
	case password == "":
		return nil, ErrInvalidPassword
	}

	return &User{
		ID:       id,
		Name:     name,
		Email:    email,
		Password: password,
	}, nil
}
