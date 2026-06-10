package response

import (
	"time"

	"github.com/google/uuid"
)

// MRPCalculationLogResponse is the API representation of an MRP run log.
type MRPCalculationLogResponse struct {
	Code        int64                  `json:"code"`
	PlanCode    int64                  `json:"plan_code"`
	StartedAt   time.Time              `json:"started_at"`
	FinishedAt  *time.Time             `json:"finished_at,omitempty"`
	Status      string                 `json:"status"`
	Errors      map[string]interface{} `json:"errors,omitempty"`
	TotalItems  int                    `json:"total_items"`
	TotalOrders int                    `json:"total_orders"`
	CreatedAt   time.Time              `json:"created_at"`
}

// MRPExceptionMessageResponse is the API representation of an MRP exception.
type MRPExceptionMessageResponse struct {
	Code        int64     `json:"code"`
	PlanCode    int64     `json:"plan_code"`
	ItemCode    int64     `json:"item_code"`
	MessageType string    `json:"message_type"`
	SourceCode  *int64    `json:"source_code,omitempty"`
	SourceType  *string   `json:"source_type,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// MRPItemProfileResponse is the API representation of an MRP item profile bucket.
type MRPItemProfileResponse struct {
	ItemCode        int64     `json:"item_code"`
	PlanCode        int64     `json:"plan_code"`
	CalculationDate time.Time `json:"calculation_date"`
	Demand          float64   `json:"demand"`
	OrdersPlanned   float64   `json:"orders_planned"`
	OrdersFirm      float64   `json:"orders_firm"`
	StockProjected  float64   `json:"stock_projected"`
	LLC             int       `json:"llc"`
	NeedDate        time.Time `json:"need_date"`
	CreatedAt       time.Time `json:"created_at"`
}

// ConfiguredItemRuleResponse is the API representation of a configured item rule.
type ConfiguredItemRuleResponse struct {
	Code      int64     `json:"code"`
	ItemCode  int64     `json:"item_code"`
	TableType string    `json:"table_type"`
	FieldName string    `json:"field_name"`
	RuleType  string    `json:"rule_type"`
	RuleValue string    `json:"rule_value"`
	Sequence  int       `json:"sequence"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy uuid.UUID `json:"created_by"`
}
