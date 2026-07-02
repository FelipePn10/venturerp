package entity

import "sort"

// CriticalPath computes the CPM critical-path lead time (in hours) across the
// operation network for a given production quantity, using PERT/CPM:
//
//	earlyStart[op]  = max over predecessors { earlyFinish[pred] - overlap% * predDuration }
//	earlyFinish[op] = earlyStart[op] + duration[op]
//
// where duration[op] = op.EffTime.LeadTimeHours(qty) (run scales with the lot;
// setup/queue/wait/move are fixed).
//
// When the predecessor's work center requires an operator, overlap is forced to
// 0 — the operator will not leave the machine until that operation is 100% done.
//
// When the route has no explicit network edges, operations are chained linearly
// in ascending sequence order (10 → 20 → 30), matching the intuitive default of
// a sequential routing.
//
// This is the single source of truth for lead-time computation; both the routing
// lead-time use case and MRP call it so their results never diverge.
func CriticalPath(ops []*RouteOperation, edges []*NetworkEdge, qty float64) LeadTimeResult {
	if len(ops) == 0 {
		return LeadTimeResult{}
	}
	if qty <= 0 {
		qty = 1
	}
	// Linear fallback: no network defined → chain consecutive operations by sequence.
	if len(edges) == 0 && len(ops) > 1 {
		edges = linearEdgesBySequence(ops)
	}

	opByID := make(map[int64]*RouteOperation, len(ops))
	for _, op := range ops {
		opByID[op.ID] = op
	}

	successors := make(map[int64][]int64)
	predecessors := make(map[int64][]int64)
	overlapByEdge := make(map[[2]int64]float64)
	for _, e := range edges {
		successors[e.PredecessorID] = append(successors[e.PredecessorID], e.SuccessorID)
		predecessors[e.SuccessorID] = append(predecessors[e.SuccessorID], e.PredecessorID)
		overlapByEdge[[2]int64{e.PredecessorID, e.SuccessorID}] = e.OverlapPct
	}

	// Topological order (Kahn).
	inDegree := make(map[int64]int, len(ops))
	for _, op := range ops {
		inDegree[op.ID] = len(predecessors[op.ID])
	}
	queue := make([]int64, 0, len(ops))
	for _, op := range ops {
		if inDegree[op.ID] == 0 {
			queue = append(queue, op.ID)
		}
	}

	earlyFinish := make(map[int64]float64, len(ops))
	prev := make(map[int64]int64)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		op := opByID[cur]
		opTime := op.EffTime.LeadTimeHours(qty)

		ef := opTime // no predecessors → earlyStart = 0
		bestPred := int64(0)
		for _, predID := range predecessors[cur] {
			overlap := overlapByEdge[[2]int64{predID, cur}] / 100.0
			if opByID[predID].RequiresOperator {
				overlap = 0
			}
			predDuration := opByID[predID].EffTime.LeadTimeHours(qty)
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

	// Sink = node with no successors and the largest early finish.
	var sinkID int64
	maxEF := 0.0
	for id, ef := range earlyFinish {
		if len(successors[id]) == 0 && ef >= maxEF {
			maxEF = ef
			sinkID = id
		}
	}

	// Reconstruct the critical path.
	path := make([]int64, 0)
	for cur := sinkID; cur != 0; {
		path = append([]int64{cur}, path...)
		p, ok := prev[cur]
		if !ok {
			break
		}
		cur = p
	}

	return LeadTimeResult{
		TotalHours:   maxEF,
		CriticalPath: path,
	}
}

// linearEdgesBySequence synthesises predecessor→successor edges chaining the
// operations in ascending sequence order.
func linearEdgesBySequence(ops []*RouteOperation) []*NetworkEdge {
	sorted := make([]*RouteOperation, len(ops))
	copy(sorted, ops)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Sequence < sorted[j].Sequence })
	edges := make([]*NetworkEdge, 0, len(sorted)-1)
	for i := 1; i < len(sorted); i++ {
		edges = append(edges, &NetworkEdge{PredecessorID: sorted[i-1].ID, SuccessorID: sorted[i].ID})
	}
	return edges
}
