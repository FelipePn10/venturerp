package generate_mask_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject"
)

type GenerateMaskForItemUseCase struct {
	Repo repository.GenerateMaskForItemRepository
	Auth ports.AuthService
}

func NewGenerateMaskItemUseCase(
	repo repository.GenerateMaskForItemRepository,
	auth ports.AuthService,
) *GenerateMaskForItemUseCase {
	return &GenerateMaskForItemUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *GenerateMaskForItemUseCase) Execute(
	ctx context.Context,
	dto request.GenerateMaskItemRequestDTO,
) (*entity.ItemMask, error) {
	if uc.Auth == nil {
		return nil, errors.New("auth mask not initialized")
	}
	if uc.Repo == nil {
		return nil, errors.New("repository not initialized")
	}

	if !uc.Auth.CanGenerateMaskForItem(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.Answers) == 0 {
		return nil, errors.New("answers cannot be empty")
	}

	answers := make([]valueobject.MaskAnswer, 0, len(dto.Answers))

	for _, a := range dto.Answers {
		optionValue, err := uc.Repo.GetOptionValue(ctx, a.OptionID)
		if err != nil {
			return nil, err
		}

		answer, err := valueobject.NewMaskAnswer(
			a.QuestionID,
			a.OptionID,
			a.Position,
			optionValue,
		)
		if err != nil {
			return nil, err
		}

		answers = append(answers, answer)
	}

	mask, err := valueobject.NewItemMask(dto.ItemCode, answers)
	if err != nil {
		return nil, err
	}

	itemMask, err := entity.NewItemMask(
		dto.ItemCode,
		mask.Value(),
		mask.Hash(),
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	generate, err := uc.Repo.Generate(ctx, itemMask)
	if err != nil {
		return nil, err
	}

	return generate, nil
}
