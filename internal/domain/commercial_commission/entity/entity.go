package entity

import (
	"github.com/google/uuid"
	"time"
)

type Settings struct {
	EnterpriseID                                      int64
	CompetenceEvent, InvoiceSharePct, ReceiptSharePct string
	UpdatedAt                                         time.Time
	UpdatedBy                                         uuid.UUID
}

// LedgerEntry é devolvida direto pelo handler: as tags são o contrato da API.
// Sem elas os campos saíam em PascalCase e uma renomeação em Go quebrava o
// cliente em silêncio.
type LedgerEntry struct {
	Code               int64      `json:"code"`
	RepresentativeCode int64      `json:"representative_code"`
	SalesOrderCode     int64      `json:"sales_order_code"`
	FiscalExitID       *int64     `json:"fiscal_exit_id,omitempty"`
	ReceivableID       *int64     `json:"receivable_id,omitempty"`
	EventType          string     `json:"event_type"`
	CompetenceDate     time.Time  `json:"competence_date"`
	BaseAmount         string     `json:"base_amount"`
	CommissionPct      string     `json:"commission_pct"`
	Amount             string     `json:"amount"`
	Status             string     `json:"status"`
	ReversalOf         *int64     `json:"reversal_of,omitempty"`
	OccurredAt         time.Time  `json:"occurred_at"`
	ReconciledAt       *time.Time `json:"reconciled_at,omitempty"`
	PaidAt             *time.Time `json:"paid_at,omitempty"`
	PaymentReference   *string    `json:"payment_reference,omitempty"`
}
type Filter struct {
	RepresentativeCode *int64
	Status             *string
	From, To           *time.Time
	Limit, Offset      int
}
type TransitionCommand struct {
	Code             int64
	Action, Reason   string
	PaymentReference *string
	IdempotencyKey   string
	ActorID          uuid.UUID
}
