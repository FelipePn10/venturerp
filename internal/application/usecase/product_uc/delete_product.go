package product_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/product/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/product/repository"
)

type DeleteProductUseCase struct {
	Repo repository.ProductRepository
	Auth ports.AuthService
}

func NewDeleteProductUseCase(
	repo repository.ProductRepository,
	auth ports.AuthService,
) *DeleteProductUseCase {
	return &DeleteProductUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *DeleteProductUseCase) Execute(
	ctx context.Context,
	id int64,
) error {
	if !uc.Auth.CanDeleteProduct(ctx) {
		return errorsuc.ErrUnauthorized
	}

	if err := entity.ValidateProductDeletion(id); err != nil {
		return err
	}
	return uc.Repo.Delete(ctx, id)
}
