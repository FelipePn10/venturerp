package response

import (
	"time"

	"github.com/google/uuid"
)

// MaintenancePlanResponse is the API representation of a maintenance plan.
type MaintenancePlanResponse struct {
	ID              int64      `json:"id"`
	Code            int64      `json:"code"`
	MachineID       int64      `json:"machine_id"`
	WorkCenterID    *int64     `json:"work_center_id,omitempty"`
	Description     string     `json:"description"`
	Frequency       string     `json:"frequency"`
	FrequencyDays   int        `json:"frequency_days"`
	EstimatedHours  float64    `json:"estimated_hours"`
	LastExecutedAt  *time.Time `json:"last_executed_at,omitempty"`
	NextScheduledAt *time.Time `json:"next_scheduled_at,omitempty"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       uuid.UUID  `json:"created_by"`
}

// MaintenanceOrderResponse is the API representation of a maintenance order.
type MaintenanceOrderResponse struct {
	ID             int64      `json:"id"`
	PlanID         int64      `json:"plan_id"`
	MachineID      *int64     `json:"machine_id,omitempty"`
	WorkCenterID   *int64     `json:"work_center_id,omitempty"`
	ScheduledDate  time.Time  `json:"scheduled_date"`
	EstimatedHours float64    `json:"estimated_hours"`
	ActualHours    *float64   `json:"actual_hours,omitempty"`
	Status         string     `json:"status"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
