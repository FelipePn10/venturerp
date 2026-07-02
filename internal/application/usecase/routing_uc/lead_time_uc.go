package routing_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
)

type LeadTimeUseCase struct {
	repo repository.RoutingRepository
}

func NewLeadTimeUseCase(repo repository.RoutingRepository) *LeadTimeUseCase {
	return &LeadTimeUseCase{repo: repo}
}

// Execute computes the critical-path lead time for the standard route of an item.
// It uses PERT/CPM on the operation network:
//
//	earlyFinish[op] = max over predecessors { earlyFinish[pred] * (1 - overlap%) } + effective_time[op]
//
// When the predecessor operation's work center has requires_operator = true, overlap is
// forced to 0: the operator will not leave the machine until it finishes, so the
// successor cannot start early regardless of the configured overlap_pct.
func (uc *LeadTimeUseCase) Execute(ctx context.Context, routeID int64, qty float64) (*response.RouteLeadTimeResponse, error) {
	if qty <= 0 {
		qty = 1
	}
	ops, err := uc.repo.GetRouteOperations(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching operations: %w", err)
	}
	if len(ops) == 0 {
		return &response.RouteLeadTimeResponse{RouteID: routeID}, nil
	}

	edges, err := uc.repo.GetNetworkEdges(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("fetching network: %w", err)
	}

	result := entity.CriticalPath(ops, edges, qty)
	result.RouteID = routeID
	return &response.RouteLeadTimeResponse{
		RouteID:      result.RouteID,
		TotalHours:   result.TotalHours,
		CriticalPath: result.CriticalPath,
	}, nil
}

// GetRouteLeadTimeHours is a convenience used directly by MRP.
// Returns 0 if the item has no route. qty scales the run (per-piece) portion.
func (uc *LeadTimeUseCase) GetRouteLeadTimeHours(ctx context.Context, itemCode int64, mask string, qty float64) (float64, error) {
	route, err := uc.repo.GetRouteForItem(ctx, itemCode, mask)
	if err != nil {
		return 0, nil // no route → no lead-time contribution
	}
	result, err := uc.Execute(ctx, route.ID, qty)
	if err != nil {
		return 0, err
	}
	return result.TotalHours, nil
}

// The CPM algorithm lives in the routing domain (entity.CriticalPath) so the
// lead-time use case and MRP share a single, consistent implementation.
