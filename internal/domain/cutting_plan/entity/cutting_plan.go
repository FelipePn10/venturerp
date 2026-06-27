// Package entity holds the Plano de Corte (cutting plan) aggregate. Phase 1
// covers one-dimensional (linear) cutting of bars, profiles and tubes — the
// metalworking backbone — where a set of demanded pieces is nested into the
// available stock lengths minimising waste.
//
// Crucially for this shop, there is no "standard bar": purchased stock has its
// own, heterogeneous lengths. The plan therefore carries an explicit list of
// available stock pieces, and the optimiser nests against that exact list.
package entity

import (
	"errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/google/uuid"
)

// CutType discriminates the cutting strategy. Phase 1 implements LINEAR_1D;
// GUILLOTINE_2D and TRUE_SHAPE_2D are reserved for later phases so the schema
// and routing don't need to change when they land.
type CutType string

const (
	CutTypeLinear1D     CutType = "LINEAR_1D"
	CutTypeGuillotine2D CutType = "GUILLOTINE_2D"
	CutTypeTrueShape2D  CutType = "TRUE_SHAPE_2D"
)

// PlanStatus is the lifecycle of a cutting plan.
type PlanStatus string

const (
	PlanStatusDraft      PlanStatus = "RASCUNHO"  // parts/stock being entered
	PlanStatusOptimized  PlanStatus = "OTIMIZADO" // patterns computed, not committed
	PlanStatusReleased   PlanStatus = "FIRMADO"   // committed: stock consumed (phase 2)
	PlanStatusInProgress PlanStatus = "EM_EXECUCAO"
	PlanStatusDone       PlanStatus = "CONCLUIDO"
)

// PlanSource records where the demand came from.
type PlanSource string

const (
	SourceManual          PlanSource = "MANUAL"
	SourceProductionOrder PlanSource = "ORDEM_PRODUCAO"
	SourcePlannedOrder    PlanSource = "ORDEM_PLANEJADA"
)

// CuttingPlan is the aggregate root: a plan that cuts ONE raw-material item
// (e.g. a specific steel bar profile) into many demanded pieces. Different
// materials are different plans — you cannot cut a 2" angle from a 1" bar.
type CuttingPlan struct {
	ID          int64
	Code        int64
	Description *string
	CutType     CutType
	Source      PlanSource
	Status      PlanStatus

	MaterialItemCode int64  // raw material being cut
	MachineCode      *int64 // saw / cut-off machine (optional)

	// Stock unit of measure of the material (snapshot from the item) and the
	// conversion factor used for mass/area/volume UoMs (stock qty per linear
	// metre). Length/piece UoMs convert geometrically and ignore the factor.
	StockUoM  types.TypeUnitOfMeasurementItem
	UoMFactor float64

	// Phase 2 — release/firmar.
	WarehouseID         *int64           // warehouse the baixa is posted against
	ProductionOrderCode *int64           // OP this cut serves (ties the consumption to the OP)
	LotConsumptionMode  *ConsumptionMode // nil = inherit the company default
	IncludeRemnants     bool             // auto-load available remnants into the stock set on optimize
	ReleasedAt          *time.Time       // set when the plan is firmed

	// Cutting parameters snapshot (so a plan is reproducible/auditable).
	KerfMM       float64 // blade/saw kerf consumed per cut
	TrimMM       float64 // length trimmed from each stock piece before cutting
	MinRemnantMM float64 // leftover >= this is a reusable remnant (phase 2)

	// Result metrics, filled by Optimize. Zero until the plan is optimized.
	UtilizationPct float64
	ScrapPct       float64
	StockUsedCount int
	CutCount       int
	TotalDemand    float64 // length (1D) or area (2D/true-shape)
	TotalStock     float64 // length (1D) or area (2D/true-shape)

	Parts       []*CuttingPlanPart
	StockPieces []*CuttingStockPiece
	Patterns    []*CuttingPattern

	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy uuid.UUID
}

// Grain is the material grain direction a 2D part must respect (furniture). When
// set (≠ NONE) the part keeps its orientation so a visible grain stays aligned.
type Grain string

const (
	GrainNone   Grain = "NONE"
	GrainLength Grain = "LENGTH"
	GrainWidth  Grain = "WIDTH"
)

// CuttingPlanPart is one demanded piece and how many are needed. For 1D only
// LengthMM is used; for 2D WidthMM/HeightMM/Grain/AllowRotation drive the nesting.
type CuttingPlanPart struct {
	ID            int64
	PlanID        int64
	ItemCode      *int64  // produced part item, when tied to an OP/BOM
	Label         string  // human label, e.g. "Perna mesa 720mm"
	LengthMM      float64 // required cut length (1D)
	WidthMM       float64 // 2D
	HeightMM      float64 // 2D
	Grain         Grain   // 2D grain constraint
	AllowRotation bool    // 2D: piece may be rotated 90°
	Geometry      *string // true-shape: JSON polygon outline ([{x,y},...]); bbox in Width/Height
	// Edge banding (fita de borda — moveleiro): which sides get banding, the band
	// material and its cost per metre.
	EdgeTop      bool
	EdgeBottom   bool
	EdgeLeft     bool
	EdgeRight    bool
	BandItemCode *int64
	BandCostPerM float64
	Quantity     int
	SourceRef    *string // OP number / planned order reference
	CreatedAt    time.Time
}

// BandingLengthMM returns the total edge-band length (mm) for ALL pieces of this
// part: the banded perimeter (top/bottom = width, left/right = height) × quantity.
func (p *CuttingPlanPart) BandingLengthMM() float64 {
	per := 0.0
	if p.EdgeTop {
		per += p.WidthMM
	}
	if p.EdgeBottom {
		per += p.WidthMM
	}
	if p.EdgeLeft {
		per += p.HeightMM
	}
	if p.EdgeRight {
		per += p.HeightMM
	}
	return per * float64(p.Quantity)
}

// CuttingStockPiece is an available raw piece to cut from. Because stock is
// heterogeneous, each row carries its own length. Quantity groups identical
// lengths. Lot/IsRemnant are populated in phase 2 when stock is drawn from real
// lot balances and reusable remnants.
type CuttingStockPiece struct {
	ID         int64
	PlanID     int64
	LengthMM   float64
	WidthMM    float64 // 2D sheet width
	HeightMM   float64 // 2D sheet height
	Quantity   int
	Lot        *string
	IsRemnant  bool
	RemnantID  *int64  // set when this stock piece IS an inventory remnant (phase 2)
	HeatNumber *string // corrida carimbada, para rastreabilidade
	CreatedAt  time.Time
}

// CuttingPattern is one cutting layout repeated RepeatCount times. Operators cut
// RepeatCount identical bars the same way, so grouping identical layouts keeps
// the shop-floor instruction compact.
type CuttingPattern struct {
	ID             int64
	PlanID         int64
	Sequence       int
	StockLengthMM  float64
	RepeatCount    int     // number of stock pieces cut with this layout
	UsedMM         float64 // sum of part lengths in the layout
	KerfLossMM     float64 // total kerf consumed in the layout
	RemnantMM      float64 // leftover per piece (reusable if >= MinRemnantMM) (1D)
	UtilizationPct float64 // used / stock
	IsRemnant      bool    // true when the stock piece cut was itself a remnant
	// 2D fields.
	StockWidthMM    float64
	StockHeightMM   float64
	UsedAreaMM2     float64
	RemnantAreaMM2  float64
	RemnantWidthMM  float64 // largest reusable leftover rectangle
	RemnantHeightMM float64
	Placements      []*PatternPlacement
}

// PatternPlacement positions one part on the stock piece. 1D uses OffsetMM;
// 2D uses PosXMM/PosYMM/WidthMM/HeightMM/Rotated.
type PatternPlacement struct {
	ID          int64
	PatternID   int64
	Sequence    int
	PartID      *int64
	Label       string
	LengthMM    float64
	OffsetMM    float64 // start position along the stock piece (1D)
	PosXMM      float64 // 2D
	PosYMM      float64 // 2D
	WidthMM     float64 // 2D
	HeightMM    float64 // 2D
	Rotated     bool    // 2D
	RotationDeg float64 // true-shape arbitrary rotation angle

	// Outline is the part's real (rotated) contour relative to the placement's
	// bounding-box origin — points in [0,WidthMM]×[0,HeightMM]. It is transient (not
	// persisted): the cutting-map renderer draws this polygon instead of the bounding
	// rectangle when present, so true-shape maps show the actual shapes. Empty for 1D
	// and rectangular 2D parts.
	Outline [][2]float64
}

// NewCuttingPlan builds a draft plan, validating the invariants a plan must
// satisfy regardless of where the demand came from.
func NewCuttingPlan(
	code int64,
	description *string,
	cutType CutType,
	source PlanSource,
	materialItemCode int64,
	machineCode *int64,
	kerfMM, trimMM, minRemnantMM float64,
	createdBy uuid.UUID,
) (*CuttingPlan, error) {
	if code <= 0 {
		return nil, errors.New("plan code must be positive")
	}
	if materialItemCode <= 0 {
		return nil, errors.New("material_item_code must be positive")
	}
	if cutType == "" {
		cutType = CutTypeLinear1D
	}
	if source == "" {
		source = SourceManual
	}
	if kerfMM < 0 || trimMM < 0 || minRemnantMM < 0 {
		return nil, errors.New("kerf, trim and min_remnant cannot be negative")
	}
	return &CuttingPlan{
		Code:             code,
		Description:      description,
		CutType:          cutType,
		Source:           source,
		Status:           PlanStatusDraft,
		MaterialItemCode: materialItemCode,
		MachineCode:      machineCode,
		KerfMM:           kerfMM,
		TrimMM:           trimMM,
		MinRemnantMM:     minRemnantMM,
		CreatedBy:        createdBy,
	}, nil
}

// NewPart validates and builds a demanded piece.
func NewPart(planID int64, itemCode *int64, label string, lengthMM float64, quantity int, sourceRef *string) (*CuttingPlanPart, error) {
	if lengthMM <= 0 {
		return nil, errors.New("part length must be positive")
	}
	if quantity <= 0 {
		return nil, errors.New("part quantity must be positive")
	}
	return &CuttingPlanPart{
		PlanID:    planID,
		ItemCode:  itemCode,
		Label:     label,
		LengthMM:  lengthMM,
		Quantity:  quantity,
		SourceRef: sourceRef,
	}, nil
}

// NewPart2D validates and builds a rectangular (2D) demanded part.
func NewPart2D(planID int64, itemCode *int64, label string, widthMM, heightMM float64, grain Grain, allowRotation bool, quantity int, sourceRef *string) (*CuttingPlanPart, error) {
	if widthMM <= 0 || heightMM <= 0 {
		return nil, errors.New("part width and height must be positive")
	}
	if quantity <= 0 {
		return nil, errors.New("part quantity must be positive")
	}
	if grain == "" {
		grain = GrainNone
	}
	return &CuttingPlanPart{
		PlanID: planID, ItemCode: itemCode, Label: label,
		WidthMM: widthMM, HeightMM: heightMM, Grain: grain, AllowRotation: allowRotation,
		Quantity: quantity, SourceRef: sourceRef,
	}, nil
}

// NewPartTrueShape validates and builds an irregular (true-shape) part: a polygon
// outline (stored as JSON) with its bounding box in width/height.
func NewPartTrueShape(planID int64, itemCode *int64, label string, geometryJSON string, bboxW, bboxH float64, allowRotation bool, quantity int, sourceRef *string) (*CuttingPlanPart, error) {
	if bboxW <= 0 || bboxH <= 0 {
		return nil, errors.New("true-shape part must have a non-degenerate bounding box")
	}
	if quantity <= 0 {
		return nil, errors.New("part quantity must be positive")
	}
	geo := geometryJSON
	return &CuttingPlanPart{
		PlanID: planID, ItemCode: itemCode, Label: label,
		WidthMM: bboxW, HeightMM: bboxH, Grain: GrainNone, AllowRotation: allowRotation,
		Geometry: &geo, Quantity: quantity, SourceRef: sourceRef,
	}, nil
}

// NewStockPiece2D validates and builds an available rectangular (2D) sheet.
func NewStockPiece2D(planID int64, widthMM, heightMM float64, quantity int, lot *string, isRemnant bool) (*CuttingStockPiece, error) {
	if widthMM <= 0 || heightMM <= 0 {
		return nil, errors.New("stock width and height must be positive")
	}
	if quantity <= 0 {
		return nil, errors.New("stock quantity must be positive")
	}
	return &CuttingStockPiece{
		PlanID: planID, WidthMM: widthMM, HeightMM: heightMM, Quantity: quantity, Lot: lot, IsRemnant: isRemnant,
	}, nil
}

// NewStockPiece validates and builds an available stock piece.
func NewStockPiece(planID int64, lengthMM float64, quantity int, lot *string, isRemnant bool) (*CuttingStockPiece, error) {
	if lengthMM <= 0 {
		return nil, errors.New("stock length must be positive")
	}
	if quantity <= 0 {
		return nil, errors.New("stock quantity must be positive")
	}
	return &CuttingStockPiece{
		PlanID:    planID,
		LengthMM:  lengthMM,
		Quantity:  quantity,
		Lot:       lot,
		IsRemnant: isRemnant,
	}, nil
}
