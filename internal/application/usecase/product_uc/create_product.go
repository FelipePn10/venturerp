package product_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/product/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/product/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/product/valueobject"
)

type CreateProductUseCase struct {
	Repo repository.ProductRepository
	Auth ports.AuthService
}

func NewCreateProductUseCase(
	repo repository.ProductRepository,
	auth ports.AuthService,
) *CreateProductUseCase {
	return &CreateProductUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateProductUseCase) Execute(
	ctx context.Context,
	dto request.CreateProductDTO,
) (*response.ProductResponse, error) {
	if !uc.Auth.CanCreateProduct(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	code, err := valueobject.NewProductCode(dto.GroupCode)
	if err != nil {
		return nil, err
	}

	exists, err := uc.Repo.ExistsProductByCode(ctx, code.String())
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errorsuc.ErrProductAlreadyExists
	}

	product, err := entity.NewProduct(
		code.String(),
		dto.GroupCode,
		dto.Name,
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	saved, err := uc.Repo.Save(ctx, product)
	if err != nil {
		return nil, err
	}

	return toProductResponse(saved), nil
}
