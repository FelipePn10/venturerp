package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/google/uuid"
)

var (
	ErrInvalidName       = domainerr.NewValidation("informe o nome")
	ErrInvalidEmail      = domainerr.NewValidation("informe o e-mail")
	ErrInvalidPassword   = domainerr.NewValidation("informe a senha")
	ErrInvalidEnterprise = domainerr.NewValidation("o código da empresa deve ser maior que zero")
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
