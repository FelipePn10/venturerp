package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"

	"github.com/google/uuid"
)

var (
	ErrInvalidCode      = domainerr.NewValidation("informe o código")
	ErrInvalidName      = domainerr.NewValidation("informe o nome")
	ErrInvalidWarehouse = domainerr.NewValidation("informe a mercadoria")
	ErrInvalidGroupCode = domainerr.NewValidation("o código do grupo deve ser maior que zero")
)

func NewComponent(
	name string,
	group_code string,
	code string,
	warehouse int64,
	created_by uuid.UUID,
) (*Component, error) {
	switch {
	case name == "":
		return nil, ErrInvalidName
	case group_code == "":
		return nil, ErrInvalidGroupCode
	case code == "":
		return nil, ErrInvalidCode
	case warehouse < 0:
		return nil, ErrInvalidWarehouse
	}

	return &Component{
		Name:      name,
		GroupCode: group_code,
		Code:      code,
		Warehouse: warehouse,
		CreatedBy: created_by,
	}, nil
}

func ValidateComponentDeletion(id int64) error {
	if id < 0 {
		return domainerr.NewValidation("o produto deve ser um código maior que zero")
	}
	return nil
}
