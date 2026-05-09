package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
)

type ItemStructureRepository interface {

	// WRITE

	Create(ctx context.Context, structure *entity.ItemStructure) (*entity.ItemStructure, error)
	Update(ctx context.Context, structure *entity.ItemStructure) (*entity.ItemStructure, error)
	Delete(ctx context.Context, id int64) error

	// READ

	GetByID(ctx context.Context, id int64) (*entity.ItemStructure, error)

	// GetAllDirectChildren retorna TODOS os filhos ativos de um pai
	// tanto genéricos quanto mascarados.
	GetAllDirectChildren(
		ctx context.Context,
		parentCode int64,
	) ([]*entity.ItemStructure, error)

	GetDirectChildrenForMask(
		ctx context.Context,
		parentCode int64,
		mask string,
	) ([]*entity.ItemStructure, error)

	// LoadBOMForRoots pre-loads the entire BOM tree for a set of root items in
	// a single recursive query, returning an adjacency map parent→children.
	// Used by the MRP engine to avoid N+1 queries during LLC computation and BOM explosion.
	LoadBOMForRoots(ctx context.Context, rootCodes []int64) (map[int64][]*entity.ItemStructure, error)

	// VALIDATIONS

	ItemExists(ctx context.Context, itemCode int64) (bool, error)
	HasCyclicReference(ctx context.Context, parentCode, childCode int64) (bool, error)

	// SUPPORT (MASK RUNTIME)
	SequenceExists(ctx context.Context, parentCode int64, sequence int) (bool, error)

	GetItemCodeAndDesc(ctx context.Context, itemCode int64) (int64, string, error)

	GetMaskAnswersByItemAndValue(
		ctx context.Context,
		itemCode int64,
		maskValue string,
	) ([]maskvo.MaskAnswer, error)

	GetItemQuestions(
		ctx context.Context,
		itemCode int64,
	) ([]service.ItemQuestion, error)
}
