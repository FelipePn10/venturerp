package delivery_promise_params

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// Os parâmetros da promessa são da empresa da sessão (um registro por empresa).

func (r *DeliveryPromiseParamsRepositorySQLC) Get(
	ctx context.Context,
) (*entity.DeliveryPromiseParams, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetDeliveryPromiseParams(ctx, empresa)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("os parâmetros de promessa de entrega não estão configurados")
		}
		return nil, fmt.Errorf("fetching delivery promise params: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryPromiseParamsRepositorySQLC) Save(
	ctx context.Context,
	p *entity.DeliveryPromiseParams,
) (*entity.DeliveryPromiseParams, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpsertDeliveryPromiseParams(
		ctx,
		sqlc.UpsertDeliveryPromiseParamsParams{
			UseDeliveryPromise:      p.UseDeliveryPromise,
			BlockedOrdersInPromise:  p.BlockedOrdersInPromise,
			DefaultOrderSort:        p.DefaultOrderSort,
			ShowOrderValues:         int32(p.ShowOrderValues),
			BlockedExportInPromise:  p.BlockedExportInPromise,
			BreakTankOccupation:     p.BreakTankOccupation,
			RecalculateAfterRelease: p.RecalculateAfterRelease,
			ReprogramLoadedOrders:   p.ReprogramLoadedOrders,
			AllowDeliveryDateChange: p.AllowDeliveryDateChange,
			UpdatedBy:               pgutil.ToPgUUID(p.UpdatedBy),
			EnterpriseID:            empresa,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("saving delivery promise params: %w", err)
	}
	return rowToEntity(row), nil
}

func rowToEntity(
	row sqlc.DeliveryPromiseParam,
) *entity.DeliveryPromiseParams {

	return &entity.DeliveryPromiseParams{
		ID:                      row.ID,
		UseDeliveryPromise:      row.UseDeliveryPromise,
		BlockedOrdersInPromise:  row.BlockedOrdersInPromise,
		DefaultOrderSort:        row.DefaultOrderSort,
		ShowOrderValues:         int(row.ShowOrderValues),
		BlockedExportInPromise:  row.BlockedExportInPromise,
		BreakTankOccupation:     row.BreakTankOccupation,
		RecalculateAfterRelease: row.RecalculateAfterRelease,
		ReprogramLoadedOrders:   row.ReprogramLoadedOrders,
		AllowDeliveryDateChange: row.AllowDeliveryDateChange,
		CreatedAt:               pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:               pgutil.FromPgTimestamptz(row.UpdatedAt),
		UpdatedBy:               pgutil.FromPgUUID(row.UpdatedBy),
	}
}
