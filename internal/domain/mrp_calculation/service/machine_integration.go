package service

import (
	"context"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	machinesvc "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	"math"
	"sort"
	"time"
)

func selectMachineTime(list []*entity.MachineTimeInfo, mask string) *entity.MachineTimeInfo {
	var best *entity.MachineTimeInfo
	for _, mt := range list {
		if mt.Mask != mask && mt.Mask != "" {
			continue
		}
		if best == nil || (mt.Mask == mask && best.Mask != mask) || (mt.Mask == best.Mask && (mt.Priority < best.Priority || mt.Priority == best.Priority && mt.MachineID < best.MachineID)) {
			best = mt
		}
	}
	return best
}

func machineMinutes(mt *entity.MachineTimeInfo, qty float64) (float64, error) {
	if mt.ProductionTime <= 0 || mt.ProductionBaseQty <= 0 || mt.SetupTime < 0 {
		return 0, fmt.Errorf("produtividade inválida do item %d na máquina %d", mt.ItemCode, mt.MachineCode)
	}
	switch mt.ProductionTimeUnit {
	case "SEGUNDO", "MINUTO", "HORA", "DIA":
	default:
		return 0, fmt.Errorf("unidade de tempo inválida do item %d", mt.ItemCode)
	}
	imt := &machineentity.ItemMachineTime{ProductionTime: mt.ProductionTime, ProductionTimeUnit: types.CapacityPeriod(mt.ProductionTimeUnit), ProductionBaseQty: mt.ProductionBaseQty, SetupTime: mt.SetupTime, EfficiencyRate: mt.EfficiencyRate, TimeBasis: mt.TimeBasis}
	// Trocar o cilindro para a máquina. Sem contar essas paradas a ordem cabe no
	// turno na conta e estoura no chão — é o erro clássico de quem planeja corte
	// a laser só pelo tempo de corte.
	if mt.ConsumptionPerHour != nil && mt.ConsumableCapacity != nil {
		uso := &machineentity.ConsumableUsage{PerHour: *mt.ConsumptionPerHour, CapacityPerRefill: *mt.ConsumableCapacity}
		if mt.ConsumableSwapMinutes != nil {
			uso.ReplacementMinutes = *mt.ConsumableSwapMinutes
		}
		if mt.ConsumableUnit != nil {
			uso.Unit = *mt.ConsumableUnit
		}
		if mt.ConsumableName != nil {
			uso.Description = *mt.ConsumableName
		}
		imt.Consumable = uso
	}
	m := &machineentity.Machine{EfficiencyRate: mt.MachineEfficiencyRate}
	result := machinesvc.CalculateProductionTime(imt, m, qty, 1, mt.WorkingHoursPerDay*60)
	if math.IsNaN(result.TotalMinutes) || math.IsInf(result.TotalMinutes, 0) || result.TotalMinutes <= 0 {
		return 0, fmt.Errorf("duração inválida do item %d", mt.ItemCode)
	}
	return result.TotalMinutes, nil
}

type machinePlanningCalendar interface {
	MachinePlanningWindows(context.Context, int64, int64, time.Time, time.Time) ([]machinesvc.CapacityWindow, []machinesvc.CapacityWindow, error)
	SaveMachineSlots(context.Context, int64, int64, *int64, []machinesvc.CapacityWindow, bool) error
}

func (s *MRPServiceImpl) processMachineIntegration(ctx context.Context, planCode int64, allCodes []int64) error {
	if tx, ok := s.MRPRepo.(interface {
		WithMachinePlanningTransaction(context.Context, func(mrprepo.MRPCalculationRepository) error) error
	}); ok {
		return tx.WithMachinePlanningTransaction(ctx, func(repo mrprepo.MRPCalculationRepository) error {
			copy := *s
			copy.MRPRepo = repo
			return copy.scheduleMachineIntegration(ctx, planCode, allCodes)
		})
	}
	return s.scheduleMachineIntegration(ctx, planCode, allCodes)
}
func (s *MRPServiceImpl) scheduleMachineIntegration(ctx context.Context, planCode int64, allCodes []int64) error {
	times, err := s.MRPRepo.ListItemMachineTimes(ctx, allCodes)
	if err != nil {
		return fmt.Errorf("carregando produtividade por item: %w", err)
	}

	suggestions, err := s.MRPRepo.ListSuggestionsByPlan(ctx, planCode)
	if err != nil {
		return err
	}
	if cleaner, ok := s.MRPRepo.(interface {
		ClearUnreleasedMachineAllocations(context.Context, int64) error
	}); ok {
		if err := cleaner.ClearUnreleasedMachineAllocations(ctx, planCode); err != nil {
			return err
		}
	}
	sort.SliceStable(suggestions, func(i, j int) bool {
		if suggestions[i].NeedDate.Equal(suggestions[j].NeedDate) {
			return suggestions[i].Code < suggestions[j].Code
		}
		return suggestions[i].NeedDate.Before(suggestions[j].NeedDate)
	})
	suggestions, err = orderMachineSuggestions(suggestions)
	if err != nil {
		return err
	}
	ready := map[int64]time.Time{}
	reserved := map[int64][]machinesvc.CapacityWindow{}
	calendar, finite := s.MRPRepo.(machinePlanningCalendar)
	for _, sug := range suggestions {
		if sug.OrderType != "FABRICACAO" {
			recordMaterialReady(ready, sug, sug.NeedDate)
			continue
		}
		if reader, ok := s.MRPRepo.(interface {
			SuggestionHasOrder(context.Context, int64) (bool, error)
		}); ok {
			exists, err := reader.SuggestionHasOrder(ctx, sug.Code)
			if err != nil {
				return err
			}
			if exists {
				end := sug.NeedDate
				if sug.EstimatedEndAt != nil {
					end = *sug.EstimatedEndAt
				}
				recordMaterialReady(ready, sug, end)
				continue
			}
		}
		steps, err := s.machineSteps(ctx, sug, times[sug.ItemCode])
		if err != nil {
			return err
		}
		if len(steps) == 0 {
			recordMaterialReady(ready, sug, sug.NeedDate)
			continue
		}
		totalMinutes := 0.0
		for _, step := range steps {
			totalMinutes += step.minutes
		}
		if err = s.MRPRepo.UpdatePlannedOrderMachine(ctx, sug.Code, steps[0].profile.MachineID, totalMinutes); err != nil {
			return err
		}
		day := sug.NeedDate
		if sug.StartDate != nil {
			day = *sug.StartDate
		}
		if sug.RequestedStartDate != nil {
			day = *sug.RequestedStartDate
		}
		if ready[sug.ItemCode].After(day) {
			day = ready[sug.ItemCode]
		}
		firstStart := day
		for i, step := range steps {
			mt := step.profile
			day = day.Add(step.before)
			if finite {
				windows, busy, err := calendar.MachinePlanningWindows(ctx, mt.MachineID, planCode, day, day.AddDate(1, 0, 0))
				if err != nil {
					return err
				}
				busy = append(busy, reserved[mt.MachineID]...)
				var slots []machinesvc.CapacityWindow
				if step.cycles > 0 {
					slots, err = machinesvc.AllocateMachineCycles(day, step.cycles, step.cycleMinutes, step.setupMinutes, windows, busy)
				} else {
					slots, err = machinesvc.AllocateMachineTime(day, step.minutes, windows, busy)
				}
				if err != nil {
					return fmt.Errorf("item %d, máquina %d: %w", sug.ItemCode, mt.MachineCode, err)
				}
				if err = calendar.SaveMachineSlots(ctx, sug.Code, mt.MachineID, step.operationID, slots, i == 0); err != nil {
					return err
				}
				reserved[mt.MachineID] = append(reserved[mt.MachineID], slots...)
				if i == 0 {
					firstStart = slots[0].Start
				}
				day = slots[len(slots)-1].End.Add(step.after)
			}
		}
		if err = s.MRPRepo.CreateMachineSchedule(ctx, &entity.MachineScheduleInfo{PlanCode: planCode, PlannedOrderCode: sug.Code, MachineID: steps[0].profile.MachineID, ScheduleDate: firstStart, ProductionTime: totalMinutes}); err != nil {
			return err
		}

		if writer, ok := s.MRPRepo.(interface {
			SaveMachineCompletion(context.Context, int64, time.Time) error
		}); ok {
			if err := writer.SaveMachineCompletion(ctx, sug.Code, day); err != nil {
				return err
			}
		}
		recordMaterialReady(ready, sug, day)
	}
	return nil
}

// Components are planned before their consuming parent. When several orders
// share a component, waiting for all its dependent suggestions is conservative.
func orderMachineSuggestions(in []*entity.PlannedOrderSuggestion) ([]*entity.PlannedOrderSuggestion, error) {
	out := make([]*entity.PlannedOrderSuggestion, 0, len(in))
	done := map[int64]bool{}
	for len(out) < len(in) {
		progress := false
		for _, s := range in {
			if done[s.Code] {
				continue
			}
			ready := true
			for _, child := range in {
				if child.ParentItemCode != nil && *child.ParentItemCode == s.ItemCode && !done[child.Code] {
					ready = false
					break
				}
			}
			if ready {
				out = append(out, s)
				done[s.Code] = true
				progress = true
			}
		}
		if !progress {
			return nil, fmt.Errorf("dependências de materiais em ciclo nas sugestões do MRP")
		}
	}
	return out, nil
}
func recordMaterialReady(ready map[int64]time.Time, s *entity.PlannedOrderSuggestion, end time.Time) {
	if s.ParentItemCode != nil && end.After(ready[*s.ParentItemCode]) {
		ready[*s.ParentItemCode] = end
	}
}
