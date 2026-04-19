package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject"
)

type ItemStructureRepository interface {

	// ESCRITA:

	// Create persiste um novo componente de estrutura.
	Create(ctx context.Context, structure *entity.ItemStructure) (*entity.ItemStructure, error)

	// Update atualiza os campos editáveis (quantity, uom, loss%, position, notes).
	Update(ctx context.Context, structure *entity.ItemStructure) (*entity.ItemStructure, error)

	// Delete realiza o soft-delete de um componente.
	Delete(ctx context.Context, id int64) error

	// LEITURA SIMPLES:

	// GetByID retorna um componente pelo seu ID (independente de is_active).
	GetByID(ctx context.Context, id int64) (*entity.ItemStructure, error)

	// GetAllDirectChildren retorna TODOS os filhos ativos de um pai,
	// tanto genéricos quanto mascarados.
	GetAllDirectChildren(
		ctx context.Context,
		parentCode int64,
	) ([]*response.StructureComponentResponse, error)
	// GetGenericChildren retorna apenas os filhos genéricos (sem máscara) de um pai.
	GetGenericChildren(ctx context.Context, parentItemCode int64) ([]*entity.ItemStructure, error)

	// GetDirectChildrenForMask retorna filhos do pai correspondentes a uma
	// máscara específica E os filhos genéricos. A prioridade (específico > genérico)
	// é resolvida na camada de aplicação.
	GetDirectChildrenForMask(ctx context.Context, parentItemCode int64, mask string) ([]*entity.ItemStructure, error)

	// VALIDAÇÕES:

	// ItemExists verifica se um item com o ID informado existe e está ativo.
	ItemExists(ctx context.Context, itemCode int64) (bool, error)

	// HasCyclicReference retorna true se adicionar childItemCode como filho de
	// parentItemCode criaria um ciclo na árvore BOM.
	HasCyclicReference(ctx context.Context, parentItemCode, childItemCode int64) (bool, error)

	// SUPORTE À PROPAGAÇÃO DE MÁSCARA:

	// GetItemCodeAndDesc retorna o código textual e descrição de um item pelo ID.
	GetItemCodeAndDesc(ctx context.Context, itemCode int64) (code int64, desc string, err error)

	// GetMaskAnswersByItemAndValue retorna as respostas de uma máscara específica
	// de um item. Usado para propagar respostas do pai para os filhos.
	GetMaskAnswersByItemAndValue(ctx context.Context, itemID int64, maskValue string) ([]valueobject.MaskAnswer, error)

	// GetItemQuestions retorna as perguntas associadas a um item ordenadas por posição.
	// Usado para calcular qual parte da máscara do pai se aplica ao filho.
	GetItemQuestions(ctx context.Context, itemID int64) ([]valueobject.ItemQuestion, error)
}
