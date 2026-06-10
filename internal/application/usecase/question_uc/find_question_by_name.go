package question_uc

import (
	"context"
	"errors"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/questions/repository"
)

type FindQuestionByName struct {
	Repo repository.QuestionsRepository
}

func NewFindQuestionByName(
	repo repository.QuestionsRepository,
) *FindQuestionByName {
	return &FindQuestionByName{
		Repo: repo,
	}
}

func (uc *FindQuestionByName) Execute(
	ctx context.Context,
	name string,
) (*response.QuestionResponse, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errorsuc.ErrInvalidSearchParams
	}

	question, err := uc.Repo.FindQuestionByName(ctx, name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errorsuc.ErrQuestionNotFound
		}
		return nil, err
	}
	return toQuestionResponse(question), nil
}
