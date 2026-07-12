package production_plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *ProductionPlanRepositorySQLC) Create(
	ctx context.Context,
	plan *entity.ProductionPlan,
) (*entity.ProductionPlan, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := json.Marshal(plan.Parameters)
	if err != nil {
		return nil, fmt.Errorf("marshaling parameters: %w", err)
	}

	row, err := r.q.CreateProductionPlan(ctx, sqlc.CreateProductionPlanParams{
		Code:                plan.Code,
		Name:                plan.Name,
		IndependentDemands:  plan.IndependentDemands,
		GroupSameDateOrders: plan.GroupSameDateOrders,
		PlanningTypes:       plan.PlanningTypes,
		Classification:      pgutil.ToPgTextFromPtr(plan.Classification),
		ClassItemCodes:      pgutil.ToPgTextFromPtr(plan.ClassItemCodes),
		OrderItemCode:       plan.OrderItemCode,
		Parameters:          paramsJSON,
		CreatedBy:           pgutil.ToPgUUID(plan.CreatedBy),
		EnterpriseID:        enterpriseID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "production_plans_code_key" {
			return nil, repository.ErrAlreadyExists
		}
		return nil, fmt.Errorf("creating production plan: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *ProductionPlanRepositorySQLC) Update(
	ctx context.Context,
	plan *entity.ProductionPlan,
) (*entity.ProductionPlan, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	paramsJSON, err := json.Marshal(plan.Parameters)
	if err != nil {
		return nil, fmt.Errorf("marshaling parameters: %w", err)
	}

	row, err := r.q.UpdateProductionPlan(ctx, sqlc.UpdateProductionPlanParams{
		Code:                plan.Code,
		Name:                plan.Name,
		IndependentDemands:  plan.IndependentDemands,
		GroupSameDateOrders: plan.GroupSameDateOrders,
		PlanningTypes:       plan.PlanningTypes,
		Classification:      pgutil.ToPgTextFromPtr(plan.Classification),
		ClassItemCodes:      pgutil.ToPgTextFromPtr(plan.ClassItemCodes),
		OrderItemCode:       plan.OrderItemCode,
		Parameters:          paramsJSON,
		EnterpriseID:        enterpriseID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("production plan %d not found", plan.Code)
		}
		return nil, fmt.Errorf("updating production plan: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *ProductionPlanRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.ProductionPlan, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetProductionPlanByCode(ctx, sqlc.GetProductionPlanByCodeParams{Code: code, EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("production plan %d not found", code)
		}
		return nil, fmt.Errorf("fetching production plan: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *ProductionPlanRepositorySQLC) List(ctx context.Context) ([]*entity.ProductionPlan, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListProductionPlans(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing production plans: %w", err)
	}
	out := make([]*entity.ProductionPlan, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *ProductionPlanRepositorySQLC) Delete(ctx context.Context, code int64) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	if err := r.q.DeleteProductionPlan(ctx, sqlc.DeleteProductionPlanParams{Code: code, EnterpriseID: enterpriseID}); err != nil {
		return fmt.Errorf("deleting production plan %d: %w", code, err)
	}
	return nil
}

func (r *ProductionPlanRepositorySQLC) UpdateLastCalculated(ctx context.Context, code int64) error {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return err
	}
	return r.q.UpdateProductionPlanLastCalculated(ctx, sqlc.UpdateProductionPlanLastCalculatedParams{
		Code: code, EnterpriseID: enterpriseID,
	})
}

func rowToEntity(row sqlc.ProductionPlan) *entity.ProductionPlan {
	params := map[string]interface{}{}
	if len(row.Parameters) > 0 {
		_ = json.Unmarshal(row.Parameters, &params)
	}

	e := &entity.ProductionPlan{
		ID:                  row.ID,
		Code:                row.Code,
		Name:                row.Name,
		IndependentDemands:  row.IndependentDemands,
		GroupSameDateOrders: row.GroupSameDateOrders,
		PlanningTypes:       row.PlanningTypes,
		Classification:      pgutil.FromPgTextPtr(row.Classification),
		ClassItemCodes:      pgutil.FromPgTextPtr(row.ClassItemCodes),
		OrderItemCode:       row.OrderItemCode,
		Parameters:          params,
		IsActive:            row.IsActive,
		CreatedAt:           pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:           pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
	}

	if row.LastCalculatedAt.Valid {
		t := pgutil.FromPgTimestamptz(row.LastCalculatedAt)
		e.LastCalculatedAt = &t
	}

	return e
}
