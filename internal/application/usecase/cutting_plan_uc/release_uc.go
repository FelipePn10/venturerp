package cutting_plan_uc

import (
	"context"
	"fmt"
	"math"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

const matchEps = 1e-4

// seedRemnants refreshes the plan's auto-loaded remnant stock pieces with the
// material's currently-available remnants, so the optimiser can reuse offcuts.
// It is idempotent: stale remnant-backed stock pieces are cleared first.
func (uc *CuttingPlanUseCase) seedRemnants(ctx context.Context, plan *entity.CuttingPlan) error {
	if err := uc.repo.DeleteRemnantStockPieces(ctx, plan.ID); err != nil {
		return err
	}
	remnants, err := uc.repo.ListAvailableRemnants(ctx, plan.MaterialItemCode, *plan.WarehouseID)
	if err != nil {
		return err
	}
	for _, rem := range remnants {
		id := rem.ID
		sp := &entity.CuttingStockPiece{
			PlanID:     plan.ID,
			LengthMM:   rem.LengthMM,
			WidthMM:    rem.WidthMM,
			HeightMM:   rem.HeightMM,
			Quantity:   1,
			Lot:        rem.Lot,
			IsRemnant:  true,
			RemnantID:  &id,
			HeatNumber: rem.HeatNumber,
		}
		if _, err := uc.repo.AddStockPiece(ctx, sp); err != nil {
			return err
		}
	}
	return nil
}

// stockUnit is one physical available piece, expanded from the plan's stock rows.
type stockUnit struct {
	length    float64
	width     float64 // 2D
	height    float64 // 2D
	isRemnant bool
	remnantID *int64
	lot       *string
	heat      *string
}

// ReleasePlan firms an optimised plan: it consumes the stock the patterns require
// (posting real stock OUT movements for full bars, marking inventory remnants
// consumed), generates the reusable remnants the cut leaves behind, records the
// consumption trail and flips the plan to FIRMADO. Consumption mode (automatic
// FIFO vs manual lot) follows the plan override or the company default.
func (uc *CuttingPlanUseCase) ReleasePlan(ctx context.Context, planID int64) (*response.CuttingPlanReleaseResponse, error) {
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.Status == entity.PlanStatusReleased {
		return nil, fmt.Errorf("plan already firmed")
	}
	if plan.Status != entity.PlanStatusOptimized {
		return nil, fmt.Errorf("plan must be optimised before firming (status=%s)", plan.Status)
	}

	settings, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	warehouse := plan.WarehouseID
	if warehouse == nil {
		warehouse = settings.DefaultWarehouseID
	}
	if warehouse == nil {
		return nil, fmt.Errorf("no warehouse set on plan or company settings to post the baixa against")
	}
	mode := plan.EffectiveConsumptionMode(settings)

	patterns, err := uc.repo.ListPatterns(ctx, planID)
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 {
		return nil, fmt.Errorf("nothing to firm; optimise the plan first")
	}
	stockPieces, err := uc.repo.ListStockPieces(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Both sheet-based cut types (guillotine and true-shape) consume rectangular
	// sheets and post an area-based baixa.
	is2D := plan.CutType == entity.CutTypeGuillotine2D || plan.CutType == entity.CutTypeTrueShape2D

	// Expand stock rows into individual available units.
	var units []stockUnit
	for _, s := range stockPieces {
		for i := 0; i < s.Quantity; i++ {
			units = append(units, stockUnit{
				length: s.LengthMM, width: s.WidthMM, height: s.HeightMM,
				isRemnant: s.IsRemnant, remnantID: s.RemnantID, lot: s.Lot, heat: s.HeatNumber,
			})
		}
	}

	// FIFO lot queue + average cost fallback (for full-bar consumption).
	var lotQ []*entity.LotAvailability
	if mode == entity.ConsumptionAutomatic {
		if lotQ, err = uc.repo.ListAvailableLotsFIFO(ctx, plan.MaterialItemCode, *warehouse); err != nil {
			return nil, err
		}
	}
	avgCost := 0.0
	if bal, berr := uc.stock.GetBalance(ctx, plan.MaterialItemCode, "", *warehouse); berr == nil && bal != nil {
		avgCost = bal.AvgCost
	}

	refType := stockentity.ReferenceTypeManual
	refCode := plan.Code
	if plan.ProductionOrderCode != nil {
		refType = stockentity.ReferenceTypeProductionOrder
		refCode = *plan.ProductionOrderCode
	}

	var (
		consumedRemnantIDs []int64
		newRemnants        []*entity.StockRemnant
		consumptions       []*entity.CuttingPlanConsumption
		barsConsumed       int
		remnantsConsumed   int
		remnantsGenerated  int
	)

	for _, pat := range patterns {
		for i := 0; i < pat.RepeatCount; i++ {
			var (
				u        *stockUnit
				pieceQty float64
				qErr     error
			)
			if is2D {
				u = popUnit2D(&units, pat.StockWidthMM, pat.StockHeightMM, pat.IsRemnant)
				if u == nil {
					return nil, fmt.Errorf("stock mismatch: no available %.0f×%.0fmm sheet for a pattern (re-optimise the plan)", pat.StockWidthMM, pat.StockHeightMM)
				}
				pieceQty, qErr = service.StockQtyForArea(plan.StockUoM, u.width, u.height, plan.UoMFactor)
			} else {
				u = popUnit(&units, pat.StockLengthMM, pat.IsRemnant)
				if u == nil {
					return nil, fmt.Errorf("stock mismatch: no available %.0fmm piece for a pattern (re-optimise the plan)", pat.StockLengthMM)
				}
				pieceQty, qErr = service.StockQtyForLength(plan.StockUoM, u.length, plan.UoMFactor)
			}
			if qErr != nil {
				return nil, fmt.Errorf("converting piece to stock UoM: %w", qErr)
			}

			var parentLot, parentHeat, parentCert *string
			var unitCost float64 // cost per stock UoM (e.g. per metre / per kg / per piece)

			if u.isRemnant && u.remnantID != nil {
				rem, gerr := uc.repo.GetRemnant(ctx, *u.remnantID)
				if gerr != nil {
					return nil, fmt.Errorf("loading remnant %d: %w", *u.remnantID, gerr)
				}
				consumedRemnantIDs = append(consumedRemnantIDs, rem.ID)
				unitCost = rem.UnitCost
				parentLot, parentHeat, parentCert = rem.Lot, rem.HeatNumber, rem.Certificate
				rid := rem.ID
				consumptions = append(consumptions, &entity.CuttingPlanConsumption{
					PlanID: planID, ItemCode: plan.MaterialItemCode, SourceType: entity.ConsumptionSourceRemnant,
					RemnantID: &rid, Quantity: pieceQty, LengthMM: u.length, UnitCost: unitCost, TotalCost: unitCost * pieceQty,
					WarehouseID: *warehouse,
				})
				remnantsConsumed++
			} else {
				lot, heat, cert, cost, rerr := uc.resolveLot(ctx, mode, u, &lotQ, avgCost, plan.MaterialItemCode)
				if rerr != nil {
					return nil, rerr
				}
				total := cost * pieceQty
				mv := &stockentity.StockMovement{
					ItemCode: plan.MaterialItemCode, Mask: "", WarehouseID: *warehouse,
					MovementType: stockentity.MovementTypeOut, Quantity: pieceQty, UnitPrice: cost, TotalPrice: total,
					ReferenceType: &refType, ReferenceCode: &refCode, Lot: lot,
					Notes: strPtr(fmt.Sprintf("Plano de corte #%d", plan.Code)), CreatedBy: plan.CreatedBy,
				}
				created, merr := uc.stock.CreateMovement(ctx, mv)
				if merr != nil {
					return nil, fmt.Errorf("posting stock baixa: %w", merr)
				}
				unitCost = cost
				parentLot, parentHeat, parentCert = lot, heat, cert
				mvID := created.ID
				consumptions = append(consumptions, &entity.CuttingPlanConsumption{
					PlanID: planID, ItemCode: plan.MaterialItemCode, SourceType: entity.ConsumptionSourceLot,
					Lot: lot, Quantity: pieceQty, LengthMM: u.length, UnitCost: unitCost, TotalCost: total,
					WarehouseID: *warehouse, MovementID: &mvID,
				})
				barsConsumed++
			}

			// Reusable remnant generated by this cut (inherits parent traceability and
			// the per-UoM cost). 2D keeps the largest leftover rectangle when both
			// sides clear the minimum; 1D keeps the bar's leftover length.
			if plan.MinRemnantMM > 0 {
				var rem *entity.StockRemnant
				if is2D {
					if pat.RemnantWidthMM >= plan.MinRemnantMM-matchEps && pat.RemnantHeightMM >= plan.MinRemnantMM-matchEps {
						rem = &entity.StockRemnant{
							ItemCode: plan.MaterialItemCode, WarehouseID: *warehouse,
							WidthMM: pat.RemnantWidthMM, HeightMM: pat.RemnantHeightMM,
							Lot: parentLot, HeatNumber: parentHeat, Certificate: parentCert,
							Status: entity.RemnantAvailable, UnitCost: unitCost, OriginPlanID: &planID, CreatedBy: plan.CreatedBy,
						}
					}
				} else if pat.RemnantMM >= plan.MinRemnantMM-matchEps {
					rem = &entity.StockRemnant{
						ItemCode: plan.MaterialItemCode, WarehouseID: *warehouse, LengthMM: pat.RemnantMM,
						Lot: parentLot, HeatNumber: parentHeat, Certificate: parentCert,
						Status: entity.RemnantAvailable, UnitCost: unitCost, OriginPlanID: &planID, CreatedBy: plan.CreatedBy,
					}
				}
				if rem != nil {
					newRemnants = append(newRemnants, rem)
					remnantsGenerated++
				}
			}
		}
	}

	if err := uc.repo.CommitRelease(ctx, planID, consumedRemnantIDs, newRemnants, consumptions); err != nil {
		return nil, err
	}

	// Allocate the consumed material cost back to each source order, proportional
	// to that order's demand — so an aggregated (multi-OP) plan still costs per OP.
	if parts, perr := uc.repo.ListParts(ctx, planID); perr == nil {
		if costs := allocateOrderCosts(parts, consumptions, is2D); len(costs) > 0 {
			for _, c := range costs {
				c.PlanID = planID
			}
			if err := uc.repo.ReplaceOrderCosts(ctx, planID, costs); err != nil {
				return nil, err
			}
		}
	}

	return &response.CuttingPlanReleaseResponse{
		PlanID:            planID,
		PlanCode:          plan.Code,
		Status:            string(entity.PlanStatusReleased),
		ConsumptionMode:   string(mode),
		WarehouseID:       *warehouse,
		BarsConsumed:      barsConsumed,
		RemnantsConsumed:  remnantsConsumed,
		RemnantsGenerated: remnantsGenerated,
	}, nil
}

// resolveLot picks the lot to consume for a full-bar unit.
//   - MANUAL: the lot assigned on the stock piece (required); heat/cert from the
//     lot registry.
//   - AUTOMATIC: the next FIFO lot with remaining quantity; falls back to a
//     non-lot baixa at average cost when no lots are on hand.
func (uc *CuttingPlanUseCase) resolveLot(
	ctx context.Context, mode entity.ConsumptionMode, u *stockUnit,
	lotQ *[]*entity.LotAvailability, avgCost float64, itemCode int64,
) (lot, heat, cert *string, cost float64, err error) {
	if mode == entity.ConsumptionManual {
		if u.lot == nil || *u.lot == "" {
			return nil, nil, nil, 0, fmt.Errorf("manual consumption: a stock piece has no lot assigned")
		}
		cost = avgCost
		h, c := u.heat, (*string)(nil)
		if reg, gerr := uc.stock.GetLot(ctx, itemCode, *u.lot); gerr == nil && reg != nil {
			h, c = reg.HeatNumber, reg.Certificate
		}
		return u.lot, h, c, cost, nil
	}

	// AUTOMATIC: consume the front lot with remaining quantity.
	for _, la := range *lotQ {
		if la.Quantity >= 1-matchEps {
			la.Quantity -= 1
			cost = la.LastCost
			if cost <= 0 {
				cost = avgCost
			}
			lotVal := la.Lot
			return &lotVal, la.HeatNumber, la.Certificate, cost, nil
		}
	}
	// No lots on hand: generic baixa at average cost (no lot tag).
	return nil, nil, nil, avgCost, nil
}

// popUnit removes and returns an available stock unit matching the pattern's
// stock length and remnant flag, or nil if none remains.
func popUnit(units *[]stockUnit, length float64, isRemnant bool) *stockUnit {
	for i := range *units {
		u := (*units)[i]
		if u.isRemnant == isRemnant && math.Abs(u.length-length) < matchEps {
			*units = append((*units)[:i], (*units)[i+1:]...)
			return &u
		}
	}
	// Fall back to a length match ignoring the remnant flag, so a manually-entered
	// stock piece still satisfies a pattern when flags drift.
	for i := range *units {
		u := (*units)[i]
		if math.Abs(u.length-length) < matchEps {
			*units = append((*units)[:i], (*units)[i+1:]...)
			return &u
		}
	}
	return nil
}

// popUnit2D removes and returns an available sheet matching the pattern's stock
// width and height (and remnant flag), or nil if none remains.
func popUnit2D(units *[]stockUnit, width, height float64, isRemnant bool) *stockUnit {
	for i := range *units {
		u := (*units)[i]
		if u.isRemnant == isRemnant && math.Abs(u.width-width) < matchEps && math.Abs(u.height-height) < matchEps {
			*units = append((*units)[:i], (*units)[i+1:]...)
			return &u
		}
	}
	for i := range *units {
		u := (*units)[i]
		if math.Abs(u.width-width) < matchEps && math.Abs(u.height-height) < matchEps {
			*units = append((*units)[:i], (*units)[i+1:]...)
			return &u
		}
	}
	return nil
}

// allocateOrderCosts splits the total consumed material cost across the source
// orders of the plan's parts, proportional to each order's demand (length for 1D,
// area for 2D/true-shape). Parts without a source order are ignored.
func allocateOrderCosts(parts []*entity.CuttingPlanPart, consumptions []*entity.CuttingPlanConsumption, is2D bool) []*entity.CuttingPlanOrderCost {
	demand := map[string]float64{}
	var refsOrder []string
	for _, p := range parts {
		if p.SourceRef == nil || *p.SourceRef == "" {
			continue
		}
		var m float64
		if is2D {
			m = p.WidthMM * p.HeightMM * float64(p.Quantity)
		} else {
			m = p.LengthMM * float64(p.Quantity)
		}
		if _, ok := demand[*p.SourceRef]; !ok {
			refsOrder = append(refsOrder, *p.SourceRef)
		}
		demand[*p.SourceRef] += m
	}
	var totalDemand float64
	for _, m := range demand {
		totalDemand += m
	}
	if totalDemand <= 0 {
		return nil
	}
	var totalCost float64
	for _, c := range consumptions {
		totalCost += c.TotalCost
	}
	out := make([]*entity.CuttingPlanOrderCost, 0, len(refsOrder))
	for _, ref := range refsOrder {
		m := demand[ref]
		out = append(out, &entity.CuttingPlanOrderCost{
			OrderRef: ref, DemandMeasure: m, AllocatedCost: totalCost * m / totalDemand,
		})
	}
	return out
}

func strPtr(s string) *string { return &s }

// ListOrderCosts returns the per-order cost allocation of a firmed plan.
func (uc *CuttingPlanUseCase) ListOrderCosts(ctx context.Context, planID int64) ([]*response.CuttingPlanOrderCostResponse, error) {
	costs, err := uc.repo.ListOrderCosts(ctx, planID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CuttingPlanOrderCostResponse, 0, len(costs))
	for _, c := range costs {
		out = append(out, &response.CuttingPlanOrderCostResponse{
			OrderRef: c.OrderRef, DemandMeasure: c.DemandMeasure, AllocatedCost: c.AllocatedCost,
		})
	}
	return out, nil
}

// ─── settings ─────────────────────────────────────────────────────────────────

func (uc *CuttingPlanUseCase) GetSettings(ctx context.Context) (*response.CuttingSettingsResponse, error) {
	s, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return toSettingsResponse(s), nil
}

func (uc *CuttingPlanUseCase) UpdateSettings(ctx context.Context, dto request.CuttingSettingsDTO) (*response.CuttingSettingsResponse, error) {
	mode := entity.ConsumptionMode(dto.DefaultConsumptionMode)
	if mode != entity.ConsumptionAutomatic && mode != entity.ConsumptionManual {
		return nil, fmt.Errorf("invalid default_consumption_mode %q (AUTOMATIC|MANUAL)", dto.DefaultConsumptionMode)
	}
	s, err := uc.repo.UpsertSettings(ctx, &entity.CuttingSettings{
		DefaultConsumptionMode: mode,
		DefaultMinRemnantMM:    dto.DefaultMinRemnantMM,
		DefaultWarehouseID:     dto.DefaultWarehouseID,
	})
	if err != nil {
		return nil, err
	}
	return toSettingsResponse(s), nil
}

// ─── remnants listing ─────────────────────────────────────────────────────────

func (uc *CuttingPlanUseCase) ListRemnants(ctx context.Context, itemCode int64, onlyAvailable bool) ([]*response.StockRemnantResponse, error) {
	rems, err := uc.repo.ListRemnantsByItem(ctx, itemCode, onlyAvailable)
	if err != nil {
		return nil, err
	}
	out := make([]*response.StockRemnantResponse, 0, len(rems))
	for _, r := range rems {
		out = append(out, toRemnantResponse(r))
	}
	return out, nil
}
