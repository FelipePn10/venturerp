package cost_center

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type CostCenterRepositorySQLC struct {
	q *sqlc.Queries
}

func NewCostCenterRepositorySQLC(q *sqlc.Queries) *CostCenterRepositorySQLC {
	return &CostCenterRepositorySQLC{q: q}
}

func (r *CostCenterRepositorySQLC) Create(
	ctx context.Context,
	cc *entity.CostCenter,
) (*entity.CostCenter, error) {

	row, err := r.q.CreateCostCenter(ctx, sqlc.CreateCostCenterParams{
		Code:        cc.Code,
		Description: cc.Description,
		ParentCode:  cc.ParentCode,
		Type:        sqlc.TypeCcEnum(cc.Type),
		IsRatio:     cc.IsRatio,
		StartDate:   cc.StartDate,
		EndDate:     cc.EndDate,
		CreatedBy:   cc.CreatedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("creating cost center: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *CostCenterRepositorySQLC) Update(
	ctx context.Context,
	cc *entity.CostCenter,
) (*entity.CostCenter, error) {

	row, err := r.q.UpdateCostCenter(ctx, sqlc.UpdateCostCenterParams{
		Description: cc.Description,
		ParentCode:  cc.ParentCode,
		Type:        sqlc.TypeCcEnum(cc.Type),
		IsRatio:     cc.IsRatio,
		StartDate:   cc.StartDate,
		EndDate:     cc.EndDate,
		ID:          cc.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating cost center: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *CostCenterRepositorySQLC) GetByCode(
	ctx context.Context,
	code int32,
) (*entity.CostCenter, error) {

	row, err := r.q.GetCostCenterByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("cost center %d not found", code)
		}

		return nil, fmt.Errorf("fetching cost center by code: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *CostCenterRepositorySQLC) List(ctx context.Context) ([]*entity.CostCenter, error) {
	rows, err := r.q.ListCostCenters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing cost centers: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *CostCenterRepositorySQLC) ListByType(
	ctx context.Context,
	ccType string,
) ([]*entity.CostCenter, error) {

	rows, err := r.q.ListCostCentersByType(ctx, sqlc.TypeCcEnum(ccType))
	if err != nil {
		return nil, fmt.Errorf("listing cost centers by type: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *CostCenterRepositorySQLC) Delete(ctx context.Context, code int32) error {
	return r.q.DeleteCostCenter(ctx, code)
}

func rowToEntity(row sqlc.CostCenter) *entity.CostCenter {
	return &entity.CostCenter{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		ParentCode:  row.ParentCode,
		Type:        types.TypeCC(row.Type),
		IsRatio:     row.IsRatio,
		StartDate:   row.StartDate,
		EndDate:     row.EndDate,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		CreatedBy:   row.CreatedBy,
	}
}

func rowsToEntities(rows []sqlc.CostCenter) []*entity.CostCenter {
	out := make([]*entity.CostCenter, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}
