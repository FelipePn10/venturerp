package response

import (
	"time"

	"github.com/google/uuid"
)

// CostCenterResponse is the API representation of a cost center.
type CostCenterResponse struct {
	ID          int64      `json:"id"`
	Code        int32      `json:"code"`
	Description string     `json:"description"`
	ParentCode  *int32     `json:"parent_code,omitempty"`
	Type        string     `json:"type"`
	IsRatio     bool       `json:"is_ratio"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   uuid.UUID  `json:"created_by"`
}
