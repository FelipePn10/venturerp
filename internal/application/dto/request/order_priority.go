package request

import "github.com/google/uuid"

type CreateOrderPriorityDTO struct {
	Code          int64     `json:"code"`
	IntervalStart float64   `json:"interval_start"`
	IntervalEnd   float64   `json:"interval_end"`
	Priority      string    `json:"priority"`
	Description   *string   `json:"description,omitempty"`
	CreatedBy     uuid.UUID `json:"created_by"`
}

type UpdateOrderPriorityDTO struct {
	Code          int64     `json:"code"`
	IntervalStart float64   `json:"interval_start"`
	IntervalEnd   float64   `json:"interval_end"`
	Priority      string    `json:"priority"`
	Description   *string   `json:"description,omitempty"`
	UpdateBy      uuid.UUID `json:"updated_by"`
}
