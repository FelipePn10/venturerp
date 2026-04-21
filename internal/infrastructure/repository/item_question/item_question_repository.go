package productquestion

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/associate_questions/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *AssociateQuestionItemRepository) Associate(
	ctx context.Context,
	pq *entity.AssociateQuestion,
) error {
	return r.q.AssociateQuestionItem(ctx, sqlc.AssociateQuestionItemParams{
		ItemCode:   pq.ItemCode,
		QuestionID: pq.QuestionID,
		Position:   int32(pq.Position),
		CreatedAt:  pq.CreatedAt,
	})
}

func (r *AssociateQuestionItemRepository) ExistsByItemAndQuestion(
	ctx context.Context,
	itemID int64,
	questionID int64,
) (bool, error) {
	return r.q.ExistsByItemAndQuestion(ctx, sqlc.ExistsByItemAndQuestionParams{
		ItemCode:   itemID,
		QuestionID: questionID,
	})
}

func (r *AssociateQuestionItemRepository) ExistsByItemAndPosition(
	ctx context.Context,
	itemID int64,
	position int,
) (bool, error) {
	return r.q.ExistsByItemAndPosition(ctx, sqlc.ExistsByItemAndPositionParams{
		ItemCode: itemID,
		Position: int32(position),
	})
}
