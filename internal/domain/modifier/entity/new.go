package entity

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidDescription = errors.New("a descrição do modificador é obrigatória")
)

func NewModifier(
	description string,
	created_by uuid.UUID,
) (*Modifier, error) {
	if description == "" {
		return nil, ErrInvalidDescription
	}

	modifier := &Modifier{
		Description: description,
		CreatedBy:   created_by,
	}

	return modifier, nil
}
