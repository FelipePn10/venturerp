package sales_forecast

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

// ---- Forecasts ----

func (r *SalesForecastRepositorySQLC) CreateForecast(
	ctx context.Context,
	f *entity.SalesForecast,
) (*entity.SalesForecast, error) {
	row, err := r.q.CreateSalesForecast(ctx, sqlc.CreateSalesForecastParams{
		ItemCode:  f.ItemCode,
		Mask:      pgutil.ToPgTextFromPtr(f.Mask),
		Week:      int32(f.Week),
		Year:      int32(f.Year),
		Quantity:  pgutil.ToPgNumericFromFloat64(f.Quantity),
		CreatedBy: pgutil.ToPgUUID(f.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales forecast: %w", err)
	}
	return forecastRowToEntity(row), nil
}

func (r *SalesForecastRepositorySQLC) UpdateForecast(
	ctx context.Context,
	f *entity.SalesForecast,
) (*entity.SalesForecast, error) {
	row, err := r.q.UpdateSalesForecast(ctx, sqlc.UpdateSalesForecastParams{
		ID:       f.ID,
		Quantity: pgutil.ToPgNumericFromFloat64(f.Quantity),
	})
	if err != nil {
		return nil, fmt.Errorf("updating sales forecast: %w", err)
	}
	return forecastRowToEntity(row), nil
}

func (r *SalesForecastRepositorySQLC) GetForecastByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.SalesForecast, error) {
	rows, err := r.q.GetSalesForecastsByItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("getting forecasts by item %d: %w", itemCode, err)
	}
	return forecastRowsToEntities(rows), nil
}

func (r *SalesForecastRepositorySQLC) ListForecasts(
	ctx context.Context,
	year int,
) ([]*entity.SalesForecast, error) {
	rows, err := r.q.ListSalesForecastsByYear(ctx, int32(year))
	if err != nil {
		return nil, fmt.Errorf("listing forecasts for year %d: %w", year, err)
	}
	return forecastRowsToEntities(rows), nil
}

func (r *SalesForecastRepositorySQLC) DeleteForecast(
	ctx context.Context,
	id int64,
) error {
	err := r.q.DeleteSalesForecast(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting sales forecast %d: %w", id, err)
	}
	return nil
}

// ---- Forecast Blocks ----

func (r *SalesForecastRepositorySQLC) CreateBlock(
	ctx context.Context,
	b *entity.SalesForecastBlock,
) (*entity.SalesForecastBlock, error) {
	row, err := r.q.CreateSalesForecastBlock(ctx, sqlc.CreateSalesForecastBlockParams{
		StartDate: pgutil.ToPgDate(b.StartDate),
		EndDate:   pgutil.ToPgDate(b.EndDate),
		Reason:    pgutil.ToPgTextFromPtr(b.Reason),
		CreatedBy: pgutil.ToPgUUID(b.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating forecast block: %w", err)
	}
	return blockRowToEntity(row), nil
}

func (r *SalesForecastRepositorySQLC) ListBlocks(
	ctx context.Context,
) ([]*entity.SalesForecastBlock, error) {
	rows, err := r.q.ListSalesForecastBlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing forecast blocks: %w", err)
	}
	return blockRowsToEntities(rows), nil
}

func (r *SalesForecastRepositorySQLC) IsBlocked(
	ctx context.Context,
	date time.Time,
) (bool, error) {
	blocked, err := r.q.IsForecastBlocked(ctx, pgutil.ToPgDate(date))
	if err != nil {
		return false, fmt.Errorf("checking if date is blocked: %w", err)
	}
	return blocked, nil
}

func (r *SalesForecastRepositorySQLC) DeleteBlock(
	ctx context.Context,
	id int64,
) error {
	err := r.q.DeleteSalesForecastBlock(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting forecast block %d: %w", id, err)
	}
	return nil
}

// ---- Appropriation Tables ----

func (r *SalesForecastRepositorySQLC) CreateAppropriation(
	ctx context.Context,
	a *entity.AppropriationTable,
) (*entity.AppropriationTable, error) {
	row, err := r.q.CreateAppropriationTable(ctx, sqlc.CreateAppropriationTableParams{
		Description:  a.Description,
		MondayPct:    pgutil.ToPgNumericFromFloat64(a.MondayPct),
		TuesdayPct:   pgutil.ToPgNumericFromFloat64(a.TuesdayPct),
		WednesdayPct: pgutil.ToPgNumericFromFloat64(a.WednesdayPct),
		ThursdayPct:  pgutil.ToPgNumericFromFloat64(a.ThursdayPct),
		FridayPct:    pgutil.ToPgNumericFromFloat64(a.FridayPct),
		SaturdayPct:  pgutil.ToPgNumericFromFloat64(a.SaturdayPct),
		SundayPct:    pgutil.ToPgNumericFromFloat64(a.SundayPct),
		IsDefault:    a.IsDefault,
		CreatedBy:    pgutil.ToPgUUID(a.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating appropriation table: %w", err)
	}
	return appropriationRowToEntity(row), nil
}

func (r *SalesForecastRepositorySQLC) UpdateAppropriation(
	ctx context.Context,
	a *entity.AppropriationTable,
) (*entity.AppropriationTable, error) {
	row, err := r.q.UpdateAppropriationTable(ctx, sqlc.UpdateAppropriationTableParams{
		ID:           a.ID,
		Description:  a.Description,
		MondayPct:    pgutil.ToPgNumericFromFloat64(a.MondayPct),
		TuesdayPct:   pgutil.ToPgNumericFromFloat64(a.TuesdayPct),
		WednesdayPct: pgutil.ToPgNumericFromFloat64(a.WednesdayPct),
		ThursdayPct:  pgutil.ToPgNumericFromFloat64(a.ThursdayPct),
		FridayPct:    pgutil.ToPgNumericFromFloat64(a.FridayPct),
		SaturdayPct:  pgutil.ToPgNumericFromFloat64(a.SaturdayPct),
		SundayPct:    pgutil.ToPgNumericFromFloat64(a.SundayPct),
	})
	if err != nil {
		return nil, fmt.Errorf("updating appropriation table: %w", err)
	}
	return appropriationRowToEntity(row), nil
}

func (r *SalesForecastRepositorySQLC) GetDefaultAppropriation(
	ctx context.Context,
) (*entity.AppropriationTable, error) {
	row, err := r.q.GetDefaultAppropriationTable(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no default appropriation table found")
		}
		return nil, fmt.Errorf("getting default appropriation table: %w", err)
	}
	return appropriationRowToEntity(row), nil
}

func (r *SalesForecastRepositorySQLC) ListAppropriations(
	ctx context.Context,
) ([]*entity.AppropriationTable, error) {
	rows, err := r.q.ListAppropriationTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing appropriation tables: %w", err)
	}
	return appropriationRowsToEntities(rows), nil
}

func (r *SalesForecastRepositorySQLC) SetDefaultAppropriation(
	ctx context.Context,
	id int64,
) error {
	if err := r.q.ClearDefaultAppropriationTable(ctx); err != nil {
		return fmt.Errorf("clearing default appropriation: %w", err)
	}
	if err := r.q.SetSingleDefaultAppropriationTable(ctx, id); err != nil {
		return fmt.Errorf("setting default appropriation %d: %w", id, err)
	}
	return nil
}

// ---- Row mappers ----

func forecastRowToEntity(row sqlc.SalesForecast) *entity.SalesForecast {
	return &entity.SalesForecast{
		ID:        row.ID,
		ItemCode:  row.ItemCode,
		Mask:      pgutil.FromPgTextPtr(row.Mask),
		Week:      int(row.Week),
		Year:      int(row.Year),
		Quantity:  pgutil.FromPgNumericToFloat64(row.Quantity),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func forecastRowsToEntities(rows []sqlc.SalesForecast) []*entity.SalesForecast {
	out := make([]*entity.SalesForecast, 0, len(rows))
	for _, row := range rows {
		out = append(out, forecastRowToEntity(row))
	}
	return out
}

func blockRowToEntity(row sqlc.SalesForecastBlock) *entity.SalesForecastBlock {
	return &entity.SalesForecastBlock{
		ID:        row.ID,
		StartDate: pgutil.FromPgDate(row.StartDate),
		EndDate:   pgutil.FromPgDate(row.EndDate),
		Reason:    pgutil.FromPgTextPtr(row.Reason),
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}

func blockRowsToEntities(rows []sqlc.SalesForecastBlock) []*entity.SalesForecastBlock {
	out := make([]*entity.SalesForecastBlock, 0, len(rows))
	for _, row := range rows {
		out = append(out, blockRowToEntity(row))
	}
	return out
}

func appropriationRowToEntity(row sqlc.AppropriationTable) *entity.AppropriationTable {
	return &entity.AppropriationTable{
		ID:           row.ID,
		Description:  row.Description,
		MondayPct:    pgutil.FromPgNumericToFloat64(row.MondayPct),
		TuesdayPct:   pgutil.FromPgNumericToFloat64(row.TuesdayPct),
		WednesdayPct: pgutil.FromPgNumericToFloat64(row.WednesdayPct),
		ThursdayPct:  pgutil.FromPgNumericToFloat64(row.ThursdayPct),
		FridayPct:    pgutil.FromPgNumericToFloat64(row.FridayPct),
		SaturdayPct:  pgutil.FromPgNumericToFloat64(row.SaturdayPct),
		SundayPct:    pgutil.FromPgNumericToFloat64(row.SundayPct),
		IsDefault:    row.IsDefault,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:    pgutil.FromPgUUID(row.CreatedBy),
	}
}

func appropriationRowsToEntities(rows []sqlc.AppropriationTable) []*entity.AppropriationTable {
	out := make([]*entity.AppropriationTable, 0, len(rows))
	for _, row := range rows {
		out = append(out, appropriationRowToEntity(row))
	}
	return out
}
