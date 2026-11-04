package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateAllocationBaseDTO struct {
	Code        int32                         `json:"code"`
	Description string                        `json:"description"`
	Period      string                        `json:"period"`
	Observation *string                       `json:"observation,omitempty"`
	Items       []CreateAllocationBaseItemDTO `json:"items"`
	CreatedBy   uuid.UUID                     `json:"created_by"`
}

type CreateAllocationBaseItemDTO struct {
	CostCenterCode int32           `json:"cost_center_code"`
	Amount         decimal.Decimal `json:"amount"`
	Percentage     decimal.Decimal `json:"percentage"`
}
