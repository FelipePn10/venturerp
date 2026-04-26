package request

import "github.com/google/uuid"

type CreateAllocationBaseDTO struct {
	Code        string                        `json:"code"`
	Description string                        `json:"description"`
	Period      string                        `json:"period"`
	Observation *string                       `json:"observation,omitempty"`
	Items       []CreateAllocationBaseItemDTO `json:"items"`
	CreatedBy   uuid.UUID                     `json:"created_by"`
}

type CreateAllocationBaseItemDTO struct {
	CostCenterID int64   `json:"cost_center_id"`
	Amount       float64 `json:"amount"`
	Percentage   float64 `json:"percentage"`
}
