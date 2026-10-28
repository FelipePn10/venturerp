package production_order_uc

import (
	"context"
	"fmt"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	stdrepo "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
)

// SettleProductionCostUseCase apura o custo real da OF (material a custo médio +
// conversão por horas apontadas × custo/hora do CT), compara com o custo padrão
// do item e grava as variâncias. Idempotente: reexecutar recalcula a apuração.
type SettleProductionCostUseCase struct {
	Repo        repository.ProductionOrderRepository
	Auth        ports.AuthService
	StdCostRepo stdrepo.StandardCostRepository
}

func (uc *SettleProductionCostUseCase) Execute(ctx context.Context, productionOrderID int64) (*entity.ProductionOrderCost, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	order, err := uc.Repo.GetByCode(ctx, productionOrderID)
	if err != nil {
		return nil, err
	}

	qty := order.ProducedQty
	if !qty.IsPositive() {
		qty = order.PlannedQty
	}

	materialReal, laborReal, err := uc.Repo.ComputeActualCostInputs(ctx, productionOrderID)
	if err != nil {
		return nil, err
	}

	// Standard unit cost is optional: items without a calculated standard simply
	// settle with zero standard (variance equals the full actual cost).
	var std entity.StandardUnitCost
	currency := "BRL"
	if uc.StdCostRepo != nil {
		// Item sem custo-padrão apura com padrão zero; falha de leitura é erro
		// (antes também apurava com zero, e a variação saía inflada).
		sc, scErr := uc.StdCostRepo.GetItemStandardCost(ctx, order.ItemCode, order.Mask)
		if scErr != nil && !errorsuc.EhNaoEncontrado(scErr) {
			return nil, fmt.Errorf("lendo o custo-padrão do item: %w", scErr)
		}
		if scErr == nil && sc != nil {
			std = entity.StandardUnitCost{
				Material: decimal.NewFromFloat(sc.MaterialCost),
				Labor:    decimal.NewFromFloat(sc.LaborCost),
				Overhead: decimal.NewFromFloat(sc.OverheadCost),
			}
			if sc.Currency != "" {
				currency = sc.Currency
			}
		}
	}

	settledBy, errUsuario := uc.Auth.UserID(ctx)
	if errUsuario != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	settlement := entity.BuildSettlement(
		productionOrderID,
		entity.ActualCostInputs{ProducedQty: qty, MaterialCostReal: materialReal, LaborCostReal: laborReal},
		std, currency, settledBy,
	)

	custo, err := uc.Repo.SettleCost(ctx, settlement)
	if err != nil {
		return nil, err
	}
	// Custo apurado: a pendência do encerramento (se havia) deixa de valer.
	if m, ok := uc.Repo.(custoPendenteRepo); ok {
		if err := m.MarcarCustoPendente(ctx, productionOrderID, nil); err != nil {
			return nil, err
		}
	}
	return custo, nil
}

// GetProductionCostUseCase reads the cost settlement of a production order.
type GetProductionCostUseCase struct {
	Repo repository.ProductionOrderRepository
	Auth ports.AuthService
}

func (uc *GetProductionCostUseCase) Execute(ctx context.Context, productionOrderID int64) (*entity.ProductionOrderCost, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetCost(ctx, productionOrderID)
}
