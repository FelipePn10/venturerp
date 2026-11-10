package routing

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5/pgtype"
)

// ListRouteMaterials: componentes da estrutura do item do roteiro e a etapa
// em que cada um entra.
func (r *RoutingRepositorySQLC) ListRouteMaterials(ctx context.Context, routeID int64) ([]entity.RouteMaterial, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRouteMaterials(ctx, sqlc.ListRouteMaterialsParams{ID: routeID, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("lendo os componentes do roteiro %d: %w", routeID, err)
	}
	out := make([]entity.RouteMaterial, 0, len(rows))
	for _, row := range rows {
		m := entity.RouteMaterial{
			StructureID: row.StructureID, ChildCode: row.ChildCode, ChildBusiness: row.ChildBusinessCode,
			ChildName: row.ChildName, Quantity: row.Quantity, Unit: row.UnitOfMeasurement,
			ParentMask: textToPtr(row.ParentMask), IsCoproduct: row.IsCoproduct,
		}
		if row.RouteOperationSequence.Valid {
			v := row.RouteOperationSequence.Int16
			m.StepSequence = &v
		}
		out = append(out, m)
	}
	return out, nil
}

// SetMaterialStep grava em que etapa do roteiro o componente entra.
func (r *RoutingRepositorySQLC) SetMaterialStep(ctx context.Context, routeID, structureID int64, step *int16) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	etapa := pgutilInt2(step)
	n, err := r.q.SetStructureRouteStep(ctx, sqlc.SetStructureRouteStepParams{
		StepSequence: etapa, StructureID: structureID, RouteID: routeID, EnterpriseID: empresa,
	})
	return afetou(n, err, "componente não pertence à estrutura do item deste roteiro")
}

// RouteStepActuals: planejado × apontado das OPs concluídas, por etapa.
func (r *RoutingRepositorySQLC) RouteStepActuals(ctx context.Context, routeID int64, dias int) ([]entity.StepActuals, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.RouteStepActuals(ctx, sqlc.RouteStepActualsParams{RouteID: routeID, EnterpriseID: empresa, Days: int32(dias)})
	if err != nil {
		return nil, fmt.Errorf("lendo o apontado das etapas do roteiro %d: %w", routeID, err)
	}
	out := make([]entity.StepActuals, 0, len(rows))
	for _, row := range rows {
		if row.RouteOperationID == nil {
			continue
		}
		a := entity.StepActuals{
			RouteOperationID: *row.RouteOperationID, Orders: int(row.Orders),
			ActualHours:  pgutil.FromPgNumericToFloat64(row.ActualHours),
			PlannedHours: pgutil.FromPgNumericToFloat64(row.PlannedHours),
			SetupHours:   pgutil.FromPgNumericToFloat64(row.SetupHours),
			Quantity:     pgutil.FromPgNumericToFloat64(row.Quantity),
		}
		if row.LastCompleted.Valid {
			t := row.LastCompleted.Time
			a.LastCompleted = &t
		}
		out = append(out, a)
	}
	return out, nil
}

func pgutilInt2(v *int16) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: *v, Valid: true}
}
