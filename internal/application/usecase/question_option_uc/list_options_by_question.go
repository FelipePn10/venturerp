package question_option_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/questions_options/repository"
)

type ListOptionsByQuestionUseCase struct {
	Repo repository.QuestionsOptionsRepository
	Auth ports.AuthService
}

func NewListOptionsByQuestionUseCase(
	repo repository.QuestionsOptionsRepository,
	auth ports.AuthService,
) *ListOptionsByQuestionUseCase {
	return &ListOptionsByQuestionUseCase{Repo: repo, Auth: auth}
}

func (uc *ListOptionsByQuestionUseCase) Execute(
	ctx context.Context,
	questionID int64,
) ([]*response.QuestionOptionResponse, error) {
	if !uc.Auth.CanCreateQuestionOption(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListByQuestionID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	return toQuestionOptionResponsesFromValues(list), nil
}
