package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// CuttingPlanRepository persists the cutting-plan aggregate. Patterns are
// replaced wholesale on each optimisation run (they are a derived result), while
// parts and stock pieces are edited incrementally while the plan is a draft.
type CuttingPlanRepository interface {
	NextPlanCode(ctx context.Context) (int64, error)

	CreatePlan(ctx context.Context, p *entity.CuttingPlan) (*entity.CuttingPlan, error)
	GetPlanByID(ctx context.Context, id int64) (*entity.CuttingPlan, error)
	ListPlans(ctx context.Context, onlyOpen bool) ([]*entity.CuttingPlan, error)
	UpdatePlanResult(ctx context.Context, p *entity.CuttingPlan) error // status + metrics
	DeletePlan(ctx context.Context, id int64) error

	AddPart(ctx context.Context, part *entity.CuttingPlanPart) (*entity.CuttingPlanPart, error)
	ListParts(ctx context.Context, planID int64) ([]*entity.CuttingPlanPart, error)
	RemovePart(ctx context.Context, id int64) error

	AddStockPiece(ctx context.Context, s *entity.CuttingStockPiece) (*entity.CuttingStockPiece, error)
	ListStockPieces(ctx context.Context, planID int64) ([]*entity.CuttingStockPiece, error)
	RemoveStockPiece(ctx context.Context, id int64) error

	// ReplacePatterns deletes the plan's existing patterns/placements and inserts
	// the supplied ones in a single transaction (the result of one optimise run).
	ReplacePatterns(ctx context.Context, planID int64, patterns []*entity.CuttingPattern) error
	ListPatterns(ctx context.Context, planID int64) ([]*entity.CuttingPattern, error)

	// ── Phase 2: release / remnants / settings ──────────────────────────────

	// DeleteRemnantStockPieces clears the auto-loaded remnant-backed stock pieces
	// of a plan, so optimise can re-seed the current available remnants idempotently.
	DeleteRemnantStockPieces(ctx context.Context, planID int64) error

	// Settings (company-level default, singleton).
	GetSettings(ctx context.Context) (*entity.CuttingSettings, error)
	UpsertSettings(ctx context.Context, s *entity.CuttingSettings) (*entity.CuttingSettings, error)

	// Reusable remnants inventory.
	ListAvailableRemnants(ctx context.Context, itemCode, warehouseID int64) ([]*entity.StockRemnant, error)
	ListRemnantsByItem(ctx context.Context, itemCode int64, onlyAvailable bool) ([]*entity.StockRemnant, error)
	GetRemnant(ctx context.Context, id int64) (*entity.StockRemnant, error)

	// FIFO lot availability for automatic consumption (oldest received first).
	ListAvailableLotsFIFO(ctx context.Context, itemCode, warehouseID int64) ([]*entity.LotAvailability, error)

	// Consumption audit trail.
	ListConsumptions(ctx context.Context, planID int64) ([]*entity.CuttingPlanConsumption, error)

	// Per-order cost allocation (replaced wholesale on each release).
	ReplaceOrderCosts(ctx context.Context, planID int64, costs []*entity.CuttingPlanOrderCost) error
	ListOrderCosts(ctx context.Context, planID int64) ([]*entity.CuttingPlanOrderCost, error)

	// CommitRelease atomically marks the firming of a plan: marks consumed
	// remnants, inserts the newly generated remnants and consumption records, and
	// flips the plan to FIRMADO — all in one transaction. The stock OUT movements
	// are posted by the use case beforehand (their ids land in the consumptions).
	CommitRelease(
		ctx context.Context,
		planID int64,
		consumedRemnantIDs []int64,
		newRemnants []*entity.StockRemnant,
		consumptions []*entity.CuttingPlanConsumption,
	) error
}
