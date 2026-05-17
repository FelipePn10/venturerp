package question_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/questions/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/questions/repository"
)

type DeleteQuestionUseCase struct {
	Repo repository.QuestionsRepository
}

func NewDeleteQuestionUseCase(
	repo repository.QuestionsRepository,
) *DeleteQuestionUseCase {
	return &DeleteQuestionUseCase{
		Repo: repo,
	}
}

func (uc *DeleteQuestionUseCase) Execute(
	ctx context.Context,
	id int64,
) error {
	if err := entity.ValidateQuestionDeletion(id); err != nil {
		return err
	}
	return uc.Repo.Delete(ctx, id)
}
