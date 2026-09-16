package crp_uc

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	crpentity "github.com/FelipePn10/panossoerp/internal/domain/crp/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/crp/repository"
	maintenancerepo "github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
)

// routeOpReader is the slice of the routing repository CRP needs to load capacity
// using the rich, quantity-aware time model (machine hours per work center).
type requirementWriter interface {
	ReplaceRequirements(context.Context, int64, []*crpentity.CapacityRequirement) error
}

type dailyCapacityReader interface {
	GetAvailableHoursOnDate(context.Context, int64, time.Time) (float64, error)
}

type routeOpReader interface {
	GetRouteOperations(ctx context.Context, routeID int64) ([]*routingentity.RouteOperation, error)
}

type CRPUseCase struct {
	repo      repository.CRPRepository
	maintRepo maintenancerepo.MaintenanceRepository
	routing   routeOpReader // optional; when nil, uses the legacy flat EffHours × qty
	// plans é o catálogo de planos do modal; também garante o isolamento por
	// empresa nas consultas de carga.
	plans planCatalog
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
	if err := uc.assertPlanTenant(ctx, dto.PlanCode); err != nil {
		return nil, err
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
	if reader, ok := uc.repo.(interface {
		GetScheduledMachineLoads(context.Context, int64) ([]repository.ScheduledLoad, error)
	}); ok {
		loads, err := reader.GetScheduledMachineLoads(ctx, dto.PlanCode)
		if err != nil {
			return nil, err
		}
		for _, load := range loads {
			k := wcDateKey{wcID: load.WorkCenterID, date: load.Day.Format("2006-01-02")}
			reqMap[k] += load.Hours
			dateMap[k] = load.Day
		}
	}
	// Capacidade por centro, resolvida uma vez só: antes era uma consulta por
	// par (centro, dia).
	var capacityErr error
	capacidadePorCentro := make(map[int64]float64)
	capacidade := func(wcID int64) float64 {
		if v, ok := capacidadePorCentro[wcID]; ok {
			return v
		}
		v, err := uc.repo.GetMachineAvailableHoursPerDay(ctx, wcID)
		if err != nil {
			capacityErr = err
		}
		capacidadePorCentro[wcID] = v
		return v
	}

	// Acumula a carga de uma operação nos dias que ela ocupa, em vez de jogar
	// tudo no dia da ordem.
	acumula := func(wcID int64, inicio time.Time, horas float64) {
		for _, c := range crpentity.DistribuiCarga(inicio, horas, capacidade(wcID)) {
			k := wcDateKey{wcID: wcID, date: c.Data.Format("2006-01-02")}
			reqMap[k] += c.Horas
			dateMap[k] = c.Data
		}
	}

	missingOrders := []int64{}
	for _, order := range orders {
		if order.RouteID == nil {
			if order.MachineWorkCenterID != nil && order.MachineHours > 0 {
				acumula(*order.MachineWorkCenterID, order.PlannedDate, order.MachineHours)
			} else {
				missingOrders = append(missingOrders, order.ID)
			}
			continue
		}
		day := truncateToDay(order.PlannedDate)

		// Preferred path: rich, quantity-aware machine hours (setup + run×batches)
		// charged to each operation's EFFECTIVE work center (override or op default).
		if uc.routing != nil {
			rops, err := uc.routing.GetRouteOperations(ctx, *order.RouteID)
			if err != nil {
				return nil, fmt.Errorf("carregando roteiro %d: %w", *order.RouteID, err)
			}
			if len(rops) == 0 {
				missingOrders = append(missingOrders, order.ID)
			}
			if err == nil {
				// Cada operação começa quando as anteriores terminam: a carga
				// segue o roteiro no tempo, não se amontoa num dia.
				inicioDaOperacao := day
				for _, op := range rops {
					if op.Situation == routingentity.RouteOpGhost || op.EffectiveWorkCenterID == nil {
						continue
					}
					wcID := *op.EffectiveWorkCenterID
					horas := op.EffTime.MachineHours(order.Quantity)
					acumula(wcID, inicioDaOperacao, horas)

					cap := capacidade(wcID)
					if cap <= 0 {
						cap = 8
					}
					dias := int(math.Ceil(horas / cap))
					if dias < 1 {
						dias = 1
					}
					inicioDaOperacao = inicioDaOperacao.AddDate(0, 0, dias)
				}
				continue
			}
		}

		// Legacy fallback: flat EffHours × quantity (setup not separated).
		ops, err := uc.repo.GetRouteOperationsByRoute(ctx, *order.RouteID)
		if err != nil {
			return nil, err
		}
		if len(ops) == 0 {
			missingOrders = append(missingOrders, order.ID)
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

	if capacityErr != nil {
		return nil, fmt.Errorf("carregando capacidade: %w", capacityErr)
	}
	requirements := []*crpentity.CapacityRequirement{}
	overloadCount := 0
	semCapacidade := map[int64]bool{}
	for k, reqHours := range reqMap {
		avail := capacidade(k.wcID)
		daily, hasDaily := uc.repo.(dailyCapacityReader)
		if hasDaily {
			var err error
			avail, err = daily.GetAvailableHoursOnDate(ctx, k.wcID, dateMap[k])
			if err != nil {
				return nil, fmt.Errorf("consultando capacidade diária: %w", err)
			}
		}
		if avail <= 0 {
			// Sem capacidade cadastrada não há como dizer se há sobrecarga. Antes
			// o sistema assumia 8 h em silêncio e o gráfico parecia confiável;
			// agora o centro é reportado para o usuário cadastrar a capacidade.
			semCapacidade[k.wcID] = true
			avail = 0
		}
		if uc.maintRepo != nil && !hasDaily {
			if blocked, err := uc.maintRepo.GetBlockedHours(ctx, k.wcID, dateMap[k]); err == nil && blocked > 0 {
				avail -= blocked
				if avail < 0 {
					avail = 0
				}
			}
		}
		req := &crpentity.CapacityRequirement{
			PlanCode:       dto.PlanCode,
			WorkCenterID:   k.wcID,
			ReqDate:        dateMap[k],
			RequiredHours:  reqHours,
			AvailableHours: avail,
		}
		requirements = append(requirements, req)
		if req.RequiredHours > req.AvailableHours {
			overloadCount++
		}

	}

	if writer, ok := uc.repo.(requirementWriter); ok {
		if err := writer.ReplaceRequirements(ctx, dto.PlanCode, requirements); err != nil {
			return nil, err
		}
	} else {
		if err := uc.repo.DeleteByPlan(ctx, dto.PlanCode); err != nil {
			return nil, err
		}
		for _, req := range requirements {
			if _, err := uc.repo.UpsertRequirement(ctx, req); err != nil {
				return nil, err
			}
		}
	}
	centrosSemCapacidade := make([]int64, 0, len(semCapacidade))
	for id := range semCapacidade {
		centrosSemCapacidade = append(centrosSemCapacidade, id)
	}
	sort.Slice(centrosSemCapacidade, func(i, j int) bool { return centrosSemCapacidade[i] < centrosSemCapacidade[j] })

	return &response.CRPSummaryResponse{
		OrdersWithoutLoad:     missingOrders,
		PlanCode:              dto.PlanCode,
		TotalEntries:          len(reqMap),
		OverloadCount:         overloadCount,
		WorkCentersNoCapacity: centrosSemCapacidade,
	}, nil
}

func (uc *CRPUseCase) ListByPlan(ctx context.Context, planCode int64) ([]*response.CRPEntryResponse, error) {
	if err := uc.assertPlanTenant(ctx, planCode); err != nil {
		return nil, err
	}
	reqs, err := uc.repo.ListByPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return toCRPSlice(reqs), nil
}

func (uc *CRPUseCase) ListOverloadedByPlan(ctx context.Context, planCode int64) ([]*response.CRPEntryResponse, error) {
	if err := uc.assertPlanTenant(ctx, planCode); err != nil {
		return nil, err
	}
	reqs, err := uc.repo.ListByPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	out := []*response.CRPEntryResponse{}
	for _, v := range toCRPSlice(reqs) {
		if v.IsOverloaded {
			out = append(out, v)
		}
	}
	return out, nil
}

func toCRPSlice(reqs []*crpentity.CapacityRequirement) []*response.CRPEntryResponse {
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
			IsOverloaded:   r.LoadPct > 100 || r.AvailableHours == 0 && r.RequiredHours > 0,
		})
	}
	return out
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
