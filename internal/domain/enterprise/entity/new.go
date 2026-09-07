package entity

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidCode      = errors.New("código inválido")
	ErrInvalidCreatedBy = errors.New("não foi possível identificar o usuário que está criando o registro")
)

func NewEnterprise(
	code int,
	name string,
	createdBy uuid.UUID,
) (*Enterprise, error) {
	if code < 0 {
		return nil, ErrInvalidCode
	}

	if createdBy == uuid.Nil {
		return nil, ErrInvalidCreatedBy
	}

	enterprise := &Enterprise{
		Code:      code,
		Name:      name,
		CreatedBy: createdBy,
	}

	return enterprise, nil
}
