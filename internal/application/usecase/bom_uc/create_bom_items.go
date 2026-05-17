package bom_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	bomrepo "github.com/FelipePn10/panossoerp/internal/domain/bom/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_items/repository"
)

type CreateBomItemUseCase struct {
	Repo repository.BomItemsRepository
	Bom  bomrepo.BomRepository
	Auth ports.AuthService
}

func NewCreateBomItemUseCase(
	repo repository.BomItemsRepository,
	bom bomrepo.BomRepository,
	auth ports.AuthService,
) *CreateBomItemUseCase {
	return &CreateBomItemUseCase{
		Repo: repo,
		Bom:  bom,
		Auth: auth,
	}
}

func (uc *CreateBomItemUseCase) Execute(
	ctx context.Context,
	dto request.CreateBomItemsRequestDTO,
) (*entity.BomItems, error) {
	if !uc.Auth.CanCreateBomItems(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	bomItem, err := entity.NewBomItems(
		dto.BomID,
		dto.ComponentID,
		dto.Quantity,
		dto.Uom,
		dto.ScrapPercent,
		dto.OperationID,
		dto.MaskComponent,
	)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidBomItems) {
			return nil, errorsuc.ErrCreateBomItem
		}
		return nil, err
	}
	created, err := uc.Repo.Create(ctx, bomItem)
	if err != nil {
		return nil, err
	}

	return created, nil
}
