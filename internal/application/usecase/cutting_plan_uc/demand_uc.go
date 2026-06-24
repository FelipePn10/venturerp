package cutting_plan_uc

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	cprepo "github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	prodrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	structqueryrepo "github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository"
)

// rawMaterialLLC is the level (per the item Planning convention) that marks an
// item as a raw material to be cut.
const rawMaterialLLC = 9

// DemandUseCase generates cutting demand automatically from production and planned
// orders. It explodes each order's BOM, turns every component that is a cut piece
// (has dimensions + a resolvable raw material) into a demanded part, and aggregates
// parts of the SAME raw material — across many orders — into one cutting plan, so
// the optimiser can nest the whole batch together for the best yield.
type DemandUseCase struct {
	cutting    cprepo.CuttingPlanRepository
	items      itemrepo.ItemRepository
	structures structqueryrepo.StructureQueryRepository
	prodOrders prodrepo.ProductionOrderRepository
	planned    plannedrepo.PlannedOrderRepository
}

func NewDemandUseCase(
	cutting cprepo.CuttingPlanRepository,
	items itemrepo.ItemRepository,
	structures structqueryrepo.StructureQueryRepository,
	prodOrders prodrepo.ProductionOrderRepository,
	planned plannedrepo.PlannedOrderRepository,
) *DemandUseCase {
	return &DemandUseCase{cutting: cutting, items: items, structures: structures, prodOrders: prodOrders, planned: planned}
}

// orderSource is one exploded order (its product, mask and quantity).
type orderSource struct {
	ref    string // human reference (OP/planned number)
	isOP   bool
	opCode int64 // production_order code, for tying the baixa to a single OP
	item   int64
	mask   string
	qty    float64
}

// matGroup aggregates the cut parts of one raw material across orders.
type matGroup struct {
	material int64
	parts    []*entity.CuttingPlanPart
	refs     map[string]struct{}
	opCodes  map[int64]struct{}
}

// GenerateFromOrders builds one cutting plan per raw material from the given orders.
func (uc *DemandUseCase) GenerateFromOrders(ctx context.Context, dto request.GenerateCuttingFromOrdersDTO) (*response.GenerateCuttingDemandResponse, error) {
	if len(dto.ProductionOrderCodes) == 0 && len(dto.PlannedOrderCodes) == 0 {
		return nil, errors.New("at least one production_order_code or planned_order_code is required")
	}
	if dto.MinRemnantMM < 0 || dto.KerfMM < 0 || dto.TrimMM < 0 {
		return nil, errors.New("kerf, trim and min_remnant cannot be negative")
	}

	// Resolve the orders to explode.
	var sources []orderSource
	var warnings []string
	for _, code := range dto.ProductionOrderCodes {
		op, err := uc.prodOrders.GetByCode(ctx, code)
		if err != nil || op == nil {
			warnings = append(warnings, fmt.Sprintf("production order %d not found", code))
			continue
		}
		sources = append(sources, orderSource{
			ref: fmt.Sprintf("OP-%d", op.OrderNumber), isOP: true, opCode: op.OrderNumber,
			item: op.ItemCode, mask: op.Mask, qty: op.PlannedQty,
		})
	}
	for _, code := range dto.PlannedOrderCodes {
		po, err := uc.planned.GetByCode(ctx, code)
		if err != nil || po == nil {
			warnings = append(warnings, fmt.Sprintf("planned order %d not found", code))
			continue
		}
		if po.OrderType != types.OrderProduction {
			warnings = append(warnings, fmt.Sprintf("planned order %d is not a production order; skipped", code))
			continue
		}
		mask := ""
		if po.Mask != nil {
			mask = *po.Mask
		}
		qty := po.QuantityCorrected
		if qty <= 0 {
			qty = po.Quantity
		}
		sources = append(sources, orderSource{
			ref: fmt.Sprintf("PLAN-%d", po.OrderNumber), item: po.ItemCode, mask: mask, qty: qty,
		})
	}

	// Explode every order and group cut parts by raw material.
	itemCache := map[int64]*itementity.Item{}
	groups := map[int64]*matGroup{}
	var order []int64

	for _, s := range sources {
		children, err := uc.structures.GetDirectChildrenForMask(ctx, s.item, s.mask)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: exploding BOM failed: %v", s.ref, err))
			continue
		}
		for _, st := range children {
			child, err := uc.loadItem(ctx, itemCache, st.ChildCode)
			if err != nil || child == nil {
				continue
			}
			dims := child.Engineering.Dimensions
			if dims == nil || (dims.Length <= 0 && dims.Width <= 0) {
				continue // not a cut piece (hardware, fastener, …)
			}
			material, ok := uc.resolveMaterial(ctx, itemCache, child)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("%s: component %d has cut dimensions but no resolvable raw material", s.ref, st.ChildCode))
				continue
			}
			matItem, err := uc.loadItem(ctx, itemCache, material)
			if err != nil || matItem == nil {
				warnings = append(warnings, fmt.Sprintf("%s: raw material %d not found", s.ref, material))
				continue
			}
			qty := int(math.Ceil(s.qty * st.EffectiveQuantity()))
			if qty <= 0 {
				continue
			}
			part, perr := buildAutoPart(child, st.ChildDescription, qty, s.ref, materialIs2D(matItem), dto.AllowRotation)
			if perr != nil {
				warnings = append(warnings, fmt.Sprintf("%s: component %d: %v", s.ref, st.ChildCode, perr))
				continue
			}

			g, exists := groups[material]
			if !exists {
				g = &matGroup{material: material, refs: map[string]struct{}{}, opCodes: map[int64]struct{}{}}
				groups[material] = g
				order = append(order, material)
			}
			g.parts = append(g.parts, part)
			g.refs[s.ref] = struct{}{}
			if s.isOP {
				g.opCodes[s.opCode] = struct{}{}
			}
		}
	}

	// Persist one plan per material.
	resp := &response.GenerateCuttingDemandResponse{Warnings: warnings}
	for _, mat := range order {
		g := groups[mat]
		summary, err := uc.createPlan(ctx, g, dto)
		if err != nil {
			return nil, err
		}
		resp.Plans = append(resp.Plans, *summary)
	}
	if len(resp.Plans) == 0 {
		resp.Warnings = append(resp.Warnings, "no cutting demand could be generated from the given orders")
	}
	return resp, nil
}

func (uc *DemandUseCase) createPlan(ctx context.Context, g *matGroup, dto request.GenerateCuttingFromOrdersDTO) (*response.GeneratedCuttingPlanSummary, error) {
	matItem, err := uc.loadItem(ctx, map[int64]*itementity.Item{}, g.material)
	if err != nil {
		return nil, err
	}
	cutType := entity.CutTypeLinear1D
	if materialIs2D(matItem) {
		cutType = entity.CutTypeGuillotine2D
	}

	code, err := uc.cutting.NextPlanCode(ctx)
	if err != nil {
		return nil, err
	}
	refs := sortedKeys(g.refs)
	desc := fmt.Sprintf("Auto-gerado de %d ordem(ns): %s", len(refs), joinRefs(refs))
	plan, err := entity.NewCuttingPlan(code, &desc, cutType, entity.SourceProductionOrder, g.material, nil, dto.KerfMM, dto.TrimMM, dto.MinRemnantMM, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	plan.WarehouseID = dto.WarehouseID
	plan.IncludeRemnants = dto.IncludeRemnants
	plan.StockUoM = matItem.Warehouse.UnitOfMeasurement
	if plan.StockUoM == "" {
		plan.StockUoM = types.UN
	}
	// Tie the baixa to the OP only when a single production order fed this material.
	if len(g.opCodes) == 1 {
		for c := range g.opCodes {
			cc := c
			plan.ProductionOrderCode = &cc
		}
	}

	created, err := uc.cutting.CreatePlan(ctx, plan)
	if err != nil {
		return nil, err
	}
	totalPieces := 0
	for _, p := range g.parts {
		p.PlanID = created.ID
		if _, err := uc.cutting.AddPart(ctx, p); err != nil {
			return nil, err
		}
		totalPieces += p.Quantity
	}

	return &response.GeneratedCuttingPlanSummary{
		PlanID:           created.ID,
		PlanCode:         created.Code,
		CutType:          string(cutType),
		MaterialItemCode: g.material,
		PartCount:        len(g.parts),
		TotalPieces:      totalPieces,
		OrderRefs:        refs,
	}, nil
}

// resolveMaterial finds the raw material a cut component is made from: a BOM child
// of the component flagged as raw material (LLC 9), else its single BOM child, else
// the component's item-base code.
func (uc *DemandUseCase) resolveMaterial(ctx context.Context, cache map[int64]*itementity.Item, child *itementity.Item) (int64, bool) {
	kids, err := uc.structures.GetDirectChildrenForMask(ctx, int64(child.Code), "")
	if err == nil {
		for _, k := range kids {
			ki, kerr := uc.loadItem(ctx, cache, k.ChildCode)
			if kerr == nil && ki != nil && ki.Planning.LLC == rawMaterialLLC {
				return k.ChildCode, true
			}
		}
		if len(kids) == 1 {
			return kids[0].ChildCode, true
		}
	}
	if child.Engineering.ItemBaseCod != nil {
		return int64(*child.Engineering.ItemBaseCod), true
	}
	return 0, false
}

func (uc *DemandUseCase) loadItem(ctx context.Context, cache map[int64]*itementity.Item, code int64) (*itementity.Item, error) {
	if it, ok := cache[code]; ok {
		return it, nil
	}
	ic, err := valueobject.NewItemCode(code)
	if err != nil {
		return nil, err
	}
	it, err := uc.items.FindItemByCode(ctx, ic)
	if err != nil {
		return nil, err
	}
	cache[code] = it
	return it, nil
}

// materialIs2D decides whether a raw material is cut as a sheet (2D) or a bar (1D),
// from its stock unit of measure.
func materialIs2D(mat *itementity.Item) bool {
	switch mat.Warehouse.UnitOfMeasurement {
	case types.M2, types.M3:
		return true
	default:
		return false
	}
}

// buildAutoPart maps a component's dimensions into a 1D or 2D cutting part. For a
// sheet the face is taken as Length×Width (Height = thickness, ignored).
func buildAutoPart(child *itementity.Item, label string, qty int, sourceRef string, is2D, allowRotation bool) (*entity.CuttingPlanPart, error) {
	d := child.Engineering.Dimensions
	itemCode := int64(child.Code)
	ref := sourceRef
	if label == "" {
		label = fmt.Sprintf("item %d", itemCode)
	}
	if is2D {
		w, h := float64(d.Length), float64(d.Width)
		if w <= 0 || h <= 0 {
			return nil, errors.New("sheet component needs Length and Width dimensions")
		}
		return entity.NewPart2D(0, &itemCode, label, w, h, entity.GrainNone, allowRotation, qty, &ref)
	}
	length := float64(d.Length)
	if length <= 0 {
		length = float64(d.Width)
	}
	if length <= 0 {
		return nil, errors.New("bar component needs a Length dimension")
	}
	return entity.NewPart(0, &itemCode, label, length, qty, &ref)
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// small, stable insertion sort to avoid importing sort for a handful of refs
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func joinRefs(refs []string) string {
	s := ""
	for i, r := range refs {
		if i > 0 {
			s += ", "
		}
		s += r
	}
	return s
}
