package errorsuc

import "errors"

var (
	ErrQuestionNotFound                  = errors.New("pergunta não encontrada")
	ErrQuestionOptionAlreadyExists       = errors.New("esta opção de resposta já existe")
	ErrWarehouseAlreadyExists            = errors.New("esta opção de almoxarifado já existe")
	ErrQuestionAlreadyExists             = errors.New("esta opção de resposta já existe")
	ErrInvalidQuestionName               = errors.New("nome de pergunta inválido")
	ErrInvalidProductNameAndCodeNotFound = errors.New("nome ou código não encontrado")
	ErrProductNotFound                   = errors.New("produto não encontrado")
	ErrProductAlreadyExists              = errors.New("este produto já existe")
	ErrItemAlreadyExists                 = errors.New("este item já existe")
	ErrCreateBom                         = errors.New("não foi possível concluir o cadastro")
	ErrCreateBomNotFound                 = errors.New("não foi possível concluir a operação; confira os dados enviados e tente novamente")
	ErrCreateBomItem                     = errors.New("não foi possível concluir o cadastro")
	ErrBomItemAlreadyExists              = errors.New("este produto já existe")
	ErrCreateBomItemNotFound             = errors.New("não foi possível concluir a operação; confira os dados enviados e tente novamente")
	ErrComponentAlreadyExists            = errors.New("este componente já existe")
	ErrWarehouseNotFound                 = errors.New("almoxarifado não encontrado")
	ErrUnauthorized                      = errors.New("usuário não autorizado")
	ErrInvalidSearchParams               = errors.New("params invalid")
)
