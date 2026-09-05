package entity

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidName       = errors.New("informe o nome")
	ErrInvalidEmail      = errors.New("informe o e-mail")
	ErrInvalidPassword   = errors.New("informe a senha")
	ErrInvalidEnterprise = errors.New("o código da empresa deve ser maior que zero")
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
