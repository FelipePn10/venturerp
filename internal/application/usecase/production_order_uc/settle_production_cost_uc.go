package production_order_uc

import (
	"context"

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
	if qty <= 0 {
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
		if sc, scErr := uc.StdCostRepo.GetItemStandardCost(ctx, order.ItemCode, order.Mask); scErr == nil && sc != nil {
			std = entity.StandardUnitCost{
				Material: sc.MaterialCost,
				Labor:    sc.LaborCost,
				Overhead: sc.OverheadCost,
			}
			if sc.Currency != "" {
				currency = sc.Currency
			}
		}
	}

	settledBy, _ := uc.Auth.UserID(ctx)
	settlement := entity.BuildSettlement(
		productionOrderID,
		entity.ActualCostInputs{ProducedQty: qty, MaterialCostReal: materialReal, LaborCostReal: laborReal},
		std, currency, settledBy,
	)

	return uc.Repo.SettleCost(ctx, settlement)
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
