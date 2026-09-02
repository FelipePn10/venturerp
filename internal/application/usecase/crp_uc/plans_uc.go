package crp_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// planCatalog é a leitura do catálogo de planos usada pelo modal de CRP.
type planCatalog interface {
	ListCRPPlans(ctx context.Context, enterpriseID int64, onlyCalculated bool) ([]sqlc.CRPPlanRow, error)
	CRPPlanBelongsToEnterprise(ctx context.Context, planCode, enterpriseID int64) (bool, error)
}

// WithPlanCatalog habilita a listagem de planos do modal e o isolamento por
// empresa nas consultas de carga.
func (uc *CRPUseCase) WithPlanCatalog(c planCatalog) *CRPUseCase {
	uc.plans = c
	return uc
}

// ListPlans devolve os planos da empresa com o resumo da carga calculada. Com
// onlyCalculated, esconde os planos que ainda não têm CRP — são justamente os
// que devolveriam uma exportação vazia.
func (uc *CRPUseCase) ListPlans(ctx context.Context, onlyCalculated bool) ([]response.CRPPlanResponse, error) {
	if uc.plans == nil {
		return []response.CRPPlanResponse{}, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := uc.plans.ListCRPPlans(ctx, enterpriseID, onlyCalculated)
	if err != nil {
		return nil, fmt.Errorf("listando planos de CRP: %w", err)
	}
	out := make([]response.CRPPlanResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.CRPPlanResponse{
			PlanCode:       row.PlanCode,
			Name:           row.Name,
			TotalEntries:   row.TotalEntries,
			OverloadCount:  row.OverloadCount,
			MaxLoadPct:     row.MaxLoadPct,
			Calculated:     row.TotalEntries > 0,
			LastCalculated: row.LastCalculated,
		})
	}
	return out, nil
}

// assertPlanTenant recusa a leitura de um plano de outra empresa. Sem catálogo
// configurado a checagem é dispensada (o comportamento anterior é preservado).
func (uc *CRPUseCase) assertPlanTenant(ctx context.Context, planCode int64) error {
	if uc.plans == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	ok, err := uc.plans.CRPPlanBelongsToEnterprise(ctx, planCode, enterpriseID)
	if err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewNotFoundError(fmt.Sprintf("plano %d não encontrado nesta empresa", planCode))
	}
	return nil
}
