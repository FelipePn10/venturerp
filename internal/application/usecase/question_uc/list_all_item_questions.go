package question_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/associate_questions/repository"
)

type ListAllItemQuestionsUseCase struct {
	Repo repository.AssociateQuestionsRepository
	Auth ports.AuthService
}

func NewListAllItemQuestionsUseCase(
	repo repository.AssociateQuestionsRepository,
	auth ports.AuthService,
) *ListAllItemQuestionsUseCase {
	return &ListAllItemQuestionsUseCase{Repo: repo, Auth: auth}
}

func (uc *ListAllItemQuestionsUseCase) Execute(
	ctx context.Context,
) ([]response.ItemQuestionRowResponse, error) {
	if !uc.Auth.CanAssociateByQuestionProduct(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return toItemQuestionRowResponses(list), nil
}
