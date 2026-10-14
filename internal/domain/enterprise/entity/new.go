package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/google/uuid"
)

var (
	ErrInvalidCode      = domainerr.NewValidation("código inválido")
	ErrInvalidCreatedBy = domainerr.NewValidation("não foi possível identificar o usuário que está criando o registro")
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
