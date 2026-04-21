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

	// VALIDATIONS

	ItemExists(ctx context.Context, itemCode int64) (bool, error)
	HasCyclicReference(ctx context.Context, parentCode, childCode int64) (bool, error)

	// SUPPORT (MASK RUNTIME)

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
