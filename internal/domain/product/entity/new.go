package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidCode      = errors.New("informe o código")
	ErrInvalidName      = errors.New("informe o nome")
	ErrInvalidGroupCode = errors.New("o código do grupo deve ser maior que zero")
)

func NewProduct(
	code string,
	group_code string,
	name string,
	createdBy uuid.UUID,
) (*Product, error) {

	switch {
	case code == "":
		return nil, ErrInvalidCode
	case name == "":
		return nil, ErrInvalidName
	case group_code <= "":
		return nil, ErrInvalidGroupCode
	case createdBy == uuid.Nil:
		return nil, errors.New("é preciso identificar o usuário que está criando o registro")
	}

	id := int64(time.Now().UnixNano())
	return &Product{
		ID:        id,
		Code:      code,
		GroupCode: group_code,
		Name:      name,
		CreatedBy: createdBy,
	}, nil
}

func ValidateProductDeletion(id int64) error {
	if id == 0 {
		return errors.New("o produto deve ser um código maior que zero")
	}
	return nil
}
