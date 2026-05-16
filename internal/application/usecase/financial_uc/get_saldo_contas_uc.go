package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetSaldoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

type SaldoContasResponse struct {
	SaldoConsolidado float64                        `json:"saldo_consolidado"`
	Contas           []SaldoContaItem               `json:"contas"`
}

type SaldoContaItem struct {
	ContaID int64   `json:"conta_id"`
	Saldo   float64 `json:"saldo"`
}

func (uc *GetSaldoContasUseCase) Execute(ctx context.Context) (*SaldoContasResponse, error) {
	if !uc.Auth.CanGetSaldoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	saldoConsolidado, err := uc.Repo.GetSaldoConsolidado(ctx)
	if err != nil {
		return nil, err
	}

	contas, err := uc.Repo.ListContasBancarias(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]SaldoContaItem, 0, len(contas))
	for _, c := range contas {
		saldo, _ := uc.Repo.GetSaldoConta(ctx, c.ID)
		items = append(items, SaldoContaItem{
			ContaID: c.ID,
			Saldo:   saldo,
		})
	}

	return &SaldoContasResponse{
		SaldoConsolidado: saldoConsolidado,
		Contas:           items,
	}, nil
}
