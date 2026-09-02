package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type DeleteSalesDivisionUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *DeleteSalesDivisionUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanDeleteSalesDivision(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if code <= 0 {
		return errorsuc.NewValidationError("código da divisão de vendas inválido")
	}
	linked, err := uc.Repo.HasReferences(ctx, code)
	if err != nil {
		return err
	}
	if linked {
		return errorsuc.NewConflictError("a divisão de vendas possui vínculos e não pode ser excluída; desative-a para impedir novos usos sem perder o histórico")
	}
	return uc.Repo.Delete(ctx, code)
}
