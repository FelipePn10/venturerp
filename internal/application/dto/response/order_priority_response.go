package response

import (
	"time"

	"github.com/google/uuid"
)

// OrderPriorityResponse is the API representation of an order priority band.
type OrderPriorityResponse struct {
	Code          int64     `json:"code"`
	IntervalStart float64   `json:"interval_start"`
	IntervalEnd   float64   `json:"interval_end"`
	Priority      string    `json:"priority"`
	Description   *string   `json:"description,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedBy     uuid.UUID `json:"created_by"`
}
