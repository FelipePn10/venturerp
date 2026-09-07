package entity

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidCode      = errors.New("código inválido")
	ErrInvalidCreatedBy = errors.New("não foi possível identificar o usuário que está criando o registro")
)

func NewGroup(
	code int,
	description string,
	enterpriseId int,
	createdBy uuid.UUID,
) (*Group, error) {
	if code < 0 {
		return nil, ErrInvalidCode
	}

	if createdBy == uuid.Nil {
		return nil, ErrInvalidCreatedBy
	}

	group := &Group{
		Code:         code,
		Description:  description,
		EnterpriseID: enterpriseId,
		CreatedBy:    createdBy,
	}

	return group, nil
}
