package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListContasReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListContasReceberUseCase) Execute(
	ctx context.Context, dto request.ListContasReceberFilter,
) ([]*entity.ContaReceber, error) {
	if !uc.Auth.CanListContasReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	filters := repository.CRFilter{
		Status:       dto.Status,
		ClienteID:    dto.ClienteID,
		SalesOrderID: dto.SalesOrderID,
		FiscalExitID: dto.FiscalExitID,
		Documento:    dto.Documento,
		ValorMinimo:  dto.ValorMinimo,
		ValorMaximo:  dto.ValorMaximo,
		DateField:    campoDeData(dto.DateField),
	}
	if dto.SomenteVencidos != nil {
		filters.SomenteVencidos = *dto.SomenteVencidos
	}
	inicio, err := dataDoFiltro(dto.StartDate, "data inicial")
	if err != nil {
		return nil, err
	}
	filters.StartDate = inicio
	fim, err := dataDoFiltro(dto.EndDate, "data final")
	if err != nil {
		return nil, err
	}
	filters.EndDate = fim
	if inicio != nil && fim != nil && fim.Before(*inicio) {
		return nil, errorsuc.NewValidationError("a data final do período é anterior à data inicial")
	}
	if filters.ValorMinimo != nil && filters.ValorMaximo != nil && *filters.ValorMaximo < *filters.ValorMinimo {
		return nil, errorsuc.NewValidationError("o valor máximo do filtro é menor que o valor mínimo")
	}

	return uc.Repo.ListContasReceber(ctx, filters)
}
