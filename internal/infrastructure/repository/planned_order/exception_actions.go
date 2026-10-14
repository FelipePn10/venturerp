package planned_order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// Reschedule move a necessidade da ordem planejada (e início/fim junto, para
// manter o prazo). É o que a ação "Reprogramar" da exceção do MRP faz.
func (r *PlannedOrderRepositorySQLC) Reschedule(ctx context.Context, code int64, needDate time.Time) (*entity.PlannedOrder, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.ReschedulePlannedOrder(ctx, sqlc.ReschedulePlannedOrderParams{
		NeedDate: toPgDatePtr(&needDate), Code: code, EnterpriseID: enterpriseID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ordem planejada %d não encontrada ou já cancelada", code))
		}
		return nil, fmt.Errorf("reprogramando a ordem planejada: %w", err)
	}
	return rowToEntity(row), nil
}

// Cancel cancela a ordem planejada: sai do suprimento firme do próximo cálculo.
func (r *PlannedOrderRepositorySQLC) Cancel(ctx context.Context, code int64) (*entity.PlannedOrder, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CancelPlannedOrder(ctx, sqlc.CancelPlannedOrderParams{Code: code, EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ordem planejada %d não encontrada ou já cancelada", code))
		}
		return nil, fmt.Errorf("cancelando a ordem planejada: %w", err)
	}
	return rowToEntity(row), nil
}

// ActiveProductionOrderNumber devolve o número da OF não cancelada gerada pela
// ordem planejada, ou nil quando ela ainda não virou OF.
func (r *PlannedOrderRepositorySQLC) ActiveProductionOrderNumber(ctx context.Context, code int64) (*int64, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	n, err := r.q.GetPlannedOrderActiveProductionOrder(ctx, sqlc.GetPlannedOrderActiveProductionOrderParams{Code: code, EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}
