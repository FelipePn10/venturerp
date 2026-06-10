package response

import (
	"time"

	"github.com/google/uuid"
)

// OverheadAllocationResponse is the API representation of an overhead allocation.
type OverheadAllocationResponse struct {
	Code            int64                      `json:"code"`
	CostCenterCode  int64                      `json:"cost_center_code"`
	PlanAccountCode *int64                     `json:"plan_account_code,omitempty"`
	AccountCode     *string                    `json:"account_code,omitempty"`
	PeriodStart     time.Time                  `json:"period_start"`
	PeriodEnd       time.Time                  `json:"period_end"`
	AllocationType  string                     `json:"allocation_type"`
	BaseCode        *int64                     `json:"base_code,omitempty"`
	Targets         []AllocationTargetResponse `json:"targets,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	CreatedBy       uuid.UUID                  `json:"created_by"`
}

// AllocationTargetResponse is the API representation of an allocation target.
type AllocationTargetResponse struct {
	Code           int64     `json:"code"`
	OverheadCode   int64     `json:"overhead_code"`
	CostCenterCode int64     `json:"cost_center_code"`
	Percentage     float64   `json:"percentage"`
	Amount         float64   `json:"amount"`
	CreatedAt      time.Time `json:"created_at"`
}
