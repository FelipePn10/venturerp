package productquestion

import "github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"

type AssociateQuestionItemRepository struct {
	q *sqlc.Queries
}

func NewAssociateQuestionItemRepositorySQLC(
	q *sqlc.Queries,
) *AssociateQuestionItemRepository {
	return &AssociateQuestionItemRepository{
		q: q,
	}
}
