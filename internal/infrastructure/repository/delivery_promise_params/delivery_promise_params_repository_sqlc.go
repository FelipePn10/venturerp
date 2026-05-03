package delivery_promise_params

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *DeliveryPromiseParamsRepositorySQLC) Get(
	ctx context.Context,
) (*entity.DeliveryPromiseParams, error) {

	row, err := r.q.GetDeliveryPromiseParams(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no delivery promise params configured")
		}
		return nil, fmt.Errorf("fetching delivery promise params: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryPromiseParamsRepositorySQLC) Save(
	ctx context.Context,
	p *entity.DeliveryPromiseParams,
) (*entity.DeliveryPromiseParams, error) {

	existing, err := r.q.GetDeliveryPromiseParams(ctx)

	if err != nil || existing.ID == 0 {
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
			},
		)
		if err != nil {
			return nil, fmt.Errorf("saving delivery promise params: %w", err)
		}

		return rowToEntity(row), nil
	}

	row, err := r.q.UpdateDeliveryPromiseParams(
		ctx,
		sqlc.UpdateDeliveryPromiseParamsParams{
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
			ID:                      existing.ID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("updating delivery promise params: %w", err)
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
