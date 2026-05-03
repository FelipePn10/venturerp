package independent_demand

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

func (r *IndependentDemandRepositorySQLC) Create(
	ctx context.Context,
	d *entity.IndependentDemand,
) (*entity.IndependentDemand, error) {

	row, err := r.q.CreateIndependentDemand(ctx, sqlc.CreateIndependentDemandParams{
		Code:           d.CodeDemand,
		ItemCode:       d.ItemCode,
		Mask:           pgutil.ToPgTextFromPtr(d.Mask),
		CostCenterCode: pgutil.ToPgInt8Ptr(d.CostCenterCode),
		Quantity:       pgutil.ToPgNumericFromFloat64(d.Quantity),
		DemandDate:     pgutil.ToPgDate(d.DemandDate),
		CreatedBy:      pgutil.ToPgUUID(d.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating independent demand: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndependentDemandRepositorySQLC) Update(
	ctx context.Context,
	d *entity.IndependentDemand,
) (*entity.IndependentDemand, error) {

	row, err := r.q.UpdateIndependentDemand(ctx, sqlc.UpdateIndependentDemandParams{
		ItemCode:       d.ItemCode,
		Mask:           pgutil.ToPgTextFromPtr(d.Mask),
		CostCenterCode: pgutil.ToPgInt8Ptr(d.CostCenterCode),
		Quantity:       pgutil.ToPgNumericFromFloat64(d.Quantity),
		DemandDate:     pgutil.ToPgDate(d.DemandDate),
		Code:           d.CodeDemand,
	})
	if err != nil {
		return nil, fmt.Errorf("updating independent demand: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndependentDemandRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.IndependentDemand, error) {

	row, err := r.q.GetIndependentDemandByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("independent demand code %d not found", code)
		}

		return nil, fmt.Errorf("fetching independent demand: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *IndependentDemandRepositorySQLC) ListFromDate(
	ctx context.Context,
	date time.Time,
) ([]*entity.IndependentDemand, error) {

	rows, err := r.q.ListDemandsFromDate(ctx, pgutil.ToPgDate(date))
	if err != nil {
		return nil, fmt.Errorf("listing demands from date: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *IndependentDemandRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.IndependentDemand, error) {

	rows, err := r.q.ListIndependentDemands(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing independent demands: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *IndependentDemandRepositorySQLC) ListByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.IndependentDemand, error) {

	rows, err := r.q.ListDemandsByItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing independent demands by item: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *IndependentDemandRepositorySQLC) Delete(
	ctx context.Context,
	code int64,
) error {

	err := r.q.DeleteIndependentDemand(ctx, code)
	if err != nil {
		return fmt.Errorf("deleting independent demand %d: %w", code, err)
	}

	return nil
}

func rowToEntity(row sqlc.IndependentDemand) *entity.IndependentDemand {
	e := &entity.IndependentDemand{
		CodeDemand: row.Code,
		ItemCode:   row.ItemCode,
		Quantity:   pgutil.FromPgNumericToFloat64(row.Quantity),
		DemandDate: pgutil.FromPgDate(row.DemandDate),
		IsActive:   row.IsActive,
		CreatedAt:  pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:  pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:  pgutil.FromPgUUID(row.CreatedBy),
	}

	if row.Mask.Valid {
		v := row.Mask.String
		e.Mask = &v
	}

	if row.CostCenterCode.Valid {
		v := row.CostCenterCode.Int64
		e.CostCenterCode = &v
	}

	return e
}

func rowsToEntities(
	rows []sqlc.IndependentDemand,
) []*entity.IndependentDemand {

	out := make([]*entity.IndependentDemand, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}
