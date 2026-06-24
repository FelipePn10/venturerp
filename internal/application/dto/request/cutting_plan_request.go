package request

import "github.com/google/uuid"

// CreateCuttingPlanDTO creates a cutting plan. Parts and StockPieces are
// optional: a plan can be created empty and filled incrementally, or in one
// call. CutType defaults to LINEAR_1D and Source to MANUAL.
type CreateCuttingPlanDTO struct {
	Description      *string `json:"description,omitempty"`
	CutType          string  `json:"cut_type,omitempty"`
	Source           string  `json:"source,omitempty"`
	MaterialItemCode int64   `json:"material_item_code"`
	MachineCode      *int64  `json:"machine_code,omitempty"`
	KerfMM           float64 `json:"kerf_mm"`
	TrimMM           float64 `json:"trim_mm"`
	MinRemnantMM     float64 `json:"min_remnant_mm"`
	// Phase 2 — release/firmar.
	WarehouseID         *int64                   `json:"warehouse_id,omitempty"`
	ProductionOrderCode *int64                   `json:"production_order_code,omitempty"`
	LotConsumptionMode  string                   `json:"lot_consumption_mode,omitempty"` // AUTOMATIC | MANUAL (vazio = padrão da empresa)
	IncludeRemnants     bool                     `json:"include_remnants,omitempty"`
	StockUoM            string                   `json:"stock_uom,omitempty"`  // UoM de estoque; vazio = busca do item
	UoMFactor           float64                  `json:"uom_factor,omitempty"` // qtd de estoque por metro linear (KG/M2/M3/TON)
	Parts               []CuttingPlanPartInput   `json:"parts,omitempty"`
	StockPieces         []CuttingStockPieceInput `json:"stock_pieces,omitempty"`
	CreatedBy           uuid.UUID                `json:"created_by"`
}

// GenerateCuttingFromOrdersDTO builds cutting plans automatically from production
// and/or planned orders, aggregating parts of the same raw material into one plan.
type GenerateCuttingFromOrdersDTO struct {
	ProductionOrderCodes []int64   `json:"production_order_codes,omitempty"`
	PlannedOrderCodes    []int64   `json:"planned_order_codes,omitempty"`
	KerfMM               float64   `json:"kerf_mm"`
	TrimMM               float64   `json:"trim_mm"`
	MinRemnantMM         float64   `json:"min_remnant_mm"`
	WarehouseID          *int64    `json:"warehouse_id,omitempty"`
	IncludeRemnants      bool      `json:"include_remnants,omitempty"`
	AllowRotation        bool      `json:"allow_rotation,omitempty"` // for 2D parts
	CreatedBy            uuid.UUID `json:"created_by"`
}

// CuttingSettingsDTO sets the company-level defaults for cutting plans.
type CuttingSettingsDTO struct {
	DefaultConsumptionMode string  `json:"default_consumption_mode"` // AUTOMATIC | MANUAL
	DefaultMinRemnantMM    float64 `json:"default_min_remnant_mm"`
	DefaultWarehouseID     *int64  `json:"default_warehouse_id,omitempty"`
}

// PointInput is one (x,y) vertex (mm) of a true-shape part outline.
type PointInput struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// CuttingPlanPartInput is one demanded piece. For 1D set LengthMM; for 2D set
// WidthMM/HeightMM; for true-shape provide Geometry (a closed polygon outline).
type CuttingPlanPartInput struct {
	ItemCode      *int64       `json:"item_code,omitempty"`
	Label         string       `json:"label"`
	LengthMM      float64      `json:"length_mm,omitempty"`
	WidthMM       float64      `json:"width_mm,omitempty"`
	HeightMM      float64      `json:"height_mm,omitempty"`
	Grain         string       `json:"grain,omitempty"` // NONE | LENGTH | WIDTH
	AllowRotation bool         `json:"allow_rotation,omitempty"`
	Geometry      []PointInput `json:"geometry,omitempty"` // true-shape outline
	// Edge banding (fita de borda — moveleiro), for 2D parts.
	EdgeTop      bool    `json:"edge_top,omitempty"`
	EdgeBottom   bool    `json:"edge_bottom,omitempty"`
	EdgeLeft     bool    `json:"edge_left,omitempty"`
	EdgeRight    bool    `json:"edge_right,omitempty"`
	BandItemCode *int64  `json:"band_item_code,omitempty"`
	BandCostPerM float64 `json:"band_cost_per_m,omitempty"`
	Quantity     int     `json:"quantity"`
	SourceRef    *string `json:"source_ref,omitempty"`
}

// CuttingStockPieceInput is one available raw piece to cut from. For 1D set
// LengthMM; for 2D set WidthMM/HeightMM.
type CuttingStockPieceInput struct {
	LengthMM  float64 `json:"length_mm,omitempty"`
	WidthMM   float64 `json:"width_mm,omitempty"`
	HeightMM  float64 `json:"height_mm,omitempty"`
	Quantity  int     `json:"quantity"`
	Lot       *string `json:"lot,omitempty"`
	IsRemnant bool    `json:"is_remnant,omitempty"`
}

// AddCuttingPlanPartDTO adds a part to an existing plan.
type AddCuttingPlanPartDTO struct {
	PlanID int64 `json:"-"`
	CuttingPlanPartInput
}

// AddCuttingStockPieceDTO adds a stock piece to an existing plan.
type AddCuttingStockPieceDTO struct {
	PlanID int64 `json:"-"`
	CuttingStockPieceInput
}
