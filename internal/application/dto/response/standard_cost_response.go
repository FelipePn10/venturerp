package response

import "time"

type WorkCenterOptionResponse struct {
	ID          int64   `json:"id"`
	Code        int64   `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"is_active"`
}

type WorkCenterOptionPageResponse struct {
	Items  []WorkCenterOptionResponse `json:"items"`
	Total  int64                      `json:"total"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
}

type WorkCenterCostResponse struct {
	ID                 int64     `json:"id"`
	WorkCenterID       int64     `json:"work_center_id"`
	CostPerHour        float64   `json:"cost_per_hour"`
	MachineCostPerHour float64   `json:"machine_cost_per_hour"`
	LaborCostPerHour   float64   `json:"labor_cost_per_hour"`
	Currency           string    `json:"currency"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ItemPurchaseCostResponse struct {
	ID        int64     `json:"id"`
	ItemCode  int64     `json:"item_code"`
	UnitCost  float64   `json:"unit_cost"`
	Currency  string    `json:"currency"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CostRollupResponse struct {
	ItemCode     int64     `json:"item_code"`
	Mask         string    `json:"mask"`
	MaterialCost float64   `json:"material_cost"`
	LaborCost    float64   `json:"labor_cost"`
	OverheadCost float64   `json:"overhead_cost"`
	TotalCost    float64   `json:"total_cost"`
	Currency     string    `json:"currency"`
	CalculatedAt time.Time `json:"calculated_at"`
}
