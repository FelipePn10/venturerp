package bom_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/bom/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/bom/repository"
)

type CreateBomUseCase struct {
	Repo repository.BomRepository
	Auth ports.AuthService
}

func NewCreateBomUseCase(
	repo repository.BomRepository,
	auth ports.AuthService,
) *CreateBomUseCase {
	return &CreateBomUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateBomUseCase) Execute(
	ctx context.Context,
	dto request.CreateBomUseCaseRequestDTO,
) (*entity.Bom, error) {
	if !uc.Auth.CanCreateBom(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	bom, err := entity.NewBom(
		dto.ProductId,
		dto.BomType,
		dto.MaskId,
		dto.Version,
		dto.ValidFrom,
		dto.Status,
	)

	if err != nil {
		if errors.Is(err, repository.ErrInvalidBom) {
			return nil, errorsuc.ErrCreateBom
		}
		return nil, err
	}

	created, err := uc.Repo.Create(ctx, bom)
	if err != nil {
		return nil, err
	}

	return created, nil
}
