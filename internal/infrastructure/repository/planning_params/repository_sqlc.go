package planning_params

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *PlanningParamRepositorySQLC) GetByNumber(
	ctx context.Context,
	paramNumber int,
) (*entity.PlanningParam, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetPlanningParamByNumber(ctx, sqlc.GetPlanningParamByNumberParams{ParamNumber: int32(paramNumber), EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("planning param %d not found", paramNumber)
		}
		return nil, fmt.Errorf("fetching planning param: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *PlanningParamRepositorySQLC) GetByKey(
	ctx context.Context,
	key string,
) (*entity.PlanningParam, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetPlanningParamByKey(ctx, sqlc.GetPlanningParamByKeyParams{ParamKey: key, EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("planning param %q not found", key)
		}
		return nil, fmt.Errorf("fetching planning param by key: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *PlanningParamRepositorySQLC) List(ctx context.Context) ([]*entity.PlanningParam, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListPlanningParams(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing planning params: %w", err)
	}
	out := make([]*entity.PlanningParam, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *PlanningParamRepositorySQLC) Update(
	ctx context.Context,
	paramNumber int,
	value string,
	updatedBy uuid.UUID,
) (*entity.PlanningParam, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdatePlanningParam(ctx, sqlc.UpdatePlanningParamParams{
		ParamNumber:  int32(paramNumber),
		Value:        value,
		UpdatedBy:    pgutil.ToPgUUID(updatedBy),
		EnterpriseID: enterpriseID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("planning param %d not found", paramNumber)
		}
		return nil, fmt.Errorf("updating planning param: %w", err)
	}
	return rowToEntity(row), nil
}

func rowToEntity(row sqlc.PlanningParam) *entity.PlanningParam {
	return &entity.PlanningParam{
		ID:          row.ID,
		ParamNumber: int(row.ParamNumber),
		ParamKey:    row.ParamKey,
		Value:       row.Value,
		Description: pgutil.FromPgTextPtr(row.Description),
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:   pgutil.FromPgTimestamptz(row.UpdatedAt),
		UpdatedBy:   pgutil.FromPgUUID(row.UpdatedBy),
	}
}
