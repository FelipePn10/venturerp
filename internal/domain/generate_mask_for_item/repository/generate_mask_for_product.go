package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity"
)

type GenerateMaskForItemRepository interface {
	Generate(ctx context.Context, mask *entity.ItemMask) (*entity.ItemMask, error)
	GetOptionValue(ctx context.Context, optionID int64) (string, error)
	// GetOptionIDByQuestionAndValue resolves a text value back to its option ID
	// (needed when a locked restriction value must be injected into the mask).
	GetOptionIDByQuestionAndValue(ctx context.Context, questionID int64, value string) (int64, error)
	// GetItemQuestionPositions returns the position of each question associated
	// with the item. Used to keep the mask position-order correct after filtering.
	GetItemQuestionPositions(ctx context.Context, itemCode int64) (map[int64]int, error)
}
