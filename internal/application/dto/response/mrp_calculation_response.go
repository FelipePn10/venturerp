package response

import (
	"github.com/shopspring/decimal"
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
	// SuggestedDate é a data em que o cálculo precisa do suprimento.
	SuggestedDate *time.Time `json:"suggested_date,omitempty"`
	// Ordem planejada de origem: número e necessidade atual.
	SourceOrderNumber *int64     `json:"source_order_number,omitempty"`
	SourceNeedDate    *time.Time `json:"source_need_date,omitempty"`
	// Tratamento (vazio = pendente): REPROGRAMADA, CANCELADA ou TRATADA.
	HandlingAction string     `json:"handling_action,omitempty"`
	HandlingNote   string     `json:"handling_note,omitempty"`
	HandledAt      *time.Time `json:"handled_at,omitempty"`
	HandledByName  string     `json:"handled_by_name,omitempty"`
	// AffectedDemands: pedidos/demandas de origem que dependem da ordem da
	// mensagem (pelo rastreamento do último cálculo).
	AffectedDemands []string `json:"affected_demands,omitempty"`
}

// MRPItemProfileResponse is the API representation of an MRP item profile bucket.
type MRPItemProfileResponse struct {
	ItemCode        int64           `json:"item_code"`
	PlanCode        int64           `json:"plan_code"`
	CalculationDate time.Time       `json:"calculation_date"`
	Demand          decimal.Decimal `json:"demand"`
	OrdersPlanned   decimal.Decimal `json:"orders_planned"`
	OrdersFirm      decimal.Decimal `json:"orders_firm"`
	StockProjected  decimal.Decimal `json:"stock_projected"`
	LLC             int             `json:"llc"`
	NeedDate        time.Time       `json:"need_date"`
	CreatedAt       time.Time       `json:"created_at"`
}

type MRPItemProfileConsultationResponse struct {
	Position string                     `json:"position"`
	Rows     []*MRPItemProfileResponse  `json:"rows"`
	Totals   map[string]decimal.Decimal `json:"totals"`
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
