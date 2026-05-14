package sales_division

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
)

func (r *SalesDivisionRepositorySQLC) Create(
	ctx context.Context,
	sd *entity.SalesDivision,
) (*entity.SalesDivision, error) {
	row, err := r.q.CreateSalesDivision(ctx, sqlc.CreateSalesDivisionParams{
		Code:                    sd.Code,
		Description:             sd.Description,
		CommercialAnalysis:      sqlc.SalesDivisionAnalysisEnum(sd.CommercialAnalysis),
		FinancialAnalysis:       sqlc.SalesDivisionAnalysisEnum(sd.FinancialAnalysis),
		IsTechnicalAssistance:   sd.IsTechnicalAssistance,
		ConsiderDeliveryPromise: sd.ConsiderDeliveryPromise,
		ConsiderMrp:             sd.ConsiderMRP,
		AllowOutsideLimits:      sd.AllowOutsideLimits,
		MinimumDeliveryDays:     int32(sd.MinimumDeliveryDays),
		FinancialDelayDays:      int32(sd.FinancialDelayDays),
		PisPercentage:           pgutil.ToPgNumericFromFloat64(sd.PISPercentage),
		CofinsPercentage:        pgutil.ToPgNumericFromFloat64(sd.CofinsPercentage),
		ParentDivisionID:        sd.ParentDivisionID,
		CreatedBy:               pgutil.ToPgUUID(sd.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales division: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesDivisionRepositorySQLC) Update(
	ctx context.Context,
	sd *entity.SalesDivision,
) (*entity.SalesDivision, error) {
	row, err := r.q.UpdateSalesDivision(ctx, sqlc.UpdateSalesDivisionParams{
		Code:                    sd.Code,
		Description:             sd.Description,
		CommercialAnalysis:      sqlc.SalesDivisionAnalysisEnum(sd.CommercialAnalysis),
		FinancialAnalysis:       sqlc.SalesDivisionAnalysisEnum(sd.FinancialAnalysis),
		IsTechnicalAssistance:   sd.IsTechnicalAssistance,
		ConsiderDeliveryPromise: sd.ConsiderDeliveryPromise,
		ConsiderMrp:             sd.ConsiderMRP,
		AllowOutsideLimits:      sd.AllowOutsideLimits,
		MinimumDeliveryDays:     int32(sd.MinimumDeliveryDays),
		FinancialDelayDays:      int32(sd.FinancialDelayDays),
		PisPercentage:           pgutil.ToPgNumericFromFloat64(sd.PISPercentage),
		CofinsPercentage:        pgutil.ToPgNumericFromFloat64(sd.CofinsPercentage),
		ParentDivisionID:        sd.ParentDivisionID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating sales division: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesDivisionRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.SalesDivision, error) {
	row, err := r.q.GetSalesDivisionByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("sales division with code %d not found", code)
		}
		return nil, fmt.Errorf("fetching sales division: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesDivisionRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.SalesDivision, error) {
	rows, err := r.q.ListSalesDivisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sales divisions: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesDivisionRepositorySQLC) ListActive(
	ctx context.Context,
) ([]*entity.SalesDivision, error) {
	rows, err := r.q.ListActiveSalesDivisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing active sales divisions: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesDivisionRepositorySQLC) Delete(
	ctx context.Context,
	code int64,
) error {
	err := r.q.DeleteSalesDivision(ctx, code)
	if err != nil {
		return fmt.Errorf("deleting sales division %d: %w", code, err)
	}
	return nil
}

func rowToEntity(row sqlc.SalesDivision) *entity.SalesDivision {
	return &entity.SalesDivision{
		ID:                      row.ID,
		Code:                    row.Code,
		Description:             row.Description,
		CommercialAnalysis:      entity.SalesDivisionAnalysis(row.CommercialAnalysis),
		FinancialAnalysis:       entity.SalesDivisionAnalysis(row.FinancialAnalysis),
		IsTechnicalAssistance:   row.IsTechnicalAssistance,
		ConsiderDeliveryPromise: row.ConsiderDeliveryPromise,
		ConsiderMRP:             row.ConsiderMrp,
		AllowOutsideLimits:      row.AllowOutsideLimits,
		MinimumDeliveryDays:     int(row.MinimumDeliveryDays),
		FinancialDelayDays:      int(row.FinancialDelayDays),
		PISPercentage:           pgutil.FromPgNumericToFloat64(row.PisPercentage),
		CofinsPercentage:        pgutil.FromPgNumericToFloat64(row.CofinsPercentage),
		ParentDivisionID:        row.ParentDivisionID,
		IsActive:                row.IsActive,
		CreatedAt:               pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:               pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:               pgutil.FromPgUUID(row.CreatedBy),
	}
}

func rowsToEntities(rows []sqlc.SalesDivision) []*entity.SalesDivision {
	out := make([]*entity.SalesDivision, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}
