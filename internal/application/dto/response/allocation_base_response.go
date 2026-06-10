package response

import (
	"time"

	"github.com/google/uuid"
)

// AllocationBaseResponse is the API representation of an allocation base.
type AllocationBaseResponse struct {
	Code        int32                        `json:"code"`
	Description string                       `json:"description"`
	Period      string                       `json:"period"`
	Observation *string                      `json:"observation,omitempty"`
	Items       []AllocationBaseItemResponse `json:"items,omitempty"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
	CreatedBy   uuid.UUID                    `json:"created_by"`
}

// AllocationBaseItemResponse is the API representation of an allocation base line.
type AllocationBaseItemResponse struct {
	AllocationBaseCode int32     `json:"allocation_base_code"`
	CostCenterCode     int32     `json:"cost_center_code"`
	Amount             float64   `json:"amount"`
	Percentage         float64   `json:"percentage"`
	CreatedAt          time.Time `json:"created_at"`
}
