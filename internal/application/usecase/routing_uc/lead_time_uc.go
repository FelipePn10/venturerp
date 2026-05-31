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
func (uc *LeadTimeUseCase) Execute(ctx context.Context, routeID int64) (*response.RouteLeadTimeResponse, error) {
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

	result := criticalPath(ops, edges)
	result.RouteID = routeID
	return &response.RouteLeadTimeResponse{
		RouteID:      result.RouteID,
		TotalHours:   result.TotalHours,
		CriticalPath: result.CriticalPath,
	}, nil
}

// GetRouteLeadTimeHours is a convenience used directly by MRP.
// Returns 0 if the item has no route.
func (uc *LeadTimeUseCase) GetRouteLeadTimeHours(ctx context.Context, itemCode int64, mask string) (float64, error) {
	route, err := uc.repo.GetRouteForItem(ctx, itemCode, mask)
	if err != nil {
		return 0, nil // no route → no lead-time contribution
	}
	result, err := uc.Execute(ctx, route.ID)
	if err != nil {
		return 0, err
	}
	return result.TotalHours, nil
}

// ─── CPM algorithm ────────────────────────────────────────────────────────────

func criticalPath(ops []*entity.RouteOperation, edges []*entity.NetworkEdge) entity.LeadTimeResult {
	// index ops by id
	opByID := make(map[int64]*entity.RouteOperation, len(ops))
	for _, op := range ops {
		opByID[op.ID] = op
	}

	// build successor and predecessor maps
	successors := make(map[int64][]int64)
	predecessors := make(map[int64][]int64)
	overlapByEdge := make(map[[2]int64]float64)
	for _, e := range edges {
		successors[e.PredecessorID] = append(successors[e.PredecessorID], e.SuccessorID)
		predecessors[e.SuccessorID] = append(predecessors[e.SuccessorID], e.PredecessorID)
		overlapByEdge[[2]int64{e.PredecessorID, e.SuccessorID}] = e.OverlapPct
	}

	// topological order (Kahn)
	inDegree := make(map[int64]int, len(ops))
	for _, op := range ops {
		if _, ok := inDegree[op.ID]; !ok {
			inDegree[op.ID] = 0
		}
		for range predecessors[op.ID] {
			inDegree[op.ID]++
		}
	}
	queue := make([]int64, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	earlyFinish := make(map[int64]float64, len(ops))
	prev := make(map[int64]int64) // for path reconstruction
	topo := make([]int64, 0, len(ops))

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		topo = append(topo, cur)

		op := opByID[cur]
		opTime := op.EffectiveStdTime + op.EffectiveSetup

		// earliest finish for cur:
		//   earlyStart[cur] = earlyFinish[pred] - overlap * predDuration
		//   earlyFinish[cur] = earlyStart[cur] + opTime
		// When the predecessor machine requires an operator, overlap is forced
		// to 0 — the operator won't leave until that operation is 100% done.
		ef := opTime // no preds → earlyStart = 0
		bestPred := int64(0)
		for _, predID := range predecessors[cur] {
			overlap := overlapByEdge[[2]int64{predID, cur}] / 100.0
			if opByID[predID].RequiresOperator {
				overlap = 0
			}
			predDuration := opByID[predID].EffectiveStdTime + opByID[predID].EffectiveSetup
			earlyStart := earlyFinish[predID] - overlap*predDuration
			if earlyStart+opTime > ef {
				ef = earlyStart + opTime
				bestPred = predID
			}
		}
		earlyFinish[cur] = ef
		if bestPred != 0 {
			prev[cur] = bestPred
		}

		for _, sucID := range successors[cur] {
			inDegree[sucID]--
			if inDegree[sucID] == 0 {
				queue = append(queue, sucID)
			}
		}
	}

	// find sink (no successors) with max earlyFinish
	var sinkID int64
	maxEF := 0.0
	for id, ef := range earlyFinish {
		if len(successors[id]) == 0 && ef > maxEF {
			maxEF = ef
			sinkID = id
		}
	}

	// reconstruct critical path
	path := make([]int64, 0)
	for cur := sinkID; cur != 0; {
		path = append([]int64{cur}, path...)
		p, ok := prev[cur]
		if !ok {
			break
		}
		cur = p
	}

	return entity.LeadTimeResult{
		TotalHours:   maxEF,
		CriticalPath: path,
	}
}
