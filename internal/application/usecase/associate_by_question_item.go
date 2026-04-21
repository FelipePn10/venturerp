package usecase

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/associate_questions/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/associate_questions/repository"
)

var (
	ErrQuestionAlreadyLinked = errors.New("question already linked to product")
	ErrPositionAlreadyUsed   = errors.New("position already used for product")
)

type AssociateByQuestionItemUseCase struct {
	repo repository.AssociateQuestionsRepository
	auth ports.AuthService
}

func (uc *AssociateByQuestionItemUseCase) Execute(
	ctx context.Context,
	dto request.AssociateByQuestionItemRequestDTO,
) error {
	if !uc.auth.CanAssociateByQuestionProduct(ctx) {
		return errorsuc.ErrUnauthorized
	}

	exists, err := uc.repo.ExistsByItemAndQuestion(
		ctx,
		dto.ItemCode,
		dto.QuestionID,
	)
	if err != nil {
		return err
	}
	if exists {
		return ErrQuestionAlreadyLinked
	}

	positionUsed, err := uc.repo.ExistsByItemAndPosition(
		ctx,
		dto.ItemCode,
		dto.Position,
	)
	if err != nil {
		return err
	}
	if positionUsed {
		return ErrPositionAlreadyUsed
	}

	pq, err := entity.New(
		dto.ItemCode,
		dto.QuestionID,
		dto.Position,
	)
	if err != nil {
		return err
	}

	return uc.repo.Associate(ctx, pq)
}
