package component_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/component/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/component/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/component/valueobject"
)

type CreateComponentUseCase struct {
	Repo repository.ComponentRepository
	Auth ports.AuthService
}

func NewCreateComponentUseCase(
	repo repository.ComponentRepository,
	auth ports.AuthService,
) *CreateComponentUseCase {
	return &CreateComponentUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateComponentUseCase) Execute(
	ctx context.Context,
	dto request.CreateComponentRequestDTO,
) (*response.ComponentResponse, error) {

	if !uc.Auth.CanCreateComponent(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	code, err := valueobject.NewComponentCode(dto.GroupCode)
	if err != nil {
		return nil, err
	}

	exists, err := uc.Repo.ExistsComponentByCode(ctx, code.String())
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errorsuc.ErrComponentAlreadyExists
	}

	component, err := entity.NewComponent(
		code.String(),
		dto.GroupCode,
		dto.Name,
		dto.Warehouse,
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	saved, err := uc.Repo.Save(ctx, component)
	if err != nil {
		return nil, err
	}
	return toComponentResponse(saved), nil
}
