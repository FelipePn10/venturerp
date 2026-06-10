package question_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/questions/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/questions/repository"
)

type CreateQuestion struct {
	Repo repository.QuestionsRepository
	Auth ports.AuthService
}

func NewCreateQuestion(
	repo repository.QuestionsRepository,
	auth ports.AuthService,
) *CreateQuestion {
	return &CreateQuestion{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateQuestion) Execute(
	ctx context.Context,
	dto request.CreateQuestionRequestDTO,
) (*response.QuestionResponse, error) {
	if !uc.Auth.CanCreateQuestion(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	exists, err := uc.Repo.ExistsQuestionByName(ctx, dto.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errorsuc.ErrQuestionAlreadyExists
	}

	qst, err := entity.NewQuestion(
		dto.Name,
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	crate, err := uc.Repo.Save(ctx, qst)
	if err != nil {
		return nil, err
	}
	return toQuestionResponse(crate), nil
}
