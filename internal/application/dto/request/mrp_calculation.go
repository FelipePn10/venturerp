package request

import "github.com/google/uuid"

type RunMRPCalculationDTO struct {
	PlanCode    int64 `json:"plan_code"`
	GenerateLLC bool  `json:"generate_llc"`
}

type CreatePlannedOrderDTO struct {
	ItemCode       int64   `json:"item_code"`
	Mask           *string `json:"mask,omitempty"`
	Quantity       float64 `json:"quantity"`
	OrderType      string  `json:"order_type"`
	NeedDate       string  `json:"need_date"`
	CostCenterCode *int64  `json:"cost_center_code,omitempty"`
	EmployeeCode   *int64  `json:"employee_code,omitempty"`
	MachineCode    *int64  `json:"machine_code,omitempty"`
	ProductionTime float64 `json:"production_time"`
	Notes          *string `json:"notes,omitempty"`
}

type FirmOrderDTO struct {
	OrderCode int64 `json:"order_code"`
}

type CreateConfiguredItemRuleDTO struct {
	ItemCode  int64     `json:"item_code"`
	TableType string    `json:"table_type"`
	FieldName string    `json:"field_name"`
	RuleType  string    `json:"rule_type"`
	RuleValue string    `json:"rule_value"`
	Sequence  int       `json:"sequence"`
	CreatedBy uuid.UUID `json:"created_by"`
}
