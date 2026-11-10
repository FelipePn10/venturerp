package production_order_uc

import (
	"context"
	"fmt"

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

	// O real aberto pelos componentes do padrão (máquina, mão de obra,
	// terceiros); repositório antigo só sabe material e conversão somada.
	var real entity.ActualCostInputs
	if r, ok := uc.Repo.(custoRealPorComponente); ok {
		real, err = r.CustoRealDaOrdem(ctx, productionOrderID)
	} else {
		real.MaterialCostReal, real.LaborCostReal, err = uc.Repo.ComputeActualCostInputs(ctx, productionOrderID)
	}
	if err != nil {
		return nil, err
	}
	real.ProducedQty = qty

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
			// Todos os componentes do padrão: antes preparação, máquina e
			// terceiros ficavam de fora e a variação comparava com um padrão menor.
			std = entity.StandardUnitCost{
				Material:    sc.MaterialCost,
				Setup:       sc.SetupCost,
				Machine:     sc.MachineCost,
				Labor:       sc.LaborCost,
				Subcontract: sc.SubcontractCost,
				Overhead:    sc.OverheadCost,
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
		real,
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

type custoRealPorComponente interface {
	CustoRealDaOrdem(ctx context.Context, productionOrderID int64) (entity.ActualCostInputs, error)
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

// ordensApuradas lista as OFs da empresa que já têm apuração de custo.
type ordensApuradas interface {
	OrdensComCustoApurado(ctx context.Context) ([]int64, error)
}

// ReapuracaoResultado é o resumo da reapuração em lote.
type ReapuracaoResultado struct {
	Reapuradas int      `json:"reapuradas"`
	Falhas     []string `json:"falhas,omitempty"`
}

// ReapurarTodas refaz a apuração de todas as OFs já apuradas da empresa pela
// regra atual (componentes do padrão, migração 480). A apuração antiga não tinha
// máquina nem terceiros: sem reapurar, a OF antiga mostrava zero nesses
// componentes. Falha numa OF não interrompe as outras; volta no resumo.
func (uc *SettleProductionCostUseCase) ReapurarTodas(ctx context.Context) (*ReapuracaoResultado, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	leitor, ok := uc.Repo.(ordensApuradas)
	if !ok {
		return nil, errorsuc.NewValidationError("a reapuração em lote não está disponível")
	}
	ids, err := leitor.OrdensComCustoApurado(ctx)
	if err != nil {
		return nil, err
	}
	out := &ReapuracaoResultado{}
	for _, id := range ids {
		if _, err := uc.Execute(ctx, id); err != nil {
			out.Falhas = append(out.Falhas, fmt.Sprintf("OF %d: %s", id, err.Error()))
			continue
		}
		out.Reapuradas++
	}
	return out, nil
}
