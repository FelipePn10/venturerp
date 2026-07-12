package request

type CreateProductionPlanDTO struct {
	Code                int64                  `json:"code"`
	Name                string                 `json:"name"`
	IndependentDemands  string                 `json:"independent_demands"` // NO, FROM_DATE, ALL
	GroupSameDateOrders bool                   `json:"group_same_date_orders"`
	PlanningTypes       []string               `json:"planning_types"`
	Classification      *string                `json:"classification"`
	ClassItemCodes      *string                `json:"class_item_codes"`
	OrderItemCode       *int64                 `json:"order_item_code"`
	Parameters          map[string]interface{} `json:"parameters"`
}

type UpdateProductionPlanDTO struct {
	Code                int64                  `json:"code"`
	Name                string                 `json:"name"`
	IndependentDemands  string                 `json:"independent_demands"`
	GroupSameDateOrders bool                   `json:"group_same_date_orders"`
	PlanningTypes       []string               `json:"planning_types"`
	Classification      *string                `json:"classification"`
	ClassItemCodes      *string                `json:"class_item_codes"`
	OrderItemCode       *int64                 `json:"order_item_code"`
	Parameters          map[string]interface{} `json:"parameters"`
}
