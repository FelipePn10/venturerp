package structure_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type GetAllDirectChildrenUseCase struct {
	Repo repository.ItemStructureRepository
	Auth ports.AuthService
}

func NewGetAllDirectChildrenUseCase(
	repo repository.ItemStructureRepository,
	auth ports.AuthService,
) *GetAllDirectChildrenUseCase {
	return &GetAllDirectChildrenUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *GetAllDirectChildrenUseCase) Execute(
	ctx context.Context,
	dto request.GetAllDirectChildrenDTO,
) ([]*entity.ItemStructure, error) {

	if !uc.Auth.GetAllStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if dto.ParentItemCode <= 0 {
		return nil, fmt.Errorf("parentItemCode invalid")
	}

	return uc.Repo.GetAllDirectChildren(ctx, dto.ParentItemCode)
}
