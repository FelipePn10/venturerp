package generatemask

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *repositoryGenerateMaskSQLC) Generate(
	ctx context.Context,
	mask *entity.ItemMask,
) (*entity.ItemMask, error) {

	params := sqlc.InsertItemtMaskParams{
		ItemCode:  mask.ItemCode,
		Mask:      mask.Mask,
		MaskHash:  mask.MaskHash,
		CreatedBy: pgutil.ToPgUUID(mask.CreatedBy),
	}

	maskRecord, err := r.q.InsertItemtMask(ctx, params)
	if err != nil {
		return nil, err
	}

	for _, ans := range mask.Answers {
		err := r.q.InsertItemMaskAnswer(ctx, sqlc.InsertItemMaskAnswerParams{
			MaskID:     maskRecord.ID,
			QuestionID: ans.QuestionID(),
			OptionID:   ans.OptionID(),
			Position:   int32(ans.Position()),
		})
		if err != nil {
			return nil, err
		}
	}

	return mask, nil
}

func (r *repositoryGenerateMaskSQLC) GetOptionValue(
	ctx context.Context,
	optionID int64,
) (string, error) {

	value, err := r.q.GetOptionValueByID(ctx, optionID)
	if err != nil {
		return "", err
	}
	return value, nil
}
