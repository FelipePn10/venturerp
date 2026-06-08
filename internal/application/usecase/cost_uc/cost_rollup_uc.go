package cost_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	"github.com/google/uuid"
)

type StandardCostUseCase struct {
	repo repository.StandardCostRepository
}

func New(repo repository.StandardCostRepository) *StandardCostUseCase {
	return &StandardCostUseCase{repo: repo}
}

// ─── price catalog management ─────────────────────────────────────────────────

func (uc *StandardCostUseCase) UpsertWorkCenterCost(ctx context.Context, dto request.UpsertWorkCenterCostDTO) (*response.WorkCenterCostResponse, error) {
	uid, err := uuid.Parse(dto.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid updated_by UUID: %w", err)
	}
	wcc := &entity.WorkCenterCost{
		WorkCenterID: dto.WorkCenterID,
		CostPerHour:  dto.CostPerHour,
		Currency:     dto.Currency,
		UpdatedBy:    uid,
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
//	labor_cost    = route_hours × work_center_cost_per_hour
//	overhead_cost = overhead_rate × (material_cost + labor_cost)   [currently 0 unless configured]
func (uc *StandardCostUseCase) RollUp(ctx context.Context, dto request.CostRollupDTO) (*response.CostRollupResponse, error) {
	calculatedBy, err := uuid.Parse(dto.CalculatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid calculated_by UUID: %w", err)
	}

	unitCache := make(map[int64]float64)
	result, err := uc.rollupItem(ctx, dto.ItemCode, dto.Mask, 0, unitCache)
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

func (uc *StandardCostUseCase) rollupItem(ctx context.Context, itemCode int64, mask string, level int, unitCache map[int64]float64) (*costNode, error) {
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
			childNode, err2 := uc.rollupItem(ctx, child.ChildCode, mask, level+1, unitCache)
			if err2 != nil {
				return nil, err2
			}
			netQty := child.Quantity * (1 + child.LossPercentage/100)
			materialCost += childNode.total() * netQty
			childNodes = append(childNodes, childNode)
		}
	}

	// Labor cost: route hours × CT cost per hour
	routeHours, _ := uc.repo.GetRouteHoursByItem(ctx, itemCode, mask)
	laborCost := routeHours * uc.workCenterCostPerHour(ctx)

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

// workCenterCostPerHour looks up the average CT cost for the item's route.
// Returns 0 if no work center cost is configured (safe default).
func (uc *StandardCostUseCase) workCenterCostPerHour(ctx context.Context) float64 {
	wccs, err := uc.repo.ListWorkCenterCosts(ctx)
	if err != nil || len(wccs) == 0 {
		return 0
	}
	// Simple average — in a real scenario we'd join by the CT used in the route.
	var total float64
	for _, w := range wccs {
		total += w.CostPerHour
	}
	return total / float64(len(wccs))
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toWCCResponse(w *entity.WorkCenterCost) *response.WorkCenterCostResponse {
	return &response.WorkCenterCostResponse{
		ID:           w.ID,
		WorkCenterID: w.WorkCenterID,
		CostPerHour:  w.CostPerHour,
		Currency:     w.Currency,
		UpdatedAt:    w.UpdatedAt,
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
