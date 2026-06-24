// Package service holds the cutting-plan optimisation logic. The optimiser is a
// pure domain service: given demand, available stock and cutting parameters it
// returns a nesting solution, with no knowledge of persistence or HTTP. That
// keeps the algorithm independently unit-testable and lets each cut type
// (1D linear, 2D guillotine, true-shape) plug in behind one interface.
package service

import (
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// Grain is the material grain direction a 2D part must respect. When set, the
// part keeps its given orientation (no rotation) so a visible wood/laminate grain
// stays aligned — the practical furniture constraint.
type Grain string

const (
	GrainNone   Grain = "NONE"
	GrainLength Grain = "LENGTH" // grain runs along the part's height (length) axis
	GrainWidth  Grain = "WIDTH"
)

// Point is a 2D vertex (mm) of an irregular part outline.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DemandPiece is one required cut and how many of it are needed. For 1D only
// Length is used; for 2D guillotine Width/Height/Grain/AllowRotation drive the
// nesting (Length is ignored); for true-shape Polygon carries the outline (its
// bounding box falls back to Width/Height).
type DemandPiece struct {
	PartID int64 // correlation id back to the persisted part (0 when synthetic)
	Label  string
	Length float64
	Qty    int

	// 2D fields.
	Width         float64
	Height        float64
	Grain         Grain
	AllowRotation bool

	// True-shape outline (optional).
	Polygon []Point
}

// StockPiece is an available raw piece to cut from. Qty groups identical pieces.
// Priority orders which stock is opened first (lower = first), so reusable
// remnants can be consumed ahead of full sheets/bars. For 1D only Length is used;
// for 2D Width/Height define the sheet.
type StockPiece struct {
	StockID   int64
	Length    float64
	Qty       int
	IsRemnant bool
	Priority  int

	// 2D fields.
	Width  float64
	Height float64
}

// CutParams are the blade/material parameters applied during nesting.
type CutParams struct {
	Kerf       float64 // material lost per cut
	Trim       float64 // length removed from each stock piece before any cut
	MinRemnant float64 // leftover >= this counts as a reusable remnant, not scrap
}

// Placement is one part positioned on a stock piece. 1D uses Offset/Length; 2D
// uses X/Y/W/H and Rotated.
type Placement struct {
	PartID int64
	Label  string
	Length float64
	Offset float64 // start position along the (trimmed) stock piece (1D)

	// 2D fields.
	X       float64
	Y       float64
	W       float64
	H       float64
	Rotated bool

	// True-shape: arbitrary rotation angle in degrees (0/90 for the bbox provider).
	RotationDeg float64
}

// Pattern is a single cutting layout for one stock piece, repeated Repeat times.
// 1D fills StockLength/UsedLength/Remnant; 2D fills StockWidth/StockHeight/
// UsedArea/RemnantArea.
type Pattern struct {
	StockID     int64
	StockLength float64
	IsRemnant   bool
	Repeat      int
	Placements  []Placement
	UsedLength  float64 // sum of part lengths (1D)
	KerfLoss    float64 // kerf consumed across the cuts
	Remnant     float64 // leftover after the last cut (1D, reusable if >= MinRemnant)

	// 2D fields.
	StockWidth    float64
	StockHeight   float64
	UsedArea      float64 // sum of placed part areas
	RemnantArea   float64 // total free area left on the sheet
	RemnantWidth  float64 // largest reusable leftover rectangle (width)
	RemnantHeight float64 // largest reusable leftover rectangle (height)
}

// Solution is the full result of an optimisation run.
type Solution struct {
	Patterns    []Pattern
	Unplaced    []DemandPiece // pieces no stock could hold
	TotalDemand float64       // sum of all placed part lengths
	TotalStock  float64       // sum of consumed stock lengths
	Utilization float64       // TotalDemand / TotalStock (0..1)
	StockUsed   int           // number of stock pieces consumed
	CutCount    int           // total number of cuts across all patterns
}

// CuttingOptimizer is the strategy implemented per cut type. New cut types
// (2D guillotine, true-shape via an external provider) implement this same
// contract and register themselves, so callers stay agnostic.
type CuttingOptimizer interface {
	Type() entity.CutType
	Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error)
}

// registry maps a cut type to its optimiser. It is populated by each optimiser's
// init() so adding a strategy is a single self-contained file.
var registry = map[entity.CutType]CuttingOptimizer{}

// register wires an optimiser into the registry. Called from optimiser init().
func register(o CuttingOptimizer) { registry[o.Type()] = o }

// ErrNoOptimizer is returned when no strategy is registered for a cut type —
// e.g. true-shape before its external provider is configured.
var ErrNoOptimizer = errors.New("no optimizer registered for cut type")

// Optimizer returns the registered optimiser for a cut type.
func Optimizer(t entity.CutType) (CuttingOptimizer, error) {
	o, ok := registry[t]
	if !ok {
		return nil, ErrNoOptimizer
	}
	return o, nil
}
