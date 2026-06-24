package cutting_plan

import (
	"context"
	"fmt"
	"time"

	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// CuttingPlanRepositorySQLC persists the cutting-plan aggregate via sqlc. The
// pool is kept for ReplacePatterns, which must delete and re-insert the derived
// patterns/placements atomically in one transaction.
type CuttingPlanRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.CuttingPlanRepository {
	return &CuttingPlanRepositorySQLC{q: q, pool: pool}
}

// ─── plans ────────────────────────────────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) NextPlanCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextCuttingPlanCode(ctx)
	return int64(v), err
}

func (r *CuttingPlanRepositorySQLC) CreatePlan(ctx context.Context, p *entity.CuttingPlan) (*entity.CuttingPlan, error) {
	row, err := r.q.CreateCuttingPlan(ctx, sqlc.CreateCuttingPlanParams{
		Code:                p.Code,
		Description:         pgutil.ToPgTextFromPtr(p.Description),
		CutType:             string(p.CutType),
		Source:              string(p.Source),
		MaterialItemCode:    p.MaterialItemCode,
		MachineCode:         p.MachineCode,
		KerfMm:              pgutil.ToPgNumericFromFloat64(p.KerfMM),
		TrimMm:              pgutil.ToPgNumericFromFloat64(p.TrimMM),
		MinRemnantMm:        pgutil.ToPgNumericFromFloat64(p.MinRemnantMM),
		WarehouseID:         p.WarehouseID,
		ProductionOrderCode: p.ProductionOrderCode,
		LotConsumptionMode:  consumptionModeToPgText(p.LotConsumptionMode),
		IncludeRemnants:     p.IncludeRemnants,
		StockUom:            string(p.StockUoM),
		UomFactor:           pgutil.ToPgNumericFromFloat64(p.UoMFactor),
		CreatedBy:           pgutil.ToPgUUID(p.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating cutting plan: %w", err)
	}
	return planRowToEntity(row), nil
}

func (r *CuttingPlanRepositorySQLC) GetPlanByID(ctx context.Context, id int64) (*entity.CuttingPlan, error) {
	row, err := r.q.GetCuttingPlanByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching cutting plan %d: %w", id, err)
	}
	return planRowToEntity(row), nil
}

func (r *CuttingPlanRepositorySQLC) ListPlans(ctx context.Context, onlyOpen bool) ([]*entity.CuttingPlan, error) {
	rows, err := r.q.ListCuttingPlans(ctx, onlyOpen)
	if err != nil {
		return nil, fmt.Errorf("listing cutting plans: %w", err)
	}
	out := make([]*entity.CuttingPlan, 0, len(rows))
	for _, row := range rows {
		out = append(out, planRowToEntity(row))
	}
	return out, nil
}

func (r *CuttingPlanRepositorySQLC) UpdatePlanResult(ctx context.Context, p *entity.CuttingPlan) error {
	return r.q.UpdateCuttingPlanResult(ctx, sqlc.UpdateCuttingPlanResultParams{
		ID:             p.ID,
		Status:         string(p.Status),
		UtilizationPct: pgutil.ToPgNumericFromFloat64(p.UtilizationPct),
		ScrapPct:       pgutil.ToPgNumericFromFloat64(p.ScrapPct),
		StockUsedCount: int32(p.StockUsedCount),
		CutCount:       int32(p.CutCount),
		TotalDemand:    pgutil.ToPgNumericFromFloat64(p.TotalDemand),
		TotalStock:     pgutil.ToPgNumericFromFloat64(p.TotalStock),
	})
}

func (r *CuttingPlanRepositorySQLC) DeletePlan(ctx context.Context, id int64) error {
	return r.q.DeleteCuttingPlan(ctx, id)
}

// ─── parts ────────────────────────────────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) AddPart(ctx context.Context, part *entity.CuttingPlanPart) (*entity.CuttingPlanPart, error) {
	row, err := r.q.AddCuttingPlanPart(ctx, sqlc.AddCuttingPlanPartParams{
		PlanID:        part.PlanID,
		ItemCode:      part.ItemCode,
		Label:         part.Label,
		LengthMm:      pgutil.ToPgNumericFromFloat64(part.LengthMM),
		Quantity:      int32(part.Quantity),
		SourceRef:     pgutil.ToPgTextFromPtr(part.SourceRef),
		WidthMm:       pgutil.ToPgNumericFromFloat64(part.WidthMM),
		HeightMm:      pgutil.ToPgNumericFromFloat64(part.HeightMM),
		Grain:         grainStr(part.Grain),
		AllowRotation: part.AllowRotation,
		Geometry:      pgutil.ToPgTextFromPtr(part.Geometry),
		EdgeTop:       part.EdgeTop,
		EdgeBottom:    part.EdgeBottom,
		EdgeLeft:      part.EdgeLeft,
		EdgeRight:     part.EdgeRight,
		BandItemCode:  part.BandItemCode,
		BandCostPerM:  pgutil.ToPgNumericFromFloat64(part.BandCostPerM),
	})
	if err != nil {
		return nil, fmt.Errorf("adding part: %w", err)
	}
	return partRowToEntity(row), nil
}

func (r *CuttingPlanRepositorySQLC) ListParts(ctx context.Context, planID int64) ([]*entity.CuttingPlanPart, error) {
	rows, err := r.q.ListCuttingPlanParts(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing parts: %w", err)
	}
	out := make([]*entity.CuttingPlanPart, 0, len(rows))
	for _, row := range rows {
		out = append(out, partRowToEntity(row))
	}
	return out, nil
}

func (r *CuttingPlanRepositorySQLC) RemovePart(ctx context.Context, id int64) error {
	return r.q.RemoveCuttingPlanPart(ctx, id)
}

// ─── stock pieces ─────────────────────────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) AddStockPiece(ctx context.Context, s *entity.CuttingStockPiece) (*entity.CuttingStockPiece, error) {
	row, err := r.q.AddCuttingStockPiece(ctx, sqlc.AddCuttingStockPieceParams{
		PlanID:     s.PlanID,
		LengthMm:   pgutil.ToPgNumericFromFloat64(s.LengthMM),
		Quantity:   int32(s.Quantity),
		Lot:        pgutil.ToPgTextFromPtr(s.Lot),
		IsRemnant:  s.IsRemnant,
		RemnantID:  s.RemnantID,
		HeatNumber: pgutil.ToPgTextFromPtr(s.HeatNumber),
		WidthMm:    pgutil.ToPgNumericFromFloat64(s.WidthMM),
		HeightMm:   pgutil.ToPgNumericFromFloat64(s.HeightMM),
	})
	if err != nil {
		return nil, fmt.Errorf("adding stock piece: %w", err)
	}
	return stockRowToEntity(row), nil
}

func (r *CuttingPlanRepositorySQLC) ListStockPieces(ctx context.Context, planID int64) ([]*entity.CuttingStockPiece, error) {
	rows, err := r.q.ListCuttingStockPieces(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing stock pieces: %w", err)
	}
	out := make([]*entity.CuttingStockPiece, 0, len(rows))
	for _, row := range rows {
		out = append(out, stockRowToEntity(row))
	}
	return out, nil
}

func (r *CuttingPlanRepositorySQLC) RemoveStockPiece(ctx context.Context, id int64) error {
	return r.q.RemoveCuttingStockPiece(ctx, id)
}

// ─── patterns (transactional replace) ─────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) ReplacePatterns(ctx context.Context, planID int64, patterns []*entity.CuttingPattern) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	if err := qtx.DeleteCuttingPatternsByPlan(ctx, planID); err != nil {
		return fmt.Errorf("clearing patterns: %w", err)
	}

	for _, pat := range patterns {
		prow, err := qtx.CreateCuttingPattern(ctx, sqlc.CreateCuttingPatternParams{
			PlanID:          planID,
			Sequence:        int32(pat.Sequence),
			StockLengthMm:   pgutil.ToPgNumericFromFloat64(pat.StockLengthMM),
			RepeatCount:     int32(pat.RepeatCount),
			UsedMm:          pgutil.ToPgNumericFromFloat64(pat.UsedMM),
			KerfLossMm:      pgutil.ToPgNumericFromFloat64(pat.KerfLossMM),
			RemnantMm:       pgutil.ToPgNumericFromFloat64(pat.RemnantMM),
			UtilizationPct:  pgutil.ToPgNumericFromFloat64(pat.UtilizationPct),
			IsRemnant:       pat.IsRemnant,
			StockWidthMm:    pgutil.ToPgNumericFromFloat64(pat.StockWidthMM),
			StockHeightMm:   pgutil.ToPgNumericFromFloat64(pat.StockHeightMM),
			UsedAreaMm2:     pgutil.ToPgNumericFromFloat64(pat.UsedAreaMM2),
			RemnantAreaMm2:  pgutil.ToPgNumericFromFloat64(pat.RemnantAreaMM2),
			RemnantWidthMm:  pgutil.ToPgNumericFromFloat64(pat.RemnantWidthMM),
			RemnantHeightMm: pgutil.ToPgNumericFromFloat64(pat.RemnantHeightMM),
		})
		if err != nil {
			return fmt.Errorf("inserting pattern: %w", err)
		}
		for _, pl := range pat.Placements {
			if _, err := qtx.CreateCuttingPatternPlacement(ctx, sqlc.CreateCuttingPatternPlacementParams{
				PatternID:   prow.ID,
				Sequence:    int32(pl.Sequence),
				PartID:      pl.PartID,
				Label:       pl.Label,
				LengthMm:    pgutil.ToPgNumericFromFloat64(pl.LengthMM),
				OffsetMm:    pgutil.ToPgNumericFromFloat64(pl.OffsetMM),
				PosXMm:      pgutil.ToPgNumericFromFloat64(pl.PosXMM),
				PosYMm:      pgutil.ToPgNumericFromFloat64(pl.PosYMM),
				WidthMm:     pgutil.ToPgNumericFromFloat64(pl.WidthMM),
				HeightMm:    pgutil.ToPgNumericFromFloat64(pl.HeightMM),
				Rotated:     pl.Rotated,
				RotationDeg: pgutil.ToPgNumericFromFloat64(pl.RotationDeg),
			}); err != nil {
				return fmt.Errorf("inserting placement: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit patterns: %w", err)
	}
	return nil
}

func (r *CuttingPlanRepositorySQLC) ListPatterns(ctx context.Context, planID int64) ([]*entity.CuttingPattern, error) {
	rows, err := r.q.ListCuttingPatternsByPlan(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing patterns: %w", err)
	}
	out := make([]*entity.CuttingPattern, 0, len(rows))
	for _, row := range rows {
		pat := patternRowToEntity(row)
		plRows, err := r.q.ListCuttingPatternPlacements(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("listing placements: %w", err)
		}
		for _, pr := range plRows {
			pat.Placements = append(pat.Placements, placementRowToEntity(pr))
		}
		out = append(out, pat)
	}
	return out, nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func planRowToEntity(row sqlc.CuttingPlan) *entity.CuttingPlan {
	return &entity.CuttingPlan{
		ID:                  row.ID,
		Code:                row.Code,
		Description:         pgutil.FromPgTextPtr(row.Description),
		CutType:             entity.CutType(row.CutType),
		Source:              entity.PlanSource(row.Source),
		Status:              entity.PlanStatus(row.Status),
		MaterialItemCode:    row.MaterialItemCode,
		MachineCode:         row.MachineCode,
		KerfMM:              pgutil.FromPgNumericToFloat64(row.KerfMm),
		TrimMM:              pgutil.FromPgNumericToFloat64(row.TrimMm),
		MinRemnantMM:        pgutil.FromPgNumericToFloat64(row.MinRemnantMm),
		UtilizationPct:      pgutil.FromPgNumericToFloat64(row.UtilizationPct),
		ScrapPct:            pgutil.FromPgNumericToFloat64(row.ScrapPct),
		StockUsedCount:      int(row.StockUsedCount),
		CutCount:            int(row.CutCount),
		TotalDemand:         pgutil.FromPgNumericToFloat64(row.TotalDemand),
		TotalStock:          pgutil.FromPgNumericToFloat64(row.TotalStock),
		StockUoM:            types.TypeUnitOfMeasurementItem(row.StockUom),
		UoMFactor:           pgutil.FromPgNumericToFloat64(row.UomFactor),
		WarehouseID:         row.WarehouseID,
		ProductionOrderCode: row.ProductionOrderCode,
		LotConsumptionMode:  pgTextToConsumptionMode(row.LotConsumptionMode),
		IncludeRemnants:     row.IncludeRemnants,
		ReleasedAt:          pgTimestamptzToPtr(row.ReleasedAt),
		CreatedAt:           pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:           pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
	}
}

func partRowToEntity(row sqlc.CuttingPlanPart) *entity.CuttingPlanPart {
	return &entity.CuttingPlanPart{
		ID:            row.ID,
		PlanID:        row.PlanID,
		ItemCode:      row.ItemCode,
		Label:         row.Label,
		LengthMM:      pgutil.FromPgNumericToFloat64(row.LengthMm),
		WidthMM:       pgutil.FromPgNumericToFloat64(row.WidthMm),
		HeightMM:      pgutil.FromPgNumericToFloat64(row.HeightMm),
		Grain:         entity.Grain(row.Grain),
		AllowRotation: row.AllowRotation,
		Geometry:      pgutil.FromPgTextPtr(row.Geometry),
		EdgeTop:       row.EdgeTop,
		EdgeBottom:    row.EdgeBottom,
		EdgeLeft:      row.EdgeLeft,
		EdgeRight:     row.EdgeRight,
		BandItemCode:  row.BandItemCode,
		BandCostPerM:  pgutil.FromPgNumericToFloat64(row.BandCostPerM),
		Quantity:      int(row.Quantity),
		SourceRef:     pgutil.FromPgTextPtr(row.SourceRef),
		CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func stockRowToEntity(row sqlc.CuttingStockPiece) *entity.CuttingStockPiece {
	return &entity.CuttingStockPiece{
		ID:         row.ID,
		PlanID:     row.PlanID,
		LengthMM:   pgutil.FromPgNumericToFloat64(row.LengthMm),
		WidthMM:    pgutil.FromPgNumericToFloat64(row.WidthMm),
		HeightMM:   pgutil.FromPgNumericToFloat64(row.HeightMm),
		Quantity:   int(row.Quantity),
		Lot:        pgutil.FromPgTextPtr(row.Lot),
		IsRemnant:  row.IsRemnant,
		RemnantID:  row.RemnantID,
		HeatNumber: pgutil.FromPgTextPtr(row.HeatNumber),
		CreatedAt:  pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func patternRowToEntity(row sqlc.CuttingPattern) *entity.CuttingPattern {
	return &entity.CuttingPattern{
		ID:              row.ID,
		PlanID:          row.PlanID,
		Sequence:        int(row.Sequence),
		StockLengthMM:   pgutil.FromPgNumericToFloat64(row.StockLengthMm),
		RepeatCount:     int(row.RepeatCount),
		UsedMM:          pgutil.FromPgNumericToFloat64(row.UsedMm),
		KerfLossMM:      pgutil.FromPgNumericToFloat64(row.KerfLossMm),
		RemnantMM:       pgutil.FromPgNumericToFloat64(row.RemnantMm),
		UtilizationPct:  pgutil.FromPgNumericToFloat64(row.UtilizationPct),
		IsRemnant:       row.IsRemnant,
		StockWidthMM:    pgutil.FromPgNumericToFloat64(row.StockWidthMm),
		StockHeightMM:   pgutil.FromPgNumericToFloat64(row.StockHeightMm),
		UsedAreaMM2:     pgutil.FromPgNumericToFloat64(row.UsedAreaMm2),
		RemnantAreaMM2:  pgutil.FromPgNumericToFloat64(row.RemnantAreaMm2),
		RemnantWidthMM:  pgutil.FromPgNumericToFloat64(row.RemnantWidthMm),
		RemnantHeightMM: pgutil.FromPgNumericToFloat64(row.RemnantHeightMm),
	}
}

func placementRowToEntity(row sqlc.CuttingPatternPlacement) *entity.PatternPlacement {
	return &entity.PatternPlacement{
		ID:          row.ID,
		PatternID:   row.PatternID,
		Sequence:    int(row.Sequence),
		PartID:      row.PartID,
		Label:       row.Label,
		LengthMM:    pgutil.FromPgNumericToFloat64(row.LengthMm),
		OffsetMM:    pgutil.FromPgNumericToFloat64(row.OffsetMm),
		PosXMM:      pgutil.FromPgNumericToFloat64(row.PosXMm),
		PosYMM:      pgutil.FromPgNumericToFloat64(row.PosYMm),
		WidthMM:     pgutil.FromPgNumericToFloat64(row.WidthMm),
		HeightMM:    pgutil.FromPgNumericToFloat64(row.HeightMm),
		Rotated:     row.Rotated,
		RotationDeg: pgutil.FromPgNumericToFloat64(row.RotationDeg),
	}
}

// ─── phase 2: settings ────────────────────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) GetSettings(ctx context.Context) (*entity.CuttingSettings, error) {
	row, err := r.q.GetCuttingSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching cutting settings: %w", err)
	}
	return settingsRowToEntity(row), nil
}

func (r *CuttingPlanRepositorySQLC) UpsertSettings(ctx context.Context, s *entity.CuttingSettings) (*entity.CuttingSettings, error) {
	row, err := r.q.UpsertCuttingSettings(ctx, sqlc.UpsertCuttingSettingsParams{
		DefaultConsumptionMode: string(s.DefaultConsumptionMode),
		DefaultMinRemnantMm:    pgutil.ToPgNumericFromFloat64(s.DefaultMinRemnantMM),
		DefaultWarehouseID:     s.DefaultWarehouseID,
	})
	if err != nil {
		return nil, fmt.Errorf("saving cutting settings: %w", err)
	}
	return settingsRowToEntity(row), nil
}

// ─── phase 2: remnants ────────────────────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) DeleteRemnantStockPieces(ctx context.Context, planID int64) error {
	return r.q.DeleteRemnantStockPieces(ctx, planID)
}

func (r *CuttingPlanRepositorySQLC) ListAvailableRemnants(ctx context.Context, itemCode, warehouseID int64) ([]*entity.StockRemnant, error) {
	rows, err := r.q.ListAvailableRemnants(ctx, sqlc.ListAvailableRemnantsParams{ItemCode: itemCode, WarehouseID: warehouseID})
	if err != nil {
		return nil, fmt.Errorf("listing available remnants: %w", err)
	}
	out := make([]*entity.StockRemnant, 0, len(rows))
	for _, row := range rows {
		out = append(out, remnantRowToEntity(row))
	}
	return out, nil
}

func (r *CuttingPlanRepositorySQLC) ListRemnantsByItem(ctx context.Context, itemCode int64, onlyAvailable bool) ([]*entity.StockRemnant, error) {
	rows, err := r.q.ListRemnantsByItem(ctx, sqlc.ListRemnantsByItemParams{ItemCode: itemCode, Column2: onlyAvailable})
	if err != nil {
		return nil, fmt.Errorf("listing remnants: %w", err)
	}
	out := make([]*entity.StockRemnant, 0, len(rows))
	for _, row := range rows {
		out = append(out, remnantRowToEntity(row))
	}
	return out, nil
}

func (r *CuttingPlanRepositorySQLC) GetRemnant(ctx context.Context, id int64) (*entity.StockRemnant, error) {
	row, err := r.q.GetStockRemnant(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching remnant %d: %w", id, err)
	}
	return remnantRowToEntity(row), nil
}

// ─── phase 2: FIFO lots + consumptions ────────────────────────────────────────

func (r *CuttingPlanRepositorySQLC) ListAvailableLotsFIFO(ctx context.Context, itemCode, warehouseID int64) ([]*entity.LotAvailability, error) {
	rows, err := r.q.ListAvailableLotsFIFO(ctx, sqlc.ListAvailableLotsFIFOParams{ItemCode: itemCode, WarehouseID: warehouseID})
	if err != nil {
		return nil, fmt.Errorf("listing FIFO lots: %w", err)
	}
	out := make([]*entity.LotAvailability, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.LotAvailability{
			Lot:         row.Lot,
			Quantity:    pgutil.FromPgNumericToFloat64(row.Quantity),
			LastCost:    pgutil.FromPgNumericToFloat64(row.LastCost),
			HeatNumber:  pgutil.FromPgTextPtr(row.HeatNumber),
			Certificate: pgutil.FromPgTextPtr(row.Certificate),
			ReceivedAt:  pgDateToPtr(row.ReceivedAt),
		})
	}
	return out, nil
}

func (r *CuttingPlanRepositorySQLC) ListConsumptions(ctx context.Context, planID int64) ([]*entity.CuttingPlanConsumption, error) {
	rows, err := r.q.ListCuttingPlanConsumptions(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing consumptions: %w", err)
	}
	out := make([]*entity.CuttingPlanConsumption, 0, len(rows))
	for _, row := range rows {
		out = append(out, consumptionRowToEntity(row))
	}
	return out, nil
}

// ─── phase 2: release (transactional cutting-side writes) ─────────────────────

func (r *CuttingPlanRepositorySQLC) CommitRelease(
	ctx context.Context,
	planID int64,
	consumedRemnantIDs []int64,
	newRemnants []*entity.StockRemnant,
	consumptions []*entity.CuttingPlanConsumption,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin release tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	for _, id := range consumedRemnantIDs {
		pid := planID
		if err := qtx.MarkRemnantConsumed(ctx, sqlc.MarkRemnantConsumedParams{ID: id, ConsumedPlanID: &pid}); err != nil {
			return fmt.Errorf("marking remnant %d consumed: %w", id, err)
		}
	}

	for _, rem := range newRemnants {
		if _, err := qtx.CreateStockRemnant(ctx, sqlc.CreateStockRemnantParams{
			ItemCode:     rem.ItemCode,
			WarehouseID:  rem.WarehouseID,
			LengthMm:     pgutil.ToPgNumericFromFloat64(rem.LengthMM),
			Lot:          pgutil.ToPgTextFromPtr(rem.Lot),
			HeatNumber:   pgutil.ToPgTextFromPtr(rem.HeatNumber),
			Certificate:  pgutil.ToPgTextFromPtr(rem.Certificate),
			UnitCost:     pgutil.ToPgNumericFromFloat64(rem.UnitCost),
			OriginPlanID: rem.OriginPlanID,
			CreatedBy:    pgutil.ToPgUUID(rem.CreatedBy),
			WidthMm:      pgutil.ToPgNumericFromFloat64(rem.WidthMM),
			HeightMm:     pgutil.ToPgNumericFromFloat64(rem.HeightMM),
		}); err != nil {
			return fmt.Errorf("creating remnant: %w", err)
		}
	}

	for _, c := range consumptions {
		if _, err := qtx.AddCuttingPlanConsumption(ctx, sqlc.AddCuttingPlanConsumptionParams{
			PlanID:      c.PlanID,
			ItemCode:    c.ItemCode,
			SourceType:  c.SourceType,
			Lot:         pgutil.ToPgTextFromPtr(c.Lot),
			RemnantID:   c.RemnantID,
			Quantity:    pgutil.ToPgNumericFromFloat64(c.Quantity),
			LengthMm:    pgutil.ToPgNumericFromFloat64(c.LengthMM),
			UnitCost:    pgutil.ToPgNumericFromFloat64(c.UnitCost),
			TotalCost:   pgutil.ToPgNumericFromFloat64(c.TotalCost),
			WarehouseID: c.WarehouseID,
			MovementID:  c.MovementID,
		}); err != nil {
			return fmt.Errorf("recording consumption: %w", err)
		}
	}

	if err := qtx.ReleaseCuttingPlan(ctx, planID); err != nil {
		return fmt.Errorf("releasing plan: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit release: %w", err)
	}
	return nil
}

// ─── phase 2 mappers + helpers ────────────────────────────────────────────────

func settingsRowToEntity(row sqlc.CuttingSetting) *entity.CuttingSettings {
	return &entity.CuttingSettings{
		DefaultConsumptionMode: entity.ConsumptionMode(row.DefaultConsumptionMode),
		DefaultMinRemnantMM:    pgutil.FromPgNumericToFloat64(row.DefaultMinRemnantMm),
		DefaultWarehouseID:     row.DefaultWarehouseID,
		UpdatedAt:              pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func remnantRowToEntity(row sqlc.StockRemnant) *entity.StockRemnant {
	return &entity.StockRemnant{
		ID:             row.ID,
		ItemCode:       row.ItemCode,
		WarehouseID:    row.WarehouseID,
		LengthMM:       pgutil.FromPgNumericToFloat64(row.LengthMm),
		WidthMM:        pgutil.FromPgNumericToFloat64(row.WidthMm),
		HeightMM:       pgutil.FromPgNumericToFloat64(row.HeightMm),
		Lot:            pgutil.FromPgTextPtr(row.Lot),
		HeatNumber:     pgutil.FromPgTextPtr(row.HeatNumber),
		Certificate:    pgutil.FromPgTextPtr(row.Certificate),
		Status:         entity.RemnantStatus(row.Status),
		UnitCost:       pgutil.FromPgNumericToFloat64(row.UnitCost),
		OriginPlanID:   row.OriginPlanID,
		ConsumedPlanID: row.ConsumedPlanID,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:      pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:      pgutil.FromPgUUID(row.CreatedBy),
	}
}

func consumptionRowToEntity(row sqlc.CuttingPlanConsumption) *entity.CuttingPlanConsumption {
	return &entity.CuttingPlanConsumption{
		ID:          row.ID,
		PlanID:      row.PlanID,
		ItemCode:    row.ItemCode,
		SourceType:  row.SourceType,
		Lot:         pgutil.FromPgTextPtr(row.Lot),
		RemnantID:   row.RemnantID,
		Quantity:    pgutil.FromPgNumericToFloat64(row.Quantity),
		LengthMM:    pgutil.FromPgNumericToFloat64(row.LengthMm),
		UnitCost:    pgutil.FromPgNumericToFloat64(row.UnitCost),
		TotalCost:   pgutil.FromPgNumericToFloat64(row.TotalCost),
		WarehouseID: row.WarehouseID,
		MovementID:  row.MovementID,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func grainStr(g entity.Grain) string {
	if g == "" {
		return string(entity.GrainNone)
	}
	return string(g)
}

func consumptionModeToPgText(m *entity.ConsumptionMode) pgtype.Text {
	if m == nil || *m == "" {
		return pgtype.Text{}
	}
	s := string(*m)
	return pgutil.ToPgTextFromPtr(&s)
}

func pgTextToConsumptionMode(t pgtype.Text) *entity.ConsumptionMode {
	if !t.Valid || t.String == "" {
		return nil
	}
	m := entity.ConsumptionMode(t.String)
	return &m
}

func pgTimestamptzToPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func pgDateToPtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	v := d.Time
	return &v
}

// ─── phase complements: per-order cost allocation ─────────────────────────────

func (r *CuttingPlanRepositorySQLC) ReplaceOrderCosts(ctx context.Context, planID int64, costs []*entity.CuttingPlanOrderCost) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin order-costs tx: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := r.q.WithTx(tx)

	if err := qtx.DeleteOrderCostsByPlan(ctx, planID); err != nil {
		return fmt.Errorf("clearing order costs: %w", err)
	}
	for _, c := range costs {
		if _, err := qtx.AddCuttingPlanOrderCost(ctx, sqlc.AddCuttingPlanOrderCostParams{
			PlanID:        planID,
			OrderRef:      c.OrderRef,
			DemandMeasure: pgutil.ToPgNumericFromFloat64(c.DemandMeasure),
			AllocatedCost: pgutil.ToPgNumericFromFloat64(c.AllocatedCost),
		}); err != nil {
			return fmt.Errorf("inserting order cost: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit order costs: %w", err)
	}
	return nil
}

func (r *CuttingPlanRepositorySQLC) ListOrderCosts(ctx context.Context, planID int64) ([]*entity.CuttingPlanOrderCost, error) {
	rows, err := r.q.ListCuttingPlanOrderCosts(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("listing order costs: %w", err)
	}
	out := make([]*entity.CuttingPlanOrderCost, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.CuttingPlanOrderCost{
			ID:            row.ID,
			PlanID:        row.PlanID,
			OrderRef:      row.OrderRef,
			DemandMeasure: pgutil.FromPgNumericToFloat64(row.DemandMeasure),
			AllocatedCost: pgutil.FromPgNumericToFloat64(row.AllocatedCost),
			CreatedAt:     pgutil.FromPgTimestamptz(row.CreatedAt),
		})
	}
	return out, nil
}
