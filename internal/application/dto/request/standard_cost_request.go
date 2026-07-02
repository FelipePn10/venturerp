package request

type UpsertWorkCenterCostDTO struct {
	WorkCenterID int64   `json:"work_center_id"`
	CostPerHour  float64 `json:"cost_per_hour"` // blended rate (machine-rate fallback)
	// Optional machine × labor split. When omitted, machine uses cost_per_hour and labor is 0.
	MachineCostPerHour float64 `json:"machine_cost_per_hour"`
	LaborCostPerHour   float64 `json:"labor_cost_per_hour"`
	Currency           string  `json:"currency"`
	UpdatedBy          string  `json:"updated_by"`
}

type UpsertItemPurchaseCostDTO struct {
	ItemCode  int64   `json:"item_code"`
	UnitCost  float64 `json:"unit_cost"`
	Currency  string  `json:"currency"`
	UpdatedBy string  `json:"updated_by"`
}

type CostRollupDTO struct {
	ItemCode int64  `json:"item_code"`
	Mask     string `json:"mask"`
	// LotSize is the reference production lot used to amortize operation setup over the
	// standard cost (setup/lote ÷ lot). Defaults to 1 (setup fully charged per unit).
	LotSize      float64 `json:"lot_size"`
	CalculatedBy string  `json:"calculated_by"`
}
