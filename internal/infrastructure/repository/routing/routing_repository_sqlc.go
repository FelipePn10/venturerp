package routing

import (
	"context"
	"fmt"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
)

type RoutingRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.RoutingRepository {
	return &RoutingRepositorySQLC{q: q}
}

func (r *RoutingRepositorySQLC) CreatedByFromUUID(v uuid.UUID) uuid.UUID { return v }

// ─── operations ──────────────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) CreateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error) {
	row, err := r.q.CreateOperation(ctx, sqlc.CreateOperationParams{
		Code:                 op.Code,
		Name:                 op.Name,
		Description:          pgutil.ToPgTextFromPtr(op.Description),
		Origin:               sqltypes.OperationOriginEnum(op.Origin),
		Situation:            sqltypes.OperationSituationEnum(op.Situation),
		DefaultWorkCenterID:  op.DefaultWorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64(op.CrewSize),
		TimeUnit:             op.TimeUnit,
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromFloat64Ptr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: operationRemittance(op.ThirdPartyRemittance),
		CreatedBy:            pgutil.ToPgUUID(op.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating operation: %w", err)
	}
	return operationRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error) {
	row, err := r.q.UpdateOperation(ctx, sqlc.UpdateOperationParams{
		ID:                   op.ID,
		Name:                 op.Name,
		Description:          pgutil.ToPgTextFromPtr(op.Description),
		Origin:               sqltypes.OperationOriginEnum(op.Origin),
		Situation:            sqltypes.OperationSituationEnum(op.Situation),
		DefaultWorkCenterID:  op.DefaultWorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64(op.CrewSize),
		TimeUnit:             op.TimeUnit,
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromFloat64Ptr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: operationRemittance(op.ThirdPartyRemittance),
	})
	if err != nil {
		return nil, fmt.Errorf("updating operation: %w", err)
	}
	return operationRowToEntity(row), nil
}

func operationRemittance(value string) string {
	if value == "" {
		return "DEMAND_ITEMS"
	}
	return value
}

func (r *RoutingRepositorySQLC) GetOperationByID(ctx context.Context, id int64) (*entity.Operation, error) {
	row, err := r.q.GetOperationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching operation %d: %w", id, err)
	}
	return operationRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) ListOperations(ctx context.Context, onlyActive bool) ([]*entity.Operation, error) {
	rows, err := r.q.ListOperations(ctx, onlyActive)
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}
	out := make([]*entity.Operation, 0, len(rows))
	for _, row := range rows {
		out = append(out, operationRowToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) DeactivateOperation(ctx context.Context, id int64) error {
	return r.q.DeactivateOperation(ctx, id)
}

func (r *RoutingRepositorySQLC) OperationUsedInRoutes(ctx context.Context, id int64) (bool, error) {
	return r.q.OperationUsedInRoutes(ctx, id)
}

func (r *RoutingRepositorySQLC) NextOperationCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextOperationCode(ctx)
	return int64(v), err
}

// ─── manufacturing_routes ─────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) CreateRoute(ctx context.Context, rt *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error) {
	row, err := r.q.CreateRoute(ctx, sqlc.CreateRouteParams{
		Code:        rt.Code,
		ItemCode:    rt.ItemCode,
		Mask:        pgutil.ToPgTextFromPtr(rt.Mask),
		Alternative: rt.Alternative,
		Description: pgutil.ToPgTextFromPtr(rt.Description),
		Situation:   sqltypes.RouteSituationEnum(rt.Situation),
		IsStandard:  rt.IsStandard,
		ValidFrom:   pgutil.ToPgDateFromPtr(rt.ValidFrom),
		ValidTo:     pgutil.ToPgDateFromPtr(rt.ValidTo),
		CreatedBy:   pgutil.ToPgUUID(rt.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating route: %w", err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRoute(ctx context.Context, rt *entity.ManufacturingRoute) (*entity.ManufacturingRoute, error) {
	row, err := r.q.UpdateRoute(ctx, sqlc.UpdateRouteParams{
		ID:          rt.ID,
		Description: pgutil.ToPgTextFromPtr(rt.Description),
		Situation:   sqltypes.RouteSituationEnum(rt.Situation),
		IsStandard:  rt.IsStandard,
		ValidFrom:   pgutil.ToPgDateFromPtr(rt.ValidFrom),
		ValidTo:     pgutil.ToPgDateFromPtr(rt.ValidTo),
	})
	if err != nil {
		return nil, fmt.Errorf("updating route: %w", err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteByID(ctx context.Context, id int64) (*entity.ManufacturingRoute, error) {
	row, err := r.q.GetRouteByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching route %d: %w", id, err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteByItemCode(ctx context.Context, itemCode int64, mask string, alternative int16) (*entity.ManufacturingRoute, error) {
	row, err := r.q.GetRouteByItemAndAlternative(ctx, sqlc.GetRouteByItemAndAlternativeParams{
		ItemCode:    itemCode,
		Mask:        pgutil.ToPgText(mask),
		Alternative: alternative,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching route for item %d: %w", itemCode, err)
	}
	return routeRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) ListRoutesByItem(ctx context.Context, itemCode int64) ([]*entity.ManufacturingRoute, error) {
	rows, err := r.q.ListRoutesByItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing routes for item %d: %w", itemCode, err)
	}
	out := make([]*entity.ManufacturingRoute, 0, len(rows))
	for _, row := range rows {
		out = append(out, routeRowToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) DeactivateRoute(ctx context.Context, id int64) error {
	return r.q.DeactivateRoute(ctx, id)
}

func (r *RoutingRepositorySQLC) NextRouteCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextRouteCode(ctx)
	return int64(v), err
}

func (r *RoutingRepositorySQLC) ItemHasRoute(ctx context.Context, itemCode int64) (bool, error) {
	return r.q.ItemHasRoute(ctx, itemCode)
}

func (r *RoutingRepositorySQLC) GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*entity.ManufacturingRoute, error) {
	row, err := r.q.GetStandardRouteForItem(ctx, sqlc.GetStandardRouteForItemParams{
		ItemCode: itemCode,
		Mask:     pgutil.ToPgText(mask),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching standard route for item %d: %w", itemCode, err)
	}
	return routeRowToEntity(row), nil
}

// ─── route_operations ────────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) AddRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error) {
	row, err := r.q.AddRouteOperation(ctx, sqlc.AddRouteOperationParams{
		RouteID:              op.RouteID,
		Sequence:             op.Sequence,
		OperationID:          op.OperationID,
		WorkCenterID:         op.WorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64Ptr(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64Ptr(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64Ptr(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64Ptr(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64Ptr(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64Ptr(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64Ptr(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64Ptr(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64Ptr(op.CrewSize),
		TimeUnit:             pgutil.ToPgTextFromPtr(op.TimeUnit),
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromFloat64Ptr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: pgutil.ToPgTextFromPtr(op.ThirdPartyRemittance),
		Situation:            sqltypes.RouteOpSituationEnum(op.Situation),
		Notes:                pgutil.ToPgTextFromPtr(op.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("adding route operation: %w", err)
	}
	return routeOpRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error) {
	row, err := r.q.UpdateRouteOperation(ctx, sqlc.UpdateRouteOperationParams{
		ID:                   op.ID,
		WorkCenterID:         op.WorkCenterID,
		StandardTime:         pgutil.ToPgNumericFromFloat64Ptr(op.StandardTime),
		SetupTime:            pgutil.ToPgNumericFromFloat64Ptr(op.SetupTime),
		RunTime:              pgutil.ToPgNumericFromFloat64Ptr(op.RunTime),
		LaborTime:            pgutil.ToPgNumericFromFloat64Ptr(op.LaborTime),
		RunTimeBaseQty:       pgutil.ToPgNumericFromFloat64Ptr(op.RunBaseQty),
		QueueTime:            pgutil.ToPgNumericFromFloat64Ptr(op.QueueTime),
		WaitTime:             pgutil.ToPgNumericFromFloat64Ptr(op.WaitTime),
		MoveTime:             pgutil.ToPgNumericFromFloat64Ptr(op.MoveTime),
		CrewSize:             pgutil.ToPgNumericFromFloat64Ptr(op.CrewSize),
		TimeUnit:             pgutil.ToPgTextFromPtr(op.TimeUnit),
		SupplierID:           op.SupplierID,
		ServiceItemCode:      op.ServiceItemCode,
		CostPerUnit:          pgutil.ToPgNumericFromFloat64Ptr(op.CostPerUnit),
		LeadTimeDays:         op.LeadTimeDays,
		ThirdPartyRemittance: pgutil.ToPgTextFromPtr(op.ThirdPartyRemittance),
		Situation:            sqltypes.RouteOpSituationEnum(op.Situation),
		Notes:                pgutil.ToPgTextFromPtr(op.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("updating route operation: %w", err)
	}
	return routeOpRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteOperations(ctx context.Context, routeID int64) ([]*entity.RouteOperation, error) {
	rows, err := r.q.GetRouteOperations(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching route operations for route %d: %w", routeID, err)
	}
	out := make([]*entity.RouteOperation, 0, len(rows))
	for _, row := range rows {
		out = append(out, routeOpRowWithNamesToEntity(row))
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) RemoveRouteOperation(ctx context.Context, id int64) error {
	return r.q.RemoveRouteOperation(ctx, id)
}

// ─── network edges ────────────────────────────────────────────────────────────

func (r *RoutingRepositorySQLC) SetNetworkEdge(ctx context.Context, edge *entity.NetworkEdge) (*entity.NetworkEdge, error) {
	row, err := r.q.UpsertNetworkEdge(ctx, sqlc.UpsertNetworkEdgeParams{
		PredecessorID: edge.PredecessorID,
		SuccessorID:   edge.SuccessorID,
		OverlapPct:    pgutil.ToPgNumericFromFloat64(edge.OverlapPct),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting network edge: %w", err)
	}
	return &entity.NetworkEdge{
		ID:            row.ID,
		PredecessorID: row.PredecessorID,
		SuccessorID:   row.SuccessorID,
		OverlapPct:    pgutil.FromPgNumericToFloat64(row.OverlapPct),
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
	}, nil
}

func (r *RoutingRepositorySQLC) DeleteNetworkEdge(ctx context.Context, predecessorID, successorID int64) error {
	return r.q.DeleteNetworkEdge(ctx, sqlc.DeleteNetworkEdgeParams{
		PredecessorID: predecessorID,
		SuccessorID:   successorID,
	})
}

func (r *RoutingRepositorySQLC) GetNetworkEdges(ctx context.Context, routeID int64) ([]*entity.NetworkEdge, error) {
	rows, err := r.q.GetNetworkEdges(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching network edges for route %d: %w", routeID, err)
	}
	out := make([]*entity.NetworkEdge, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.NetworkEdge{
			ID:            row.ID,
			PredecessorID: row.PredecessorID,
			SuccessorID:   row.SuccessorID,
			OverlapPct:    pgutil.FromPgNumericToFloat64(row.OverlapPct),
			CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
		})
	}
	return out, nil
}

// ─── route operation resources (alternatives) ─────────────────────────────────

func (r *RoutingRepositorySQLC) AddRouteOpResource(ctx context.Context, res *entity.RouteOpResource) (*entity.RouteOpResource, error) {
	row, err := r.q.AddRouteOpResource(ctx, sqlc.AddRouteOpResourceParams{
		RouteOperationID: res.RouteOperationID,
		WorkCenterID:     res.WorkCenterID,
		Priority:         res.Priority,
		TimeFactor:       pgutil.ToPgNumericFromFloat64(res.TimeFactor),
		IsPrimary:        res.IsPrimary,
	})
	if err != nil {
		return nil, fmt.Errorf("adding route op resource: %w", err)
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRouteOpResource(ctx context.Context, res *entity.RouteOpResource) (*entity.RouteOpResource, error) {
	row, err := r.q.UpdateRouteOpResource(ctx, sqlc.UpdateRouteOpResourceParams{
		ID:         res.ID,
		Priority:   res.Priority,
		TimeFactor: pgutil.ToPgNumericFromFloat64(res.TimeFactor),
	})
	if err != nil {
		return nil, fmt.Errorf("updating route op resource: %w", err)
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) GetRouteOpResource(ctx context.Context, id int64) (*entity.RouteOpResource, error) {
	row, err := r.q.GetRouteOpResource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching route op resource %d: %w", id, err)
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) RemoveRouteOpResource(ctx context.Context, id int64) error {
	return r.q.RemoveRouteOpResource(ctx, id)
}

// SetRouteOpResourcePrimary makes one resource the primary for its operation:
// it clears any other primary, flags this one, and mirrors the choice onto
// route_operations.work_center_id (so cost/CRP/lead-time use it).
func (r *RoutingRepositorySQLC) SetRouteOpResourcePrimary(ctx context.Context, id, routeOperationID, workCenterID int64) (*entity.RouteOpResource, error) {
	if err := r.q.ClearPrimaryResources(ctx, routeOperationID); err != nil {
		return nil, fmt.Errorf("clearing primary resources: %w", err)
	}
	row, err := r.q.SetResourcePrimary(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("setting resource primary: %w", err)
	}
	if err := r.q.SetRouteOpWorkCenter(ctx, sqlc.SetRouteOpWorkCenterParams{
		ID:           routeOperationID,
		WorkCenterID: &workCenterID,
	}); err != nil {
		return nil, fmt.Errorf("syncing route op work center: %w", err)
	}
	return resourceRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) ListResourcesByRouteOp(ctx context.Context, routeOperationID int64) ([]*entity.RouteOpResource, error) {
	rows, err := r.q.ListResourcesByRouteOp(ctx, routeOperationID)
	if err != nil {
		return nil, fmt.Errorf("listing resources for route op %d: %w", routeOperationID, err)
	}
	out := make([]*entity.RouteOpResource, 0, len(rows))
	for _, row := range rows {
		e := &entity.RouteOpResource{
			ID:               row.ID,
			RouteOperationID: row.RouteOperationID,
			WorkCenterID:     row.WorkCenterID,
			Priority:         row.Priority,
			TimeFactor:       pgutil.FromPgNumericToFloat64(row.TimeFactor),
			IsPrimary:        row.IsPrimary,
			CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
			UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		}
		if row.WorkCenterName.Valid {
			e.WorkCenterName = row.WorkCenterName.String
		}
		out = append(out, e)
	}
	return out, nil
}

func (r *RoutingRepositorySQLC) ListResourcesByRoute(ctx context.Context, routeID int64) ([]*entity.RouteOpResource, error) {
	rows, err := r.q.ListResourcesByRoute(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("listing resources for route %d: %w", routeID, err)
	}
	out := make([]*entity.RouteOpResource, 0, len(rows))
	for _, row := range rows {
		e := &entity.RouteOpResource{
			ID:               row.ID,
			RouteOperationID: row.RouteOperationID,
			WorkCenterID:     row.WorkCenterID,
			Priority:         row.Priority,
			TimeFactor:       pgutil.FromPgNumericToFloat64(row.TimeFactor),
			IsPrimary:        row.IsPrimary,
			CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
			UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
		}
		if row.WorkCenterName.Valid {
			e.WorkCenterName = row.WorkCenterName.String
		}
		out = append(out, e)
	}
	return out, nil
}

func resourceRowToEntity(row sqlc.RouteOperationResource) *entity.RouteOpResource {
	return &entity.RouteOpResource{
		ID:               row.ID,
		RouteOperationID: row.RouteOperationID,
		WorkCenterID:     row.WorkCenterID,
		Priority:         row.Priority,
		TimeFactor:       pgutil.FromPgNumericToFloat64(row.TimeFactor),
		IsPrimary:        row.IsPrimary,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

// ─── mappers ─────────────────────────────────────────────────────────────────

func operationRowToEntity(row sqlc.Operation) *entity.Operation {
	e := &entity.Operation{
		ID:                   row.ID,
		Code:                 row.Code,
		Name:                 row.Name,
		Origin:               entity.OperationOrigin(row.Origin),
		Situation:            entity.OperationSituation(row.Situation),
		StandardTime:         pgutil.FromPgNumericToFloat64(row.StandardTime),
		SetupTime:            pgutil.FromPgNumericToFloat64(row.SetupTime),
		RunTime:              pgutil.FromPgNumericToFloat64(row.RunTime),
		LaborTime:            pgutil.FromPgNumericToFloat64(row.LaborTime),
		RunBaseQty:           pgutil.FromPgNumericToFloat64(row.RunTimeBaseQty),
		QueueTime:            pgutil.FromPgNumericToFloat64(row.QueueTime),
		WaitTime:             pgutil.FromPgNumericToFloat64(row.WaitTime),
		MoveTime:             pgutil.FromPgNumericToFloat64(row.MoveTime),
		CrewSize:             pgutil.FromPgNumericToFloat64(row.CrewSize),
		TimeUnit:             row.TimeUnit,
		SupplierID:           row.SupplierID,
		ServiceItemCode:      row.ServiceItemCode,
		CostPerUnit:          numToPtr(row.CostPerUnit),
		LeadTimeDays:         row.LeadTimeDays,
		ThirdPartyRemittance: row.ThirdPartyRemittance,
		IsActive:             row.IsActive,
		CreatedAt:            pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:            pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:            pgutil.FromPgUUID(row.CreatedBy),
		DefaultWorkCenterID:  row.DefaultWorkCenterID,
	}
	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}
	return e
}

// numToPtr converts a nullable pg numeric to *float64 (nil when NULL).
func numToPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v := pgutil.FromPgNumericToFloat64(n)
	return &v
}

// textToPtr converts a nullable pg text to *string (nil when NULL).
func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

func routeRowToEntity(row sqlc.ManufacturingRoute) *entity.ManufacturingRoute {
	e := &entity.ManufacturingRoute{
		ID:          row.ID,
		Code:        row.Code,
		ItemCode:    row.ItemCode,
		Alternative: row.Alternative,
		Situation:   entity.RouteSituation(row.Situation),
		IsStandard:  row.IsStandard,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:   pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:   pgutil.FromPgUUID(row.CreatedBy),
		ValidFrom:   pgutil.FromPgDateToPtr(row.ValidFrom),
		ValidTo:     pgutil.FromPgDateToPtr(row.ValidTo),
	}
	if row.Mask.Valid {
		v := row.Mask.String
		e.Mask = &v
	}
	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}
	return e
}

func routeOpRowToEntity(row sqlc.RouteOperation) *entity.RouteOperation {
	e := &entity.RouteOperation{
		ID:           row.ID,
		RouteID:      row.RouteID,
		Sequence:     row.Sequence,
		OperationID:  row.OperationID,
		WorkCenterID: row.WorkCenterID,
		Situation:    entity.RouteOpSituation(row.Situation),
		IsActive:     row.IsActive,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
	if row.StandardTime.Valid {
		v := pgutil.FromPgNumericToFloat64(row.StandardTime)
		e.StandardTime = &v
	}
	if row.SetupTime.Valid {
		v := pgutil.FromPgNumericToFloat64(row.SetupTime)
		e.SetupTime = &v
	}
	e.RunTime = numToPtr(row.RunTime)
	e.LaborTime = numToPtr(row.LaborTime)
	e.RunBaseQty = numToPtr(row.RunTimeBaseQty)
	e.QueueTime = numToPtr(row.QueueTime)
	e.WaitTime = numToPtr(row.WaitTime)
	e.MoveTime = numToPtr(row.MoveTime)
	e.CrewSize = numToPtr(row.CrewSize)
	e.TimeUnit = textToPtr(row.TimeUnit)
	e.SupplierID = row.SupplierID
	e.ServiceItemCode = row.ServiceItemCode
	e.CostPerUnit = numToPtr(row.CostPerUnit)
	e.LeadTimeDays = row.LeadTimeDays
	e.ThirdPartyRemittance = textToPtr(row.ThirdPartyRemittance)
	if row.Notes.Valid {
		v := row.Notes.String
		e.Notes = &v
	}
	return e
}

func routeOpRowWithNamesToEntity(row sqlc.GetRouteOperationsRow) *entity.RouteOperation {
	e := routeOpRowToEntity(sqlc.RouteOperation{
		ID:                   row.ID,
		RouteID:              row.RouteID,
		Sequence:             row.Sequence,
		OperationID:          row.OperationID,
		WorkCenterID:         row.WorkCenterID,
		StandardTime:         row.StandardTime,
		SetupTime:            row.SetupTime,
		RunTime:              row.RunTime,
		LaborTime:            row.LaborTime,
		RunTimeBaseQty:       row.RunTimeBaseQty,
		QueueTime:            row.QueueTime,
		WaitTime:             row.WaitTime,
		MoveTime:             row.MoveTime,
		CrewSize:             row.CrewSize,
		TimeUnit:             row.TimeUnit,
		Situation:            row.Situation,
		Notes:                row.Notes,
		IsActive:             row.IsActive,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		SupplierID:           row.SupplierID,
		ServiceItemCode:      row.ServiceItemCode,
		CostPerUnit:          row.CostPerUnit,
		LeadTimeDays:         row.LeadTimeDays,
		ThirdPartyRemittance: row.ThirdPartyRemittance,
	})
	e.OperationName = row.OperationName
	e.OperationOrigin = entity.OperationOrigin(row.OperationOrigin)
	e.RequiresOperator = row.RequiresOperator
	e.EffectiveWorkCenterID = row.EffectiveWorkCenterID
	if row.WorkCenterName.Valid {
		e.WorkCenterName = row.WorkCenterName.String
	}

	// Resolve the quantity-aware time model (route-op overrides ∘ operation defaults),
	// normalised to hours.
	def := entity.TimeComponents{
		Setup:      pgutil.FromPgNumericToFloat64(row.OpSetupTime),
		Run:        pgutil.FromPgNumericToFloat64(row.OpRunTime),
		Labor:      pgutil.FromPgNumericToFloat64(row.OpLaborTime),
		RunBaseQty: pgutil.FromPgNumericToFloat64(row.OpRunTimeBaseQty),
		Queue:      pgutil.FromPgNumericToFloat64(row.OpQueueTime),
		Wait:       pgutil.FromPgNumericToFloat64(row.OpWaitTime),
		Move:       pgutil.FromPgNumericToFloat64(row.OpMoveTime),
		CrewSize:   pgutil.FromPgNumericToFloat64(row.OpCrewSize),
		Unit:       row.OpTimeUnit,
	}
	e.EffTime = entity.ResolveOperationTime(e.Overrides(), def)
	e.EffectiveStdTime = e.EffTime.Run
	e.EffectiveSetup = e.EffTime.Setup
	return e
}

func (r *RoutingRepositorySQLC) GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*entity.ExternalOp, error) {
	rows, err := r.q.GetExternalRouteOpsForItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching external ops for item %d: %w", itemCode, err)
	}
	out := make([]*entity.ExternalOp, 0, len(rows))
	for _, row := range rows {
		var supplierID *int64
		if row.SupplierID > 0 {
			value := row.SupplierID
			supplierID = &value
		}
		op := &entity.ExternalOp{
			RouteOpID:       row.ID,
			OperationID:     row.OperationID,
			OperationName:   row.OperationName,
			EffectiveHours:  pgutil.FromPgNumericToFloat64(row.EffectiveHours),
			Origin:          entity.OperationOrigin(row.Origin),
			WorkCenterID:    row.WorkCenterID,
			SupplierID:      supplierID,
			ServiceItemCode: row.ServiceItemCode,
			CostPerUnit:     pgutil.FromPgNumericToFloat64(row.CostPerUnit),
			LeadTimeDays:    row.LeadTimeDays,
			RemittanceType:  row.RemittanceType,
		}
		out = append(out, op)
	}
	return out, nil
}
