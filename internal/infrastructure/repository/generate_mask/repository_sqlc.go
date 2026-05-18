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
	return r.q.GetOptionValueByID(ctx, optionID)
}

func (r *repositoryGenerateMaskSQLC) GetOptionIDByQuestionAndValue(
	ctx context.Context,
	questionID int64,
	value string,
) (int64, error) {
	return r.q.GetOptionIDByQuestionAndValue(ctx, sqlc.GetOptionIDByQuestionAndValueParams{
		QuestionID: questionID,
		Lower:      value,
	})
}

func (r *repositoryGenerateMaskSQLC) GetItemQuestionPositions(
	ctx context.Context,
	itemCode int64,
) (map[int64]int, error) {
	rows, err := r.q.GetItemQuestionPositions(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]int, len(rows))
	for _, row := range rows {
		out[row.QuestionID] = int(row.Position)
	}
	return out, nil
}
