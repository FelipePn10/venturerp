package production_plan

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *ProductionPlanRepositorySQLC) ReplaceInterFactories(ctx context.Context, planCode int64, entries []*entity.InterFactoryEnterprise) ([]*entity.InterFactoryEnterprise, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	codes := make([]int64, len(entries))
	releases := make([]bool, len(entries))
	for i, entry := range entries {
		codes[i], releases[i] = entry.EnterpriseCode, entry.AutoRelease
	}
	rows, err := r.q.ReplaceProductionPlanInterFactories(ctx, sqlc.ReplaceProductionPlanInterFactoriesParams{PlanCode: planCode, EnterpriseID: enterpriseID, EnterpriseCodes: codes, AutoReleases: releases})
	if err != nil {
		return nil, fmt.Errorf("replacing production plan inter-factories: %w", err)
	}
	return interFactoryRows(rows), nil
}

func (r *ProductionPlanRepositorySQLC) ListInterFactories(ctx context.Context, planCode int64) ([]*entity.InterFactoryEnterprise, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListProductionPlanInterFactories(ctx, sqlc.ListProductionPlanInterFactoriesParams{PlanCode: planCode, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, fmt.Errorf("listing production plan inter-factories: %w", err)
	}
	out := make([]*entity.InterFactoryEnterprise, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.InterFactoryEnterprise{EnterpriseCode: int64(row.EnterpriseCode), EnterpriseName: row.EnterpriseName, AutoRelease: row.AutoRelease})
	}
	return out, nil
}

func interFactoryRows(rows []sqlc.ReplaceProductionPlanInterFactoriesRow) []*entity.InterFactoryEnterprise {
	out := make([]*entity.InterFactoryEnterprise, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.InterFactoryEnterprise{EnterpriseCode: int64(row.EnterpriseCode), EnterpriseName: row.EnterpriseName, AutoRelease: row.AutoRelease})
	}
	return out
}
