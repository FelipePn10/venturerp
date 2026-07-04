package crp_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/crp/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/crp/repository"
	maintenancerepo "github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
)

// routeOpReader is the slice of the routing repository CRP needs to load capacity
// using the rich, quantity-aware time model (machine hours per work center).
type routeOpReader interface {
	GetRouteOperations(ctx context.Context, routeID int64) ([]*routingentity.RouteOperation, error)
}

type CRPUseCase struct {
	repo      repository.CRPRepository
	maintRepo maintenancerepo.MaintenanceRepository
	routing   routeOpReader // optional; when nil, uses the legacy flat EffHours × qty
}

func New(repo repository.CRPRepository) *CRPUseCase {
	return &CRPUseCase{repo: repo}
}

// WithMaintenance injects the maintenance repository so CRP deducts
// scheduled maintenance hours from available capacity per work-center/date.
func (uc *CRPUseCase) WithMaintenance(r maintenancerepo.MaintenanceRepository) *CRPUseCase {
	uc.maintRepo = r
	return uc
}

// WithRouting enables quantity-aware machine-hour loading (setup + run×batches) per
// effective work center, replacing the flat EffHours × quantity estimate.
func (uc *CRPUseCase) WithRouting(r routeOpReader) *CRPUseCase {
	uc.routing = r
	return uc
}

// CalculateCRP computes and stores capacity requirements for a given MRP plan.
//
// Algorithm:
//  1. Fetch planned orders for the plan.
//  2. For each order with a route: expand route operations and accumulate
//     required_hours per (work_center_id, date).
//  3. Query available hours per work center.
//  4. Upsert all entries into capacity_requirements.
func (uc *CRPUseCase) CalculateCRP(ctx context.Context, dto request.CalculateCRPDTO) (*response.CRPSummaryResponse, error) {
	if err := uc.repo.DeleteByPlan(ctx, dto.PlanCode); err != nil {
		return nil, fmt.Errorf("clearing CRP for plan %d: %w", dto.PlanCode, err)
	}

	orders, err := uc.repo.GetPlannedOrdersByPlan(ctx, dto.PlanCode)
	if err != nil {
		return nil, fmt.Errorf("fetching planned orders: %w", err)
	}

	type wcDateKey struct {
		wcID int64
		date string
	}
	reqMap := make(map[wcDateKey]float64)
	dateMap := make(map[wcDateKey]time.Time)

	for _, order := range orders {
		if order.RouteID == nil {
			continue
		}
		day := truncateToDay(order.PlannedDate)

		// Preferred path: rich, quantity-aware machine hours (setup + run×batches)
		// charged to each operation's EFFECTIVE work center (override or op default).
		if uc.routing != nil {
			rops, err := uc.routing.GetRouteOperations(ctx, *order.RouteID)
			if err == nil {
				for _, op := range rops {
					if op.Situation == routingentity.RouteOpGhost || op.EffectiveWorkCenterID == nil {
						continue
					}
					k := wcDateKey{wcID: *op.EffectiveWorkCenterID, date: day.Format("2006-01-02")}
					reqMap[k] += op.EffTime.MachineHours(order.Quantity)
					dateMap[k] = day
				}
				continue
			}
		}

		// Legacy fallback: flat EffHours × quantity (setup not separated).
		ops, err := uc.repo.GetRouteOperationsByRoute(ctx, *order.RouteID)
		if err != nil {
			continue
		}
		for _, op := range ops {
			if op.WorkCenterID == nil {
				continue
			}
			k := wcDateKey{wcID: *op.WorkCenterID, date: day.Format("2006-01-02")}
			reqMap[k] += op.EffHours * order.Quantity
			dateMap[k] = day
		}
	}

	overloadCount := 0
	for k, reqHours := range reqMap {
		avail, _ := uc.repo.GetMachineAvailableHoursPerDay(ctx, k.wcID)
		if avail <= 0 {
			avail = 8
		}
		if uc.maintRepo != nil {
			if blocked, err := uc.maintRepo.GetBlockedHours(ctx, k.wcID, dateMap[k]); err == nil && blocked > 0 {
				avail -= blocked
				if avail < 0 {
					avail = 0
				}
			}
		}
		req := &entity.CapacityRequirement{
			PlanCode:       dto.PlanCode,
			WorkCenterID:   k.wcID,
			ReqDate:        dateMap[k],
			RequiredHours:  reqHours,
			AvailableHours: avail,
		}
		saved, err := uc.repo.UpsertRequirement(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("upserting CRP entry: %w", err)
		}
		if saved.LoadPct > 100 {
			overloadCount++
		}
	}

	return &response.CRPSummaryResponse{
		PlanCode:      dto.PlanCode,
		TotalEntries:  len(reqMap),
		OverloadCount: overloadCount,
	}, nil
}

func (uc *CRPUseCase) ListByPlan(ctx context.Context, planCode int64) ([]*response.CRPEntryResponse, error) {
	reqs, err := uc.repo.ListByPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return toCRPSlice(reqs), nil
}

func (uc *CRPUseCase) ListOverloadedByPlan(ctx context.Context, planCode int64) ([]*response.CRPEntryResponse, error) {
	reqs, err := uc.repo.ListOverloadedByPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return toCRPSlice(reqs), nil
}

func toCRPSlice(reqs []*entity.CapacityRequirement) []*response.CRPEntryResponse {
	out := make([]*response.CRPEntryResponse, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, &response.CRPEntryResponse{
			ID:             r.ID,
			PlanCode:       r.PlanCode,
			WorkCenterID:   r.WorkCenterID,
			ReqDate:        r.ReqDate,
			RequiredHours:  r.RequiredHours,
			AvailableHours: r.AvailableHours,
			LoadPct:        r.LoadPct,
			IsOverloaded:   r.LoadPct > 100,
		})
	}
	return out
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
