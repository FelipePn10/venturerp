package response

import "time"

// CuttingPlanResponse is the plan header with its result metrics.
type CuttingPlanResponse struct {
	ID               int64     `json:"id"`
	Code             int64     `json:"code"`
	Description      *string   `json:"description,omitempty"`
	CutType          string    `json:"cut_type"`
	Source           string    `json:"source"`
	Status           string    `json:"status"`
	MaterialItemCode int64     `json:"material_item_code"`
	MachineCode      *int64    `json:"machine_code,omitempty"`
	StockUoM         string    `json:"stock_uom"`
	UoMFactor        float64   `json:"uom_factor"`
	KerfMM           float64   `json:"kerf_mm"`
	TrimMM           float64   `json:"trim_mm"`
	MinRemnantMM     float64   `json:"min_remnant_mm"`
	UtilizationPct   float64   `json:"utilization_pct"`
	ScrapPct         float64   `json:"scrap_pct"`
	StockUsedCount   int       `json:"stock_used_count"`
	CutCount         int       `json:"cut_count"`
	TotalDemand      float64   `json:"total_demand"`
	TotalStock       float64   `json:"total_stock"`
	CreatedAt        time.Time `json:"created_at"`
}

type CuttingPlanPartResponse struct {
	ID              int64   `json:"id"`
	PlanID          int64   `json:"plan_id"`
	ItemCode        *int64  `json:"item_code,omitempty"`
	Label           string  `json:"label"`
	LengthMM        float64 `json:"length_mm"`
	WidthMM         float64 `json:"width_mm,omitempty"`
	HeightMM        float64 `json:"height_mm,omitempty"`
	Grain           string  `json:"grain,omitempty"`
	AllowRotation   bool    `json:"allow_rotation,omitempty"`
	Geometry        *string `json:"geometry,omitempty"` // true-shape outline (JSON)
	EdgeTop         bool    `json:"edge_top,omitempty"`
	EdgeBottom      bool    `json:"edge_bottom,omitempty"`
	EdgeLeft        bool    `json:"edge_left,omitempty"`
	EdgeRight       bool    `json:"edge_right,omitempty"`
	BandItemCode    *int64  `json:"band_item_code,omitempty"`
	BandingLengthMM float64 `json:"banding_length_mm,omitempty"` // banded perimeter × qty
	Quantity        int     `json:"quantity"`
	SourceRef       *string `json:"source_ref,omitempty"`
}

// BandingSummaryResponse totals edge-band consumption for the plan.
type BandingSummaryResponse struct {
	TotalLengthMM float64 `json:"total_length_mm"`
	TotalCost     float64 `json:"total_cost"`
}

// CuttingPlanOrderCostResponse is one order's share of a firmed plan's cost.
type CuttingPlanOrderCostResponse struct {
	OrderRef      string  `json:"order_ref"`
	DemandMeasure float64 `json:"demand_measure"`
	AllocatedCost float64 `json:"allocated_cost"`
}

type CuttingStockPieceResponse struct {
	ID        int64   `json:"id"`
	PlanID    int64   `json:"plan_id"`
	LengthMM  float64 `json:"length_mm"`
	WidthMM   float64 `json:"width_mm,omitempty"`
	HeightMM  float64 `json:"height_mm,omitempty"`
	Quantity  int     `json:"quantity"`
	Lot       *string `json:"lot,omitempty"`
	IsRemnant bool    `json:"is_remnant"`
}

type CuttingPatternPlacementResponse struct {
	Sequence    int64   `json:"sequence"`
	PartID      *int64  `json:"part_id,omitempty"`
	Label       string  `json:"label"`
	LengthMM    float64 `json:"length_mm,omitempty"`
	OffsetMM    float64 `json:"offset_mm,omitempty"`
	PosXMM      float64 `json:"pos_x_mm,omitempty"`
	PosYMM      float64 `json:"pos_y_mm,omitempty"`
	WidthMM     float64 `json:"width_mm,omitempty"`
	HeightMM    float64 `json:"height_mm,omitempty"`
	Rotated     bool    `json:"rotated,omitempty"`
	RotationDeg float64 `json:"rotation_deg,omitempty"`
}

type CuttingPatternResponse struct {
	Sequence        int                               `json:"sequence"`
	StockLengthMM   float64                           `json:"stock_length_mm,omitempty"`
	StockWidthMM    float64                           `json:"stock_width_mm,omitempty"`
	StockHeightMM   float64                           `json:"stock_height_mm,omitempty"`
	RepeatCount     int                               `json:"repeat_count"`
	UsedMM          float64                           `json:"used_mm,omitempty"`
	UsedAreaMM2     float64                           `json:"used_area_mm2,omitempty"`
	KerfLossMM      float64                           `json:"kerf_loss_mm,omitempty"`
	RemnantMM       float64                           `json:"remnant_mm,omitempty"`
	RemnantAreaMM2  float64                           `json:"remnant_area_mm2,omitempty"`
	RemnantWidthMM  float64                           `json:"remnant_width_mm,omitempty"`
	RemnantHeightMM float64                           `json:"remnant_height_mm,omitempty"`
	UtilizationPct  float64                           `json:"utilization_pct"`
	IsRemnant       bool                              `json:"is_remnant"`
	ReusableScrap   bool                              `json:"reusable_remnant"`
	Placements      []CuttingPatternPlacementResponse `json:"placements"`
}

// UnplacedPieceResponse is a demanded piece no stock could hold — a warning the
// operator must resolve (longer than any available stock).
type UnplacedPieceResponse struct {
	Label    string  `json:"label"`
	LengthMM float64 `json:"length_mm,omitempty"`
	WidthMM  float64 `json:"width_mm,omitempty"`
	HeightMM float64 `json:"height_mm,omitempty"`
	Quantity int     `json:"quantity"`
}

// CuttingPlanDetailResponse is the full plan: header, demand, stock, the computed
// patterns and any pieces that could not be placed.
type CuttingPlanDetailResponse struct {
	Plan        CuttingPlanResponse         `json:"plan"`
	Parts       []CuttingPlanPartResponse   `json:"parts"`
	StockPieces []CuttingStockPieceResponse `json:"stock_pieces"`
	Patterns    []CuttingPatternResponse    `json:"patterns"`
	Unplaced    []UnplacedPieceResponse     `json:"unplaced,omitempty"`
	Banding     *BandingSummaryResponse     `json:"banding,omitempty"`
}

// CuttingPlanReleaseResponse summarises the outcome of firming a plan.
type CuttingPlanReleaseResponse struct {
	PlanID            int64  `json:"plan_id"`
	PlanCode          int64  `json:"plan_code"`
	Status            string `json:"status"`
	ConsumptionMode   string `json:"consumption_mode"`
	WarehouseID       int64  `json:"warehouse_id"`
	BarsConsumed      int    `json:"bars_consumed"`      // full bars drawn from stock (with baixa)
	RemnantsConsumed  int    `json:"remnants_consumed"`  // inventory offcuts reused
	RemnantsGenerated int    `json:"remnants_generated"` // new reusable offcuts created
}

// GeneratedCuttingPlanSummary describes one plan created from orders.
type GeneratedCuttingPlanSummary struct {
	PlanID           int64    `json:"plan_id"`
	PlanCode         int64    `json:"plan_code"`
	CutType          string   `json:"cut_type"`
	MaterialItemCode int64    `json:"material_item_code"`
	PartCount        int      `json:"part_count"`   // distinct part lines
	TotalPieces      int      `json:"total_pieces"` // sum of quantities
	OrderRefs        []string `json:"order_refs"`
}

// GenerateCuttingDemandResponse is the outcome of generating plans from orders.
type GenerateCuttingDemandResponse struct {
	Plans    []GeneratedCuttingPlanSummary `json:"plans"`
	Warnings []string                      `json:"warnings,omitempty"`
}

// CutProgramStepResponse is one cut in the shop-floor program.
type CutProgramStepResponse struct {
	Sequence    int     `json:"sequence"`
	Label       string  `json:"label"`
	OffsetMM    float64 `json:"offset_mm,omitempty"`
	LengthMM    float64 `json:"length_mm,omitempty"`
	PosXMM      float64 `json:"pos_x_mm,omitempty"`
	PosYMM      float64 `json:"pos_y_mm,omitempty"`
	WidthMM     float64 `json:"width_mm,omitempty"`
	HeightMM    float64 `json:"height_mm,omitempty"`
	RotationDeg float64 `json:"rotation_deg,omitempty"`
}

// CutProgramCutResponse is one full edge-to-edge guillotine cut (panel saw program).
type CutProgramCutResponse struct {
	Sequence   int     `json:"sequence"`
	Level      int     `json:"level"` // 0 = primary head cut; deeper = sub-panel cuts
	Axis       string  `json:"axis"`  // VERTICAL | HORIZONTAL
	PositionMM float64 `json:"position_mm"`
	FromMM     float64 `json:"from_mm"`
	ToMM       float64 `json:"to_mm"`
}

type CutProgramPatternResponse struct {
	Sequence      int                      `json:"sequence"`
	RepeatCount   int                      `json:"repeat_count"`
	StockLengthMM float64                  `json:"stock_length_mm,omitempty"`
	StockWidthMM  float64                  `json:"stock_width_mm,omitempty"`
	StockHeightMM float64                  `json:"stock_height_mm,omitempty"`
	Steps         []CutProgramStepResponse `json:"steps"`
	// Cuts is the derived guillotine cut sequence for 2D sheets (the seccionadora
	// program). Empty for 1D bars and for layouts that are not guillotine-separable.
	Cuts []CutProgramCutResponse `json:"cuts,omitempty"`
}

// CutProgramResponse is the ordered cut program for a plan.
type CutProgramResponse struct {
	PlanID   int64                       `json:"plan_id"`
	PlanCode int64                       `json:"plan_code"`
	CutType  string                      `json:"cut_type"`
	Patterns []CutProgramPatternResponse `json:"patterns"`
}

// CutScheduleResponse summarises a plan booked onto a machine.
type CutScheduleResponse struct {
	PlanID        int64     `json:"plan_id"`
	PlanCode      int64     `json:"plan_code"`
	ScheduleCode  int64     `json:"schedule_code"`
	MachineCode   int64     `json:"machine_code"`
	PlannedPieces int       `json:"planned_pieces"`
	ScheduleDate  time.Time `json:"schedule_date"`
}

// CuttingSettingsResponse is the company-level default config.
type CuttingSettingsResponse struct {
	DefaultConsumptionMode string  `json:"default_consumption_mode"`
	DefaultMinRemnantMM    float64 `json:"default_min_remnant_mm"`
	DefaultWarehouseID     *int64  `json:"default_warehouse_id,omitempty"`
}

// StockRemnantResponse is a reusable offcut in inventory.
type StockRemnantResponse struct {
	ID          int64   `json:"id"`
	ItemCode    int64   `json:"item_code"`
	WarehouseID int64   `json:"warehouse_id"`
	LengthMM    float64 `json:"length_mm,omitempty"`
	WidthMM     float64 `json:"width_mm,omitempty"`
	HeightMM    float64 `json:"height_mm,omitempty"`
	Lot         *string `json:"lot,omitempty"`
	HeatNumber  *string `json:"heat_number,omitempty"`
	Certificate *string `json:"certificate,omitempty"`
	Status      string  `json:"status"`
	UnitCost    float64 `json:"unit_cost"`
}
