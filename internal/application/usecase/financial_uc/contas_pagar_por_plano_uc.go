package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

// ContasPagarPorPlanoUseCase é o contas a pagar visto por plano de contas: a
// nota de 100 mil com 50 mil de matéria-prima e 50 mil de EPI aparece como 50
// mil em cada plano, com o pago e o aberto de cada um.
type ContasPagarPorPlanoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

type ContasPagarPorPlanoDTO struct {
	StartDate    *string
	EndDate      *string
	DateField    *string
	Status       *string
	FornecedorID *int64
}

func (uc *ContasPagarPorPlanoUseCase) Execute(ctx context.Context, dto ContasPagarPorPlanoDTO) ([]repository.CPPorPlano, error) {
	if !uc.Auth.CanListContasPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	inicio, err := dataDoFiltro(dto.StartDate, "data inicial")
	if err != nil {
		return nil, err
	}
	fim, err := dataDoFiltro(dto.EndDate, "data final")
	if err != nil {
		return nil, err
	}
	if inicio != nil && fim != nil && fim.Before(*inicio) {
		return nil, errorsuc.NewValidationError("a data final do período é anterior à data inicial")
	}
	return uc.Repo.ContasPagarPorPlano(ctx, repository.CPPorPlanoFilter{
		StartDate:    inicio,
		EndDate:      fim,
		DateField:    campoDeData(dto.DateField),
		Status:       dto.Status,
		FornecedorID: dto.FornecedorID,
	})
}
