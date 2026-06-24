// Package cutting_plan_uc orchestrates the Plano de Corte: building a plan from
// demand and heterogeneous stock, running the pure optimiser, and persisting the
// resulting cutting patterns with their shop-floor metrics.
package cutting_plan_uc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	machinerepo "github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// remnantPriority is opened before fullBarPriority, so reusable remnants are
// consumed ahead of full bars.
const (
	remnantPriority = 0
	fullBarPriority = 10
)

type CuttingPlanUseCase struct {
	repo  repository.CuttingPlanRepository
	stock stockrepo.StockRepository
	items itemrepo.ItemRepository
	// trueShape optionally overrides the native bounding-box true-shape provider
	// with an external nesting engine (e.g. DeepNest/ProNest). Nil = use the
	// registered native provider.
	trueShape service.CuttingOptimizer
	// machines optionally enables booking a plan onto a machine schedule.
	machines machinerepo.MachineRepository
}

func NewCuttingPlanUseCase(repo repository.CuttingPlanRepository, stock stockrepo.StockRepository, items itemrepo.ItemRepository) *CuttingPlanUseCase {
	return &CuttingPlanUseCase{repo: repo, stock: stock, items: items}
}

// WithTrueShapeProvider injects an external true-shape nesting engine, used for
// TRUE_SHAPE_2D plans in place of the native bounding-box provider.
func (uc *CuttingPlanUseCase) WithTrueShapeProvider(p service.CuttingOptimizer) *CuttingPlanUseCase {
	uc.trueShape = p
	return uc
}

// WithMachineRepo enables scheduling a plan onto a machine's calendar.
func (uc *CuttingPlanUseCase) WithMachineRepo(m machinerepo.MachineRepository) *CuttingPlanUseCase {
	uc.machines = m
	return uc
}

// resolveOptimizer picks the optimiser for a cut type: the injected external
// true-shape engine when present, otherwise the registered native strategy.
func (uc *CuttingPlanUseCase) resolveOptimizer(cutType entity.CutType) (service.CuttingOptimizer, error) {
	if cutType == entity.CutTypeTrueShape2D && uc.trueShape != nil {
		return uc.trueShape, nil
	}
	return service.Optimizer(cutType)
}

// Create builds a draft plan, optionally seeding it with parts and stock pieces.
func (uc *CuttingPlanUseCase) Create(ctx context.Context, dto request.CreateCuttingPlanDTO) (*response.CuttingPlanResponse, error) {
	if dto.MaterialItemCode <= 0 {
		return nil, fmt.Errorf("material_item_code is required")
	}
	code, err := uc.repo.NextPlanCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating plan code: %w", err)
	}
	plan, err := entity.NewCuttingPlan(
		code, dto.Description,
		entity.CutType(dto.CutType), entity.PlanSource(dto.Source),
		dto.MaterialItemCode, dto.MachineCode,
		dto.KerfMM, dto.TrimMM, dto.MinRemnantMM, dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	switch plan.CutType {
	case entity.CutTypeLinear1D, entity.CutTypeGuillotine2D, entity.CutTypeTrueShape2D:
	default:
		return nil, fmt.Errorf("cut_type %q not supported", plan.CutType)
	}
	plan.WarehouseID = dto.WarehouseID
	plan.ProductionOrderCode = dto.ProductionOrderCode
	plan.IncludeRemnants = dto.IncludeRemnants
	plan.UoMFactor = dto.UoMFactor

	// Resolve the material's stock UoM: explicit on the DTO, else snapshot it from
	// the item registry so the baixa later converts the cut length correctly.
	if dto.StockUoM != "" {
		plan.StockUoM = types.TypeUnitOfMeasurementItem(dto.StockUoM)
	} else if uc.items != nil {
		if ic, icErr := valueobject.NewItemCode(dto.MaterialItemCode); icErr == nil {
			if item, iErr := uc.items.FindItemByCode(ctx, ic); iErr == nil && item != nil {
				plan.StockUoM = item.Warehouse.UnitOfMeasurement
			}
		}
	}
	if plan.StockUoM == "" {
		plan.StockUoM = types.UN
	}
	if dto.LotConsumptionMode != "" {
		mode := entity.ConsumptionMode(dto.LotConsumptionMode)
		if mode != entity.ConsumptionAutomatic && mode != entity.ConsumptionManual {
			return nil, fmt.Errorf("invalid lot_consumption_mode %q (AUTOMATIC|MANUAL)", dto.LotConsumptionMode)
		}
		plan.LotConsumptionMode = &mode
	}
	created, err := uc.repo.CreatePlan(ctx, plan)
	if err != nil {
		return nil, err
	}

	for _, p := range dto.Parts {
		part, err := buildPart(created.ID, p)
		if err != nil {
			return nil, err
		}
		if _, err := uc.repo.AddPart(ctx, part); err != nil {
			return nil, err
		}
	}
	for _, s := range dto.StockPieces {
		sp, err := buildStock(created.ID, s)
		if err != nil {
			return nil, err
		}
		if _, err := uc.repo.AddStockPiece(ctx, sp); err != nil {
			return nil, err
		}
	}
	return toPlanResponse(created), nil
}

// buildPart builds a 1D, 2D or true-shape part from one input: a polygon marks a
// true-shape part, width/height a rectangular (2D) part, otherwise it is 1D.
func buildPart(planID int64, p request.CuttingPlanPartInput) (*entity.CuttingPlanPart, error) {
	var (
		part *entity.CuttingPlanPart
		err  error
	)
	switch {
	case len(p.Geometry) >= 3:
		poly := make([]service.Point, len(p.Geometry))
		for i, pt := range p.Geometry {
			poly[i] = service.Point{X: pt.X, Y: pt.Y}
		}
		w, h := service.PolygonBBox(poly)
		raw, merr := json.Marshal(poly)
		if merr != nil {
			return nil, fmt.Errorf("encoding geometry: %w", merr)
		}
		part, err = entity.NewPartTrueShape(planID, p.ItemCode, p.Label, string(raw), w, h, p.AllowRotation, p.Quantity, p.SourceRef)
	case p.WidthMM > 0 || p.HeightMM > 0:
		part, err = entity.NewPart2D(planID, p.ItemCode, p.Label, p.WidthMM, p.HeightMM, entity.Grain(p.Grain), p.AllowRotation, p.Quantity, p.SourceRef)
	default:
		part, err = entity.NewPart(planID, p.ItemCode, p.Label, p.LengthMM, p.Quantity, p.SourceRef)
	}
	if err != nil {
		return nil, err
	}
	// Edge banding applies to rectangular faces (2D / true-shape bbox).
	part.EdgeTop, part.EdgeBottom = p.EdgeTop, p.EdgeBottom
	part.EdgeLeft, part.EdgeRight = p.EdgeLeft, p.EdgeRight
	part.BandItemCode, part.BandCostPerM = p.BandItemCode, p.BandCostPerM
	return part, nil
}

// buildStock builds a 1D or 2D stock piece from one input.
func buildStock(planID int64, s request.CuttingStockPieceInput) (*entity.CuttingStockPiece, error) {
	if s.WidthMM > 0 || s.HeightMM > 0 {
		return entity.NewStockPiece2D(planID, s.WidthMM, s.HeightMM, s.Quantity, s.Lot, s.IsRemnant)
	}
	return entity.NewStockPiece(planID, s.LengthMM, s.Quantity, s.Lot, s.IsRemnant)
}

func (uc *CuttingPlanUseCase) AddPart(ctx context.Context, dto request.AddCuttingPlanPartDTO) (*response.CuttingPlanPartResponse, error) {
	part, err := buildPart(dto.PlanID, dto.CuttingPlanPartInput)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.AddPart(ctx, part)
	if err != nil {
		return nil, err
	}
	return toPartResponse(created), nil
}

func (uc *CuttingPlanUseCase) RemovePart(ctx context.Context, id int64) error {
	return uc.repo.RemovePart(ctx, id)
}

func (uc *CuttingPlanUseCase) AddStock(ctx context.Context, dto request.AddCuttingStockPieceDTO) (*response.CuttingStockPieceResponse, error) {
	sp, err := buildStock(dto.PlanID, dto.CuttingStockPieceInput)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.AddStockPiece(ctx, sp)
	if err != nil {
		return nil, err
	}
	return toStockResponse(created), nil
}

func (uc *CuttingPlanUseCase) RemoveStock(ctx context.Context, id int64) error {
	return uc.repo.RemoveStockPiece(ctx, id)
}

func (uc *CuttingPlanUseCase) List(ctx context.Context, onlyOpen bool) ([]*response.CuttingPlanResponse, error) {
	plans, err := uc.repo.ListPlans(ctx, onlyOpen)
	if err != nil {
		return nil, err
	}
	out := make([]*response.CuttingPlanResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, toPlanResponse(p))
	}
	return out, nil
}

func (uc *CuttingPlanUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.DeletePlan(ctx, id)
}

// GetDetail assembles the full plan: header, demand, stock and stored patterns.
func (uc *CuttingPlanUseCase) GetDetail(ctx context.Context, planID int64) (*response.CuttingPlanDetailResponse, error) {
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	parts, err := uc.repo.ListParts(ctx, planID)
	if err != nil {
		return nil, err
	}
	stock, err := uc.repo.ListStockPieces(ctx, planID)
	if err != nil {
		return nil, err
	}
	patterns, err := uc.repo.ListPatterns(ctx, planID)
	if err != nil {
		return nil, err
	}
	return buildDetail(plan, parts, stock, patterns, nil), nil
}

// Optimize runs the cutting optimiser for the plan, persists the resulting
// patterns, updates the plan metrics and returns the full result (including any
// pieces that could not be placed).
func (uc *CuttingPlanUseCase) Optimize(ctx context.Context, planID int64) (*response.CuttingPlanDetailResponse, error) {
	plan, err := uc.repo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.Status == entity.PlanStatusReleased {
		return nil, fmt.Errorf("plan already firmed; cannot re-optimize")
	}
	// When configured, seed the plan with the material's available remnants so the
	// optimiser consumes offcuts before opening full bars. Re-seeding is idempotent.
	if plan.IncludeRemnants && plan.WarehouseID != nil {
		if err := uc.seedRemnants(ctx, plan); err != nil {
			return nil, fmt.Errorf("seeding remnants: %w", err)
		}
	}
	parts, err := uc.repo.ListParts(ctx, planID)
	if err != nil {
		return nil, err
	}
	stock, err := uc.repo.ListStockPieces(ctx, planID)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("plan has no parts to cut")
	}
	if len(stock) == 0 {
		return nil, fmt.Errorf("plan has no stock to cut from")
	}

	optimizer, err := uc.resolveOptimizer(plan.CutType)
	if err != nil {
		return nil, fmt.Errorf("cut type %q: %w", plan.CutType, err)
	}

	demand := make([]service.DemandPiece, 0, len(parts))
	for _, p := range parts {
		d := service.DemandPiece{
			PartID: p.ID, Label: p.Label, Length: p.LengthMM, Qty: p.Quantity,
			Width: p.WidthMM, Height: p.HeightMM, Grain: service.Grain(p.Grain), AllowRotation: p.AllowRotation,
		}
		if p.Geometry != nil && *p.Geometry != "" {
			var poly []service.Point
			if err := json.Unmarshal([]byte(*p.Geometry), &poly); err == nil {
				d.Polygon = poly
			}
		}
		demand = append(demand, d)
	}
	stockPieces := make([]service.StockPiece, 0, len(stock))
	for _, s := range stock {
		prio := fullBarPriority
		if s.IsRemnant {
			prio = remnantPriority
		}
		stockPieces = append(stockPieces, service.StockPiece{
			StockID: s.ID, Length: s.LengthMM, Qty: s.Quantity, IsRemnant: s.IsRemnant, Priority: prio,
			Width: s.WidthMM, Height: s.HeightMM,
		})
	}

	sol, err := optimizer.Optimize(demand, stockPieces, service.CutParams{
		Kerf: plan.KerfMM, Trim: plan.TrimMM, MinRemnant: plan.MinRemnantMM,
	})
	if err != nil {
		return nil, fmt.Errorf("optimizing: %w", err)
	}

	patterns := solutionToPatterns(plan, sol)
	if err := uc.repo.ReplacePatterns(ctx, planID, patterns); err != nil {
		return nil, err
	}

	applyMetrics(plan, sol)
	if err := uc.repo.UpdatePlanResult(ctx, plan); err != nil {
		return nil, err
	}

	return buildDetail(plan, parts, stock, patterns, sol.Unplaced), nil
}

// solutionToPatterns maps the pure solver output into persistable patterns,
// numbering sequences and computing per-pattern utilisation.
func solutionToPatterns(plan *entity.CuttingPlan, sol *service.Solution) []*entity.CuttingPattern {
	patterns := make([]*entity.CuttingPattern, 0, len(sol.Patterns))
	for i, sp := range sol.Patterns {
		sheetArea := sp.StockWidth * sp.StockHeight
		util := 0.0
		switch {
		case sheetArea > 0:
			util = sp.UsedArea / sheetArea * 100 // 2D
		case sp.StockLength > 0:
			util = sp.UsedLength / sp.StockLength * 100 // 1D
		}
		pat := &entity.CuttingPattern{
			PlanID:          plan.ID,
			Sequence:        i + 1,
			StockLengthMM:   sp.StockLength,
			RepeatCount:     sp.Repeat,
			UsedMM:          sp.UsedLength,
			KerfLossMM:      sp.KerfLoss,
			RemnantMM:       sp.Remnant,
			UtilizationPct:  util,
			IsRemnant:       sp.IsRemnant,
			StockWidthMM:    sp.StockWidth,
			StockHeightMM:   sp.StockHeight,
			UsedAreaMM2:     sp.UsedArea,
			RemnantAreaMM2:  sp.RemnantArea,
			RemnantWidthMM:  sp.RemnantWidth,
			RemnantHeightMM: sp.RemnantHeight,
		}
		for j, pl := range sp.Placements {
			partID := pl.PartID
			var partRef *int64
			if partID > 0 {
				partRef = &partID
			}
			pat.Placements = append(pat.Placements, &entity.PatternPlacement{
				Sequence:    j + 1,
				PartID:      partRef,
				Label:       pl.Label,
				LengthMM:    pl.Length,
				OffsetMM:    pl.Offset,
				PosXMM:      pl.X,
				PosYMM:      pl.Y,
				WidthMM:     pl.W,
				HeightMM:    pl.H,
				Rotated:     pl.Rotated,
				RotationDeg: pl.RotationDeg,
			})
		}
		patterns = append(patterns, pat)
	}
	return patterns
}

// applyMetrics rolls the solution metrics into the plan. Reusable remnants
// (leftover >= MinRemnantMM) are NOT counted as scrap, since phase 2 returns them
// to stock — only true offcuts and kerf are scrap.
func applyMetrics(plan *entity.CuttingPlan, sol *service.Solution) {
	var reusable float64
	for _, p := range sol.Patterns {
		if plan.MinRemnantMM <= 0 {
			continue
		}
		if p.StockWidth > 0 { // 2D: reusable rectangle area when both sides clear the minimum
			if p.RemnantWidth >= plan.MinRemnantMM && p.RemnantHeight >= plan.MinRemnantMM {
				reusable += p.RemnantWidth * p.RemnantHeight * float64(p.Repeat)
			}
		} else if p.Remnant >= plan.MinRemnantMM { // 1D
			reusable += p.Remnant * float64(p.Repeat)
		}
	}
	scrap := sol.TotalStock - sol.TotalDemand - reusable
	if scrap < 0 {
		scrap = 0
	}
	scrapPct := 0.0
	if sol.TotalStock > 0 {
		scrapPct = scrap / sol.TotalStock * 100
	}

	plan.Status = entity.PlanStatusOptimized
	plan.UtilizationPct = sol.Utilization * 100
	plan.ScrapPct = scrapPct
	plan.StockUsedCount = sol.StockUsed
	plan.CutCount = sol.CutCount
	plan.TotalDemand = sol.TotalDemand
	plan.TotalStock = sol.TotalStock
}
