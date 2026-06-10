package response

import (
	"time"

	"github.com/google/uuid"
)

// IndependentDemandResponse is the API representation of an independent demand.
type IndependentDemandResponse struct {
	CodeDemand     int64     `json:"code_demand"`
	ItemCode       int64     `json:"item_code"`
	Mask           *string   `json:"mask,omitempty"`
	CostCenterCode *int64    `json:"cost_center_code,omitempty"`
	Quantity       float64   `json:"quantity"`
	DemandDate     time.Time `json:"demand_date"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      uuid.UUID `json:"created_by"`
}
