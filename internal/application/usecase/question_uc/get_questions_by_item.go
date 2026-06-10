package question_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/associate_questions/repository"
)

type GetQuestionsByItemUseCase struct {
	Repo repository.AssociateQuestionsRepository
	Auth ports.AuthService
}

func NewGetQuestionsByItemUseCase(
	repo repository.AssociateQuestionsRepository,
	auth ports.AuthService,
) *GetQuestionsByItemUseCase {
	return &GetQuestionsByItemUseCase{Repo: repo, Auth: auth}
}

func (uc *GetQuestionsByItemUseCase) Execute(
	ctx context.Context,
	itemCode int64,
) ([]response.AssociateQuestionDetailResponse, error) {
	if !uc.Auth.CanAssociateByQuestionProduct(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.GetByItemCode(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toAssociateQuestionDetailResponses(list), nil
}
