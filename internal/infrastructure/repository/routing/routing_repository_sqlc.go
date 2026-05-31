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
		Code:                op.Code,
		Name:                op.Name,
		Description:         pgutil.ToPgTextFromPtr(op.Description),
		Origin:              sqltypes.OperationOriginEnum(op.Origin),
		Situation:           sqltypes.OperationSituationEnum(op.Situation),
		DefaultWorkCenterID: op.DefaultWorkCenterID,
		StandardTime:        pgutil.ToPgNumericFromFloat64(op.StandardTime),
		SetupTime:           pgutil.ToPgNumericFromFloat64(op.SetupTime),
		CreatedBy:           pgutil.ToPgUUID(op.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating operation: %w", err)
	}
	return operationRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateOperation(ctx context.Context, op *entity.Operation) (*entity.Operation, error) {
	row, err := r.q.UpdateOperation(ctx, sqlc.UpdateOperationParams{
		ID:                  op.ID,
		Name:                op.Name,
		Description:         pgutil.ToPgTextFromPtr(op.Description),
		Origin:              sqltypes.OperationOriginEnum(op.Origin),
		Situation:           sqltypes.OperationSituationEnum(op.Situation),
		DefaultWorkCenterID: op.DefaultWorkCenterID,
		StandardTime:        pgutil.ToPgNumericFromFloat64(op.StandardTime),
		SetupTime:           pgutil.ToPgNumericFromFloat64(op.SetupTime),
	})
	if err != nil {
		return nil, fmt.Errorf("updating operation: %w", err)
	}
	return operationRowToEntity(row), nil
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
		RouteID:      op.RouteID,
		Sequence:     op.Sequence,
		OperationID:  op.OperationID,
		WorkCenterID: op.WorkCenterID,
		StandardTime: pgutil.ToPgNumericFromFloat64Ptr(op.StandardTime),
		SetupTime:    pgutil.ToPgNumericFromFloat64Ptr(op.SetupTime),
		Situation:    sqltypes.RouteOpSituationEnum(op.Situation),
		Notes:        pgutil.ToPgTextFromPtr(op.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("adding route operation: %w", err)
	}
	return routeOpRowToEntity(row), nil
}

func (r *RoutingRepositorySQLC) UpdateRouteOperation(ctx context.Context, op *entity.RouteOperation) (*entity.RouteOperation, error) {
	row, err := r.q.UpdateRouteOperation(ctx, sqlc.UpdateRouteOperationParams{
		ID:           op.ID,
		WorkCenterID: op.WorkCenterID,
		StandardTime: pgutil.ToPgNumericFromFloat64Ptr(op.StandardTime),
		SetupTime:    pgutil.ToPgNumericFromFloat64Ptr(op.SetupTime),
		Situation:    sqltypes.RouteOpSituationEnum(op.Situation),
		Notes:        pgutil.ToPgTextFromPtr(op.Notes),
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

// ─── mappers ─────────────────────────────────────────────────────────────────

func operationRowToEntity(row sqlc.Operation) *entity.Operation {
	e := &entity.Operation{
		ID:                  row.ID,
		Code:                row.Code,
		Name:                row.Name,
		Origin:              entity.OperationOrigin(row.Origin),
		Situation:           entity.OperationSituation(row.Situation),
		StandardTime:        pgutil.FromPgNumericToFloat64(row.StandardTime),
		SetupTime:           pgutil.FromPgNumericToFloat64(row.SetupTime),
		IsActive:            row.IsActive,
		CreatedAt:           pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:           pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
		DefaultWorkCenterID: row.DefaultWorkCenterID,
	}
	if row.Description.Valid {
		v := row.Description.String
		e.Description = &v
	}
	return e
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
	if row.Notes.Valid {
		v := row.Notes.String
		e.Notes = &v
	}
	return e
}

func routeOpRowWithNamesToEntity(row sqlc.GetRouteOperationsRow) *entity.RouteOperation {
	e := routeOpRowToEntity(sqlc.RouteOperation{
		ID:           row.ID,
		RouteID:      row.RouteID,
		Sequence:     row.Sequence,
		OperationID:  row.OperationID,
		WorkCenterID: row.WorkCenterID,
		StandardTime: row.StandardTime,
		SetupTime:    row.SetupTime,
		Situation:    row.Situation,
		Notes:        row.Notes,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	})
	e.OperationName = row.OperationName
	e.OperationOrigin = entity.OperationOrigin(row.OperationOrigin)
	e.RequiresOperator = row.RequiresOperator
	e.EffectiveStdTime = effectiveTime(row.StandardTime, row.OpStandardTime)
	e.EffectiveSetup = effectiveTime(row.SetupTime, row.OpSetupTime)
	if row.WorkCenterName.Valid {
		e.WorkCenterName = row.WorkCenterName.String
	}
	return e
}

func (r *RoutingRepositorySQLC) GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*entity.ExternalOp, error) {
	rows, err := r.q.GetExternalRouteOpsForItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching external ops for item %d: %w", itemCode, err)
	}
	out := make([]*entity.ExternalOp, 0, len(rows))
	for _, row := range rows {
		op := &entity.ExternalOp{
			RouteOpID:      row.ID,
			OperationID:    row.OperationID,
			OperationName:  row.OperationName,
			EffectiveHours: pgutil.FromPgNumericToFloat64(row.EffectiveHours),
			Origin:         entity.OperationOrigin(row.Origin),
			WorkCenterID:   row.WorkCenterID,
		}
		out = append(out, op)
	}
	return out, nil
}

func effectiveTime(override pgtype.Numeric, fallback pgtype.Numeric) float64 {
	if override.Valid {
		return pgutil.FromPgNumericToFloat64(override)
	}
	return pgutil.FromPgNumericToFloat64(fallback)
}
