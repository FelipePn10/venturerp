package usecase

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

type GetAllDirectChildrenUseCase struct {
	repo repository.ItemStructureRepository
	auth ports.AuthService
}

func (uc *GetAllDirectChildrenUseCase) Execute(
	ctx context.Context,
	dto request.GetAllDirectChildrenDTO,
) ([]*response.StructureComponentResponse, error) {

	if !uc.auth.GetAllStructure(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if dto.ParentItemCode <= 0 {
		return nil, fmt.Errorf("parentItemCode invalid")
	}

	return uc.repo.GetAllDirectChildren(ctx, dto.ParentItemCode)
}
