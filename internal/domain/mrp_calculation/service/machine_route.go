package service

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"math"
	"sort"
	"time"
)

type machineStep struct {
	profile                            *entity.MachineTimeInfo
	minutes                            float64
	cycles, cycleMinutes, setupMinutes float64
	operationID                        *int64
	before, after                      time.Duration
}
type machineRouteReader interface {
	MachineRoute(context.Context, int64, string) ([]*routing.RouteOperation, []*routing.NetworkEdge, error)
}

// Route-specific times take precedence. Otherwise the measured item/machine
// productivity supplies the operation's processing time. Every internal step
// needs an eligible machine; missing data must not produce a partial schedule.
func (s *MRPServiceImpl) machineSteps(ctx context.Context, sug *entity.PlannedOrderSuggestion, profiles []*entity.MachineTimeInfo) ([]machineStep, error) {
	var ops []*routing.RouteOperation
	var edges []*routing.NetworkEdge
	if reader, ok := s.MRPRepo.(machineRouteReader); ok {
		var err error
		ops, edges, err = reader.MachineRoute(ctx, sug.ItemCode, sug.Mask)
		if err != nil {
			return nil, err
		}
	}
	if len(ops) == 0 {
		mt := selectMachineTime(profiles, sug.Mask)
		if mt == nil {
			return nil, nil
		}
		minutes, err := machineMinutes(mt, sug.Quantity)
		step := measuredMachineStep(mt, sug.Quantity, minutes)
		return []machineStep{step}, err
	}
	ordered, err := orderMachineRoute(ops, edges)
	if err != nil {
		return nil, err
	}
	out := []machineStep{}
	delay := time.Duration(0)
	for _, op := range ordered {
		if op.Situation == routing.RouteOpGhost || op.Situation == routing.RouteOpInactive {
			continue
		}
		if op.OperationOrigin == routing.OriginExternal || op.OperationOrigin == routing.OriginThirdPart {
			if op.LeadTimeDays == nil || *op.LeadTimeDays <= 0 {
				return nil, fmt.Errorf("informe o prazo da operação externa %d antes de planejar a capacidade", op.ID)
			}
			delay += time.Duration(*op.LeadTimeDays) * 24 * time.Hour
			continue
		}
		if op.EffectiveWorkCenterID == nil {
			return nil, fmt.Errorf("operação %d do item %d sem centro de trabalho", op.ID, sug.ItemCode)
		}
		candidates := []*entity.MachineTimeInfo{}
		for _, profile := range profiles {
			if profile.WorkCenterID == *op.EffectiveWorkCenterID {
				candidates = append(candidates, profile)
			}
		}
		mt := selectMachineTime(candidates, sug.Mask)
		if mt == nil {
			return nil, fmt.Errorf("cadastre a produtividade do item %d em uma máquina do centro %d antes de planejar", sug.ItemCode, *op.EffectiveWorkCenterID)
		}
		minutes, err := machineMinutes(mt, sug.Quantity)
		if err != nil {
			return nil, err
		}
		if op.RunTime != nil || op.StandardTime != nil {
			minutes = op.EffTime.MachineHours(sug.Quantity) * 60
		}
		if minutes <= 0 {
			return nil, fmt.Errorf("tempo inválido na operação %d", op.ID)
		}
		id := op.ID
		step := measuredMachineStep(mt, sug.Quantity, minutes)
		if op.RunTime != nil || op.StandardTime != nil {
			step.cycles = op.EffTime.Batches(sug.Quantity)
			step.cycleMinutes = op.EffTime.Run * 60
			step.setupMinutes = op.EffTime.Setup * 60
			if step.cycleMinutes <= 0 {
				step.cycles = 0
			}
		}
		step.operationID = &id
		step.before = delay + time.Duration(op.EffTime.Queue*float64(time.Hour))
		step.after = time.Duration((op.EffTime.Wait + op.EffTime.Move) * float64(time.Hour))
		out = append(out, step)
		delay = 0
	}
	if len(out) > 0 {
		out[len(out)-1].after += delay
	}
	return out, nil
}

// Stable topological order preserves all dependencies. Parallel branches are
// conservatively sequenced; the finite scheduler never starts a successor early.
func orderMachineRoute(ops []*routing.RouteOperation, edges []*routing.NetworkEdge) ([]*routing.RouteOperation, error) {
	sorted := append([]*routing.RouteOperation(nil), ops...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Sequence < sorted[j].Sequence })
	if len(edges) == 0 {
		return sorted, nil
	}
	done := map[int64]bool{}
	out := make([]*routing.RouteOperation, 0, len(ops))
	for len(out) < len(ops) {
		progress := false
		for _, op := range sorted {
			if done[op.ID] {
				continue
			}
			ready := true
			for _, edge := range edges {
				if edge.SuccessorID == op.ID && !done[edge.PredecessorID] {
					ready = false
					break
				}
			}
			if ready {
				done[op.ID] = true
				out = append(out, op)
				progress = true
			}
		}
		if !progress {
			return nil, fmt.Errorf("roteiro com ciclo ou precedência inválida")
		}
	}
	return out, nil
}

func measuredMachineStep(mt *entity.MachineTimeInfo, qty, minutes float64) machineStep {
	step := machineStep{profile: mt, minutes: minutes}
	if mt.TimeBasis != "PROPORTIONAL" && mt.ProductionBaseQty > 0 {
		step.cycles = math.Ceil(qty / float64(mt.ProductionBaseQty))
		step.setupMinutes = mt.SetupTime
		if step.cycles > 0 {
			step.cycleMinutes = (minutes - mt.SetupTime) / step.cycles
		}
	}
	return step
}
