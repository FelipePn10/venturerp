package production_order_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

// OrderOperationsUseCase manages production order operations (exploding route + advancing status).
type OrderOperationsUseCase struct {
	Q *sqlc.Queries
}

// ExplodeRoute creates production_order_operations from a manufacturing route.
// Called after creating a production order when route_id is provided.
func (uc *OrderOperationsUseCase) ExplodeRoute(ctx context.Context, orderID, routeID int64) ([]*response.ProductionOrderOperationResponse, error) {
	ops, err := uc.Q.GetRouteOpsForExplode(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching route operations: %w", err)
	}

	out := make([]*response.ProductionOrderOperationResponse, 0, len(ops))
	for _, op := range ops {
		poo, err := uc.Q.CreateProductionOrderOperation(ctx, sqlc.CreateProductionOrderOperationParams{
			ProductionOrderID: orderID,
			RouteOperationID:  pgutil.ToPgInt8Ptr(&op.ID),
			Sequence:          op.Sequence,
			OperationName:     op.OperationName,
			WorkCenterID:      op.WorkCenterID,
			PlannedHours:      pgutil.ToPgNumericFromFloat64(op.PlannedHours),
			SetupHours:        pgutil.ToPgNumericFromFloat64(op.SetupHours),
		})
		if err != nil {
			return nil, fmt.Errorf("creating order operation seq %d: %w", op.Sequence, err)
		}
		out = append(out, pooToResponse(poo))
	}
	return out, nil
}

// ListOperations lists operations for a production order.
func (uc *OrderOperationsUseCase) ListOperations(ctx context.Context, orderID int64) ([]*response.ProductionOrderOperationResponse, error) {
	poos, err := uc.Q.ListProductionOrderOperations(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("listing operations: %w", err)
	}
	out := make([]*response.ProductionOrderOperationResponse, 0, len(poos))
	for _, poo := range poos {
		out = append(out, pooToResponse(poo))
	}
	return out, nil
}

// AdvanceOperation changes an operation status (PENDING → IN_PROGRESS → DONE).
func (uc *OrderOperationsUseCase) AdvanceOperation(ctx context.Context, dto request.AdvanceOperationDTO) (*response.ProductionOrderOperationResponse, error) {
	if dto.OperationID == 0 {
		return nil, fmt.Errorf("operation_id is required")
	}
	switch dto.Status {
	case "PENDING", "IN_PROGRESS", "DONE", "SKIPPED":
	default:
		return nil, fmt.Errorf("invalid status %q: must be PENDING, IN_PROGRESS, DONE or SKIPPED", dto.Status)
	}
	poo, err := uc.Q.AdvanceProductionOrderOperation(ctx, dto.OperationID, dto.Status)
	if err != nil {
		return nil, fmt.Errorf("advancing operation: %w", err)
	}
	if dto.ActualHours > 0 {
		_ = uc.Q.AddActualHours(ctx, dto.OperationID, pgutil.ToPgNumericFromFloat64(dto.ActualHours))
	}
	return pooToResponse(poo), nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func pooToResponse(p sqlc.DBProductionOrderOperation) *response.ProductionOrderOperationResponse {
	r := &response.ProductionOrderOperationResponse{
		ID:                p.ID,
		ProductionOrderID: p.ProductionOrderID,
		Sequence:          int(p.Sequence),
		OperationName:     p.OperationName,
		PlannedHours:      pgutil.FromPgNumericToFloat64(p.PlannedHours),
		SetupHours:        pgutil.FromPgNumericToFloat64(p.SetupHours),
		ActualHours:       pgutil.FromPgNumericToFloat64(p.ActualHours),
		Status:            p.Status,
	}
	if p.RouteOperationID.Valid {
		v := p.RouteOperationID.Int64
		r.RouteOperationID = &v
	}
	if p.WorkCenterID.Valid {
		v := p.WorkCenterID.Int64
		r.WorkCenterID = &v
	}
	if p.StartedAt.Valid {
		t := pgutil.FromPgTimestamptz(p.StartedAt)
		r.StartedAt = &t
	}
	if p.CompletedAt.Valid {
		t := pgutil.FromPgTimestamptz(p.CompletedAt)
		r.CompletedAt = &t
	}
	if p.Notes.Valid {
		r.Notes = &p.Notes.String
	}
	return r
}
