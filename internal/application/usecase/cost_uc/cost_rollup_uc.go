package cost_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	"github.com/google/uuid"
)

// routingReader is the slice of the routing repository the cost roll-up needs to
// charge each operation at its real work-center rate using the rich time model.
type routingReader interface {
	GetRouteForItem(ctx context.Context, itemCode int64, mask string) (*routingentity.ManufacturingRoute, error)
	GetRouteOperations(ctx context.Context, routeID int64) ([]*routingentity.RouteOperation, error)
}

type StandardCostUseCase struct {
	repo    repository.StandardCostRepository
	routing routingReader // optional; when nil, falls back to the legacy average-rate estimate
}

func New(repo repository.StandardCostRepository) *StandardCostUseCase {
	return &StandardCostUseCase{repo: repo}
}

// WithRouting enables per-operation, per-work-center, quantity-aware labor costing.
func (uc *StandardCostUseCase) WithRouting(r routingReader) *StandardCostUseCase {
	uc.routing = r
	return uc
}

// ─── price catalog management ─────────────────────────────────────────────────

func (uc *StandardCostUseCase) UpsertWorkCenterCost(ctx context.Context, dto request.UpsertWorkCenterCostDTO) (*response.WorkCenterCostResponse, error) {
	uid, err := uuid.Parse(dto.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_by UUID: %w", err)
	}
	// When the machine × labor split is not provided, the machine rate defaults to the
	// blended cost_per_hour so the stored/displayed value matches the effective rate.
	machineRate := dto.MachineCostPerHour
	if machineRate <= 0 {
		machineRate = dto.CostPerHour
	}
	wcc := &entity.WorkCenterCost{
		WorkCenterID:       dto.WorkCenterID,
		CostPerHour:        dto.CostPerHour,
		MachineCostPerHour: machineRate,
		LaborCostPerHour:   dto.LaborCostPerHour,
		Currency:           dto.Currency,
		UpdatedBy:          uid,
	}
	saved, err := uc.repo.UpsertWorkCenterCost(ctx, wcc)
	if err != nil {
		return nil, err
	}
	return toWCCResponse(saved), nil
}

func (uc *StandardCostUseCase) ListWorkCenterCosts(ctx context.Context) ([]*response.WorkCenterCostResponse, error) {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.WorkCenterCostResponse, 0, len(wccs))
	for _, w := range wccs {
		out = append(out, toWCCResponse(w))
	}
	return out, nil
}

func (uc *StandardCostUseCase) UpsertItemPurchaseCost(ctx context.Context, dto request.UpsertItemPurchaseCostDTO) (*response.ItemPurchaseCostResponse, error) {
	uid, err := uuid.Parse(dto.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_by UUID: %w", err)
	}
	ipc := &entity.ItemPurchaseCost{
		ItemCode:  dto.ItemCode,
		UnitCost:  dto.UnitCost,
		Currency:  dto.Currency,
		UpdatedBy: uid,
	}
	saved, err := uc.repo.UpsertItemPurchaseCost(ctx, ipc)
	if err != nil {
		return nil, err
	}
	return toIPCResponse(saved), nil
}

func (uc *StandardCostUseCase) GetItemPurchaseCost(ctx context.Context, itemCode int64) (*response.ItemPurchaseCostResponse, error) {
	ipc, err := uc.repo.GetItemPurchaseCost(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("purchase cost not found: %w", err)
	}
	return toIPCResponse(ipc), nil
}

// ─── cost rollup ──────────────────────────────────────────────────────────────

// RollUp calculates and saves the standard cost for itemCode + mask using
// bottom-up BOM traversal:
//
//	material_cost = Σ (child.unit_cost × qty × (1 + loss%))
//	labor_cost    = Σ_ops [ MachineHours(lot)×machine_rate + LaborHours(lot)×labor_rate ] ÷ lot
//	overhead_cost = overhead_rate × (material_cost + labor_cost)   [currently 0 unless configured]
func (uc *StandardCostUseCase) RollUp(ctx context.Context, dto request.CostRollupDTO) (*response.CostRollupResponse, error) {
	calculatedBy, err := uuid.Parse(dto.CalculatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid calculated_by UUID: %w", err)
	}

	lotSize := dto.LotSize
	if lotSize <= 0 {
		lotSize = 1
	}

	unitCache := make(map[int64]float64)
	result, err := uc.rollupItem(ctx, dto.ItemCode, dto.Mask, 0, lotSize, unitCache)
	if err != nil {
		return nil, err
	}

	cost := &entity.ItemStandardCost{
		ItemCode:     dto.ItemCode,
		Mask:         dto.Mask,
		MaterialCost: result.MaterialCost,
		LaborCost:    result.LaborCost,
		OverheadCost: result.OverheadCost,
		Currency:     "BRL",
		CalculatedBy: calculatedBy,
	}
	saved, err := uc.repo.UpsertItemStandardCost(ctx, cost)
	if err != nil {
		return nil, fmt.Errorf("saving standard cost: %w", err)
	}

	for _, entry := range result.Detail {
		_ = uc.repo.InsertRollupLog(ctx, &entry)
	}

	return &response.CostRollupResponse{
		ItemCode:     saved.ItemCode,
		Mask:         saved.Mask,
		MaterialCost: saved.MaterialCost,
		LaborCost:    saved.LaborCost,
		OverheadCost: saved.OverheadCost,
		TotalCost:    saved.TotalCost,
		Currency:     saved.Currency,
		CalculatedAt: saved.CalculatedAt,
	}, nil
}

func (uc *StandardCostUseCase) GetStandardCost(ctx context.Context, itemCode int64, mask string) (*response.CostRollupResponse, error) {
	cost, err := uc.repo.GetItemStandardCost(ctx, itemCode, mask)
	if err != nil {
		return nil, fmt.Errorf("standard cost not found: %w", err)
	}
	return &response.CostRollupResponse{
		ItemCode:     cost.ItemCode,
		Mask:         cost.Mask,
		MaterialCost: cost.MaterialCost,
		LaborCost:    cost.LaborCost,
		OverheadCost: cost.OverheadCost,
		TotalCost:    cost.TotalCost,
		Currency:     cost.Currency,
		CalculatedAt: cost.CalculatedAt,
	}, nil
}

// ─── rollup algorithm ─────────────────────────────────────────────────────────

type costNode struct {
	MaterialCost float64
	LaborCost    float64
	OverheadCost float64
	Detail       []entity.CostRollupLogEntry
}

func (uc *StandardCostUseCase) rollupItem(ctx context.Context, itemCode int64, mask string, level int, lotSize float64, unitCache map[int64]float64) (*costNode, error) {
	children, err := uc.repo.GetDirectChildren(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching BOM for item %d: %w", itemCode, err)
	}

	var materialCost float64
	var childNodes []*costNode

	if len(children) == 0 {
		// Leaf node — use purchase cost
		if cached, ok := unitCache[itemCode]; ok {
			materialCost = cached
		} else {
			ipc, err2 := uc.repo.GetItemPurchaseCost(ctx, itemCode)
			if err2 == nil {
				materialCost = ipc.UnitCost
			}
			unitCache[itemCode] = materialCost
		}
	} else {
		// Manufactured item — recurse into children
		for _, child := range children {
			childNode, err2 := uc.rollupItem(ctx, child.ChildCode, mask, level+1, lotSize, unitCache)
			if err2 != nil {
				return nil, err2
			}
			netQty := child.Quantity * (1 + child.LossPercentage/100)
			materialCost += childNode.total() * netQty
			childNodes = append(childNodes, childNode)
		}
	}

	// Conversion (labor + machine) cost per unit, setup amortized over the reference lot.
	laborCost := uc.conversionCost(ctx, itemCode, mask, lotSize)

	node := &costNode{
		MaterialCost: materialCost,
		LaborCost:    laborCost,
		OverheadCost: 0,
		Detail: []entity.CostRollupLogEntry{{
			ItemCode:     itemCode,
			Mask:         mask,
			BOMLevel:     level,
			MaterialCost: materialCost,
			LaborCost:    laborCost,
			OverheadCost: 0,
		}},
	}
	for _, cn := range childNodes {
		node.Detail = append(node.Detail, cn.Detail...)
	}

	return node, nil
}

func (n *costNode) total() float64 {
	return n.MaterialCost + n.LaborCost + n.OverheadCost
}

// conversionCost is the per-unit machine + labor cost of the item's standard route.
// Each operation is charged at ITS OWN work-center rate using the rich time model,
// with setup amortized over the reference lot:
//
//	( Σ [ MachineHours(lot) × machineRate(CT) + LaborHours(lot) × laborRate(CT) ] ) ÷ lot
//
// A lot of 1 charges the full setup per unit (conservative). When the routing
// repository is not wired, it falls back to the legacy average-rate estimate.
func (uc *StandardCostUseCase) conversionCost(ctx context.Context, itemCode int64, mask string, lot float64) float64 {
	if lot <= 0 {
		lot = 1
	}
	if uc.routing == nil {
		routeHours, _ := uc.repo.GetRouteHoursByItem(ctx, itemCode, mask)
		return routeHours * uc.averageRate(ctx)
	}

	route, err := uc.routing.GetRouteForItem(ctx, itemCode, mask)
	if err != nil || route == nil {
		return 0 // no route → no conversion cost
	}
	ops, err := uc.routing.GetRouteOperations(ctx, route.ID)
	if err != nil || len(ops) == 0 {
		return 0
	}

	rates := uc.workCenterRates(ctx) // wcID → cost

	var total float64
	for _, op := range ops {
		if op.Situation == routingentity.RouteOpGhost {
			continue // phantom operation: no cost
		}
		machineRate, laborRate := 0.0, 0.0
		if op.EffectiveWorkCenterID != nil {
			if wc, ok := rates[*op.EffectiveWorkCenterID]; ok {
				machineRate = wc.MachineRate()
				laborRate = wc.LaborRate()
			}
		}
		total += op.EffTime.MachineHours(lot)*machineRate + op.EffTime.LaborHours(lot)*laborRate
	}
	return total / lot
}

// workCenterRates indexes the configured work-center costs by work-center id.
func (uc *StandardCostUseCase) workCenterRates(ctx context.Context) map[int64]*entity.WorkCenterCost {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	m := make(map[int64]*entity.WorkCenterCost, len(wccs))
	if err != nil {
		return m
	}
	for _, w := range wccs {
		m[w.WorkCenterID] = w
	}
	return m
}

// averageRate is the legacy fallback used only when the routing repo is not wired.
func (uc *StandardCostUseCase) averageRate(ctx context.Context) float64 {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	if err != nil || len(wccs) == 0 {
		return 0
	}
	var total float64
	for _, w := range wccs {
		total += w.CostPerHour
	}
	return total / float64(len(wccs))
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toWCCResponse(w *entity.WorkCenterCost) *response.WorkCenterCostResponse {
	return &response.WorkCenterCostResponse{
		ID:                 w.ID,
		WorkCenterID:       w.WorkCenterID,
		CostPerHour:        w.CostPerHour,
		MachineCostPerHour: w.MachineCostPerHour,
		LaborCostPerHour:   w.LaborCostPerHour,
		Currency:           w.Currency,
		UpdatedAt:          w.UpdatedAt,
	}
}

func toIPCResponse(i *entity.ItemPurchaseCost) *response.ItemPurchaseCostResponse {
	return &response.ItemPurchaseCostResponse{
		ID:        i.ID,
		ItemCode:  i.ItemCode,
		UnitCost:  i.UnitCost,
		Currency:  i.Currency,
		UpdatedAt: i.UpdatedAt,
	}
}
