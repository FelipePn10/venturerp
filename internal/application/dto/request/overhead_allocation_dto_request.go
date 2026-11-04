package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateOverheadAllocationDTO struct {
	CostCenterCode  int64                       `json:"cost_center_code"`
	PlanAccountCode *int64                      `json:"plan_account_code,omitempty"`
	AccountCode     *string                     `json:"account_code,omitempty"`
	PeriodStart     string                      `json:"period_start"`
	PeriodEnd       string                      `json:"period_end"`
	AllocationType  string                      `json:"allocation_type"`
	BaseCode        *int64                      `json:"base_code,omitempty"`
	CreatedBy       uuid.UUID                   `json:"created_by"`
	Targets         []CreateAllocationTargetDTO `json:"targets"`
}

type CreateAllocationTargetDTO struct {
	CostCenterCoed int64           `json:"cost_center_code"`
	Percentage     decimal.Decimal `json:"percentage"`
	Amount         decimal.Decimal `json:"amount"`
}
