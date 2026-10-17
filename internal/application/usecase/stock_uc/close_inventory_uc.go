package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type CloseInventoryUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

// Execute fecha o inventário lançando o ajuste de cada diferença.
//
// Antes o fechamento só trocava a situação para CLOSED: a contagem nunca
// chegava ao saldo, e a conferência "todas as divergências ajustadas" nunca
// disparava porque a diferença não era calculada. As regras (item sem contagem,
// ajuste ao custo médio, transação única) estão no repositório, que trava o
// inventário enquanto lança.
func (uc *CloseInventoryUseCase) Execute(ctx context.Context, id int64, dto request.CloseInventoryDTO) (*response.InventoryCloseResponse, error) {
	if !uc.Auth.CanCloseInventory(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := uc.Repo.GetInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != "OPEN" {
		return nil, errorsuc.NewValidationError("somente inventário aberto pode ser fechado")
	}
	if inv.TotalItems == 0 {
		return nil, errorsuc.NewValidationError("inventário sem itens não pode ser fechado: conte ao menos um item")
	}
	res, err := uc.Repo.FecharInventario(ctx, id, ator, dto.ZerarNaoContados)
	if err != nil {
		return nil, err
	}
	return &response.InventoryCloseResponse{
		Adjustments: res.Ajustes, SurplusQty: res.Sobras, ShortageQty: res.Faltas,
		AdjustedValue: res.ValorAjustado, UncountedZeroed: res.NaoContadosZerados,
	}, nil
}
