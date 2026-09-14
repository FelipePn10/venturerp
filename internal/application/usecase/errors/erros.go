package errorsuc

import "errors"

// Estes sentinelas são devolvidos pelos casos de uso e chegam ao usuário. Eram
// `errors.New` puro, e RespondUseCaseError classifica por TIPO: qualquer um
// deles virava "erro interno do servidor" (500), escondendo a mensagem que já
// estava escrita e em português. Tipando-os, cada um passa a sair com o status
// que lhe cabe — 404 para "não encontrado", 409 para "já existe", 422 para o
// resto — sem tocar em nenhum caso de uso.
//
// `errors.Is` continua funcionando: a comparação é por identidade do ponteiro,
// e os handlers que já faziam essa checagem seguem válidos.
var (
	ErrQuestionNotFound                  = NewNotFoundError("pergunta não encontrada")
	ErrQuestionOptionAlreadyExists       = NewConflictError("esta opção de resposta já existe")
	ErrWarehouseAlreadyExists            = NewConflictError("esta opção de almoxarifado já existe")
	ErrQuestionAlreadyExists             = NewConflictError("esta opção de resposta já existe")
	ErrInvalidQuestionName               = NewValidationError("nome de pergunta inválido")
	ErrInvalidProductNameAndCodeNotFound = NewNotFoundError("nome ou código não encontrado")
	ErrProductNotFound                   = NewNotFoundError("produto não encontrado")
	ErrProductAlreadyExists              = NewConflictError("este produto já existe")
	ErrItemAlreadyExists                 = NewConflictError("este item já existe")
	ErrCreateBom                         = NewValidationError("não foi possível concluir o cadastro")
	ErrCreateBomNotFound                 = NewValidationError("não foi possível concluir a operação; confira os dados enviados e tente novamente")
	ErrCreateBomItem                     = NewValidationError("não foi possível concluir o cadastro")
	ErrBomItemAlreadyExists              = NewConflictError("este produto já existe")
	ErrCreateBomItemNotFound             = NewValidationError("não foi possível concluir a operação; confira os dados enviados e tente novamente")
	ErrComponentAlreadyExists            = NewConflictError("este componente já existe")
	ErrWarehouseNotFound                 = NewNotFoundError("almoxarifado não encontrado")
	// Continua como sentinela simples: RespondUseCaseError já o trata primeiro,
	// mapeando para 403.
	ErrUnauthorized = errors.New("usuário não autorizado")
	// Estava em inglês ("params invalid") e chegava assim ao usuário.
	ErrInvalidSearchParams = NewValidationError("parâmetros de busca inválidos")
)
