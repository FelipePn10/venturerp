package aps_uc

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/aps/repository"
)

// RescheduleSequence applies a planner's manual board move ("drag-drop"): it moves
// one scheduled operation to a new start (and optionally a new work center),
// preserving its wall-clock duration. When cascade is on (the default) it pushes the
// downstream operations of the same order so the finish-start chain stays valid, then
// checks whether any touched work center is now booked beyond its daily capacity and
// reports those days as non-blocking warnings — the move always goes through, matching
// how interactive APS boards let the planner override and then flag the conflict.
func (uc *APSUseCase) RescheduleSequence(ctx context.Context, dto request.RescheduleSequenceDTO) (*response.RescheduleResultResponse, error) {
	if dto.SequenceID <= 0 {
		return nil, fmt.Errorf("sequence_id is required")
	}
	if dto.NewStart.IsZero() {
		return nil, fmt.Errorf("new_start is required")
	}

	target, err := uc.repo.GetSequence(ctx, dto.SequenceID)
	if err != nil {
		return nil, fmt.Errorf("loading sequence %d: %w", dto.SequenceID, err)
	}

	duration := target.ScheduledEnd.Sub(target.ScheduledStart)
	if duration < 0 {
		duration = 0
	}
	newStart := dto.NewStart.In(time.Local)
	target.ScheduledStart = newStart
	target.ScheduledEnd = newStart.Add(duration)
	if dto.NewWorkCenterID != nil && *dto.NewWorkCenterID > 0 {
		target.WorkCenterID = *dto.NewWorkCenterID
	}
	if dto.NewMachineID != nil {
		selection, ok := uc.repo.(repository.SelectionRepository)
		if !ok {
			return nil, fmt.Errorf("machine rescheduling is not supported")
		}
		candidates, loadErr := selection.ListCandidateMachines(ctx, target.WorkCenterID, []int64{*dto.NewMachineID})
		if loadErr != nil {
			return nil, loadErr
		}
		if len(candidates) != 1 {
			return nil, fmt.Errorf("machine does not belong to the selected work center or tenant")
		}
		target.MachineID = dto.NewMachineID
	} else if dto.NewWorkCenterID != nil {
		target.MachineID = nil
	}

	moved, err := uc.repo.UpdateSequence(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("moving sequence %d: %w", dto.SequenceID, err)
	}

	cascade := dto.Cascade == nil || *dto.Cascade
	var shifted []*entity.ProductionSequence
	if cascade {
		shifted, err = uc.cascadeShift(ctx, moved)
		if err != nil {
			return nil, err
		}
	}

	out := &response.RescheduleResultResponse{
		Moved:          rescheduledBar(moved),
		CascadeApplied: cascade,
		Warnings:       uc.capacityWarnings(ctx, moved, shifted),
	}
	for _, s := range shifted {
		out.Shifted = append(out.Shifted, rescheduledBar(s))
	}
	return out, nil
}

// cascadeShift pushes the downstream operations of the moved sequence's order so that
// each successor starts no earlier than its predecessor finishes (minus the edge's
// overlap window). It uses the explicit route_operation_network edges when present,
// otherwise a linear chain inferred from the operations' sequence positions. Only
// sequences that actually had to move are persisted and returned.
func (uc *APSUseCase) cascadeShift(ctx context.Context, moved *entity.ProductionSequence) ([]*entity.ProductionSequence, error) {
	seqs, err := uc.repo.ListByOrder(ctx, moved.ProductionOrderID)
	if err != nil {
		return nil, fmt.Errorf("loading order %d sequences: %w", moved.ProductionOrderID, err)
	}
	byID := make(map[int64]*entity.ProductionSequence, len(seqs))
	for _, s := range seqs {
		if s.ID == moved.ID {
			byID[s.ID] = moved // use the freshly moved values
			continue
		}
		byID[s.ID] = s
	}

	edges := uc.orderEdges(ctx, moved.ProductionOrderID, seqs)
	succ := map[int64][]dependencyEdge{}
	for _, e := range edges {
		succ[e.from] = append(succ[e.from], e)
	}

	changed := map[int64]bool{}
	// Relaxation worklist; the iteration cap defends against a malformed cyclic graph.
	queue := []int64{moved.ID}
	maxIter := (len(seqs) + 1) * (len(seqs) + 1)
	for len(queue) > 0 && maxIter > 0 {
		maxIter--
		curID := queue[0]
		queue = queue[1:]
		cur := byID[curID]
		if cur == nil {
			continue
		}
		predDur := cur.ScheduledEnd.Sub(cur.ScheduledStart)
		for _, e := range succ[curID] {
			s := byID[e.to]
			if s == nil {
				continue
			}
			overlap := time.Duration(float64(predDur) * clamp01(e.overlapPct/100))
			required := cur.ScheduledEnd.Add(-overlap)
			if s.ScheduledStart.Before(required) {
				delta := required.Sub(s.ScheduledStart)
				s.ScheduledStart = required
				s.ScheduledEnd = s.ScheduledEnd.Add(delta)
				changed[s.ID] = true
				queue = append(queue, s.ID)
			}
		}
	}

	shifted := make([]*entity.ProductionSequence, 0, len(changed))
	for id := range changed {
		saved, err := uc.repo.UpdateSequence(ctx, byID[id])
		if err != nil {
			return nil, fmt.Errorf("shifting sequence %d: %w", id, err)
		}
		shifted = append(shifted, saved)
	}
	sort.SliceStable(shifted, func(i, j int) bool { return shifted[i].ScheduledStart.Before(shifted[j].ScheduledStart) })
	return shifted, nil
}

type dependencyEdge struct {
	from, to   int64
	overlapPct float64
}

// orderEdges returns the finish-start edges for one order: the explicit
// route_operation_network edges, or a synthesised linear chain by sequence position
// when the order has none.
func (uc *APSUseCase) orderEdges(ctx context.Context, orderID int64, seqs []*entity.ProductionSequence) []dependencyEdge {
	if deps, err := uc.repo.ListOrderDependencies(ctx, orderID); err == nil && len(deps) > 0 {
		out := make([]dependencyEdge, 0, len(deps))
		for _, d := range deps {
			out = append(out, dependencyEdge{from: d.FromSequenceID, to: d.ToSequenceID, overlapPct: d.OverlapPct})
		}
		return out
	}
	chain := append([]*entity.ProductionSequence(nil), seqs...)
	sort.SliceStable(chain, func(i, j int) bool {
		if chain[i].SequencePosition != chain[j].SequencePosition {
			return chain[i].SequencePosition < chain[j].SequencePosition
		}
		return chain[i].ScheduledStart.Before(chain[j].ScheduledStart)
	})
	out := make([]dependencyEdge, 0, len(chain))
	for i := 1; i < len(chain); i++ {
		out = append(out, dependencyEdge{from: chain[i-1].ID, to: chain[i].ID})
	}
	return out
}

// capacityWarnings flags every day on which a touched work center is now scheduled
// beyond its available hours. It aggregates the per-day overlap hours of all
// sequences on the affected work centers within the days the move actually touched.
func (uc *APSUseCase) capacityWarnings(ctx context.Context, moved *entity.ProductionSequence, shifted []*entity.ProductionSequence) []response.CapacityWarningResponse {
	touched := append([]*entity.ProductionSequence{moved}, shifted...)

	wcSet := map[int64]bool{}
	minStart, maxEnd := moved.ScheduledStart, moved.ScheduledEnd
	for _, s := range touched {
		wcSet[s.WorkCenterID] = true
		if s.ScheduledStart.Before(minStart) {
			minStart = s.ScheduledStart
		}
		if s.ScheduledEnd.After(maxEnd) {
			maxEnd = s.ScheduledEnd
		}
	}
	// The set of calendar days the move actually touched; warnings are limited to
	// these so an otherwise-busy work center doesn't produce noise.
	affectedDays := map[string]bool{}
	for _, s := range touched {
		for d := truncDay(s.ScheduledStart); d.Before(s.ScheduledEnd); d = d.AddDate(0, 0, 1) {
			affectedDays[d.Format("2006-01-02")] = true
		}
	}

	// Padded window so ListByWorkCenter (which matches fully-contained sequences)
	// still picks up the bars around the touched days.
	from := truncDay(minStart).AddDate(0, 0, -1)
	to := truncDay(maxEnd).AddDate(0, 0, 2)

	var warnings []response.CapacityWarningResponse
	for wc := range wcSet {
		capHours, _ := uc.repo.GetWorkCenterCapacity(ctx, wc)
		if capHours <= 0 {
			capHours = 8
		}
		seqs, err := uc.repo.ListByWorkCenter(ctx, wc, from, to)
		if err != nil {
			continue
		}
		perDay := map[string]float64{}
		for _, s := range seqs {
			for d := truncDay(s.ScheduledStart); d.Before(s.ScheduledEnd); d = d.AddDate(0, 0, 1) {
				key := d.Format("2006-01-02")
				if !affectedDays[key] {
					continue
				}
				perDay[key] += dayOverlapHours(s.ScheduledStart, s.ScheduledEnd, d)
			}
		}
		for key, hours := range perDay {
			if hours > capHours+1e-6 {
				day, _ := time.ParseInLocation("2006-01-02", key, time.Local)
				warnings = append(warnings, response.CapacityWarningResponse{
					WorkCenterID:   wc,
					Date:           day,
					ScheduledHours: round2(hours),
					AvailableHours: round2(capHours),
					OverByHours:    round2(hours - capHours),
				})
			}
		}
	}
	sort.SliceStable(warnings, func(i, j int) bool {
		if !warnings[i].Date.Equal(warnings[j].Date) {
			return warnings[i].Date.Before(warnings[j].Date)
		}
		return warnings[i].WorkCenterID < warnings[j].WorkCenterID
	})
	return warnings
}

// dayOverlapHours is the number of hours the interval [start, end) spends inside the
// single calendar day beginning at dayStart.
func dayOverlapHours(start, end, dayStart time.Time) float64 {
	dayEnd := dayStart.AddDate(0, 0, 1)
	lo := start
	if dayStart.After(lo) {
		lo = dayStart
	}
	hi := end
	if dayEnd.Before(hi) {
		hi = dayEnd
	}
	if hi.Before(lo) {
		return 0
	}
	return hi.Sub(lo).Hours()
}

func rescheduledBar(s *entity.ProductionSequence) response.RescheduledBarResponse {
	return response.RescheduledBarResponse{
		SequenceID:        s.ID,
		ProductionOrderID: s.ProductionOrderID,
		WorkCenterID:      s.WorkCenterID,
		MachineID:         s.MachineID,
		ScheduledStart:    s.ScheduledStart,
		ScheduledEnd:      s.ScheduledEnd,
		DurationHours:     round2(s.ScheduledEnd.Sub(s.ScheduledStart).Hours()),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
