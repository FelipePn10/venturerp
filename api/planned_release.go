package main

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc"
	productionuc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	mrprepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order"
	paramsrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params"
	prodrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order"
	reqrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	structurerepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure"
	servicerepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// All mutations from a batch release share one transaction. Ordered row locks
// make simultaneous retries see the committed state before creating an OF.
func wireAtomicPlannedRelease(pool *pgxpool.Pool, q *sqlc.Queries, uc *planned_order_uc.FirmPlannedOrderUseCase, ops *productionuc.OrderOperationsUseCase) {
	uc.TransitionAtomically = func(ctx context.Context, dto request.TransitionPlannedOrderDTO) ([]*response.PlannedOrderResponse, error) {
		enterprise, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		rows, err := tx.Query(ctx, `SELECT id FROM planned_orders WHERE enterprise_id=$1 AND code=ANY($2::bigint[]) ORDER BY id FOR UPDATE`, enterprise, dto.OrderCodes)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		inner := *uc
		inner.TransitionAtomically = nil
		bound := q.WithTx(tx)
		inner.Items = itemrepo.NewRepositoryItemSQLC(bound)
		inner.Structure = structurerepo.NewItemStructureRepository(bound)
		inner.Params = paramsrepo.NewPlanningParamRepositorySQLC(bound)
		production := prodrepo.NewProductionOrderRepositoryPGX(tx)
		inner.OrderOps = ops
		inner.Repo = plannedrepo.NewPlannedOrderRepositorySQLC(bound)
		inner.ProdOrderRepo = production
		inner.ServiceLinker = production
		inner.ReleaseValidator = production
		inner.ReqRepo = reqrepo.New(bound, pool)
		inner.ServiceOrders = servicerepo.New(tx)
		inner.Routing = routingrepo.New(bound)
		inner.ExternalOps = routingrepo.New(bound)
		inner.CreateAtomically = func(ctx context.Context, order *productionentity.ProductionOrder, materials []*productionentity.ProductionOrderMaterial, routeID int64) (*productionentity.ProductionOrder, error) {
			return production.CreateWithMaterialsAndCallback(ctx, order, materials, func(ctx context.Context, nested pgx.Tx, created *productionentity.ProductionOrder) error {
				execution := *ops
				execution.Q = q.WithTx(nested)
				execution.WithinTransaction = nil
				execution.Routing = routingrepo.New(execution.Q)
				execution.MachinePlanning = mrprepo.NewMRPCalculationRepositorySQLC(execution.Q, nested)
				_, err := execution.ExplodeRoute(ctx, created.ID, routeID)
				return err
			})
		}
		result, err := inner.ExecuteTransition(ctx, dto)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return result, nil
	}
}
