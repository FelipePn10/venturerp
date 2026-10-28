package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateIndependentDemandDTO struct {
	CodeDemand     int64           `json:"code_demand"`
	ItemCode       int64           `json:"item_code"`
	Mask           *string         `json:"mask,omitempty"`
	CostCenterCode *int64          `json:"cost_center_code,omitempty"`
	Quantity       decimal.Decimal `json:"quantity"`
	DemandDate     string          `json:"demand_date"`
	CreatedBy      uuid.UUID       `json:"created_by"`
}

type UpdateIndependentDemandDTO struct {
	CodeDemand     int64           `json:"code_demand"`
	ItemCode       int64           `json:"item_code"`
	Mask           *string         `json:"mask,omitempty"`
	CostCenterCode *int64          `json:"cost_center_code,omitempty"`
	Quantity       decimal.Decimal `json:"quantity"`
	DemandDate     string          `json:"demand_date"`
}
