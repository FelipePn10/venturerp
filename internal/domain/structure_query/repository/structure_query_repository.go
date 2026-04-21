package repository

import (
	"context"

	maskservice "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service"
	maskvo "github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	str "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/google/uuid"
)

type StructureQueryRepository interface {
	// Item
	GetItemByCode(ctx context.Context, code int64) (*itementity.Item, error)

	// Estrutura — mask="" retorna apenas filhos universais (parent_mask IS NULL).
	// mask="1.94M#1.94M" retorna universais + específicos para aquela máscara.
	GetDirectChildrenForMask(ctx context.Context, parentCode int64, mask string) ([]*str.ItemStructure, error)

	// Máscara — o SQL deve fazer JOIN e retornar o option_value.
	GetMaskAnswersByItemAndValue(ctx context.Context, itemCode int64, mask string) ([]maskvo.MaskAnswer, error)
	GetItemQuestions(ctx context.Context, itemCode int64) ([]maskservice.ItemQuestion, error)

	// Cria automaticamente uma máscara propagada; as respostas já chegam com
	// questionID + optionID + position, sem precisar de reverse-lookup.
	CreateMaskForItem(ctx context.Context, itemCode int64, mask string, answers []maskservice.ChildMaskAnswerInput, createdBy uuid.UUID) error
}
