package question_option_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/questions_options/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/questions_options/repository"
)

type DeleteQuestionOptionUseCase struct {
	Repo repository.QuestionsOptionsRepository
}

func NewDeleteQuestionOptionUseCase(
	repo repository.QuestionsOptionsRepository,
) *DeleteQuestionOptionUseCase {
	return &DeleteQuestionOptionUseCase{
		Repo: repo,
	}
}

func (uc *DeleteQuestionOptionUseCase) Execute(
	ctx context.Context,
	questionid int64,
) error {
	if err := entity.ValidateQuestionOptionDeletion(questionid); err != nil {
		return err
	}
	return uc.Repo.Delete(ctx, questionid)
}
