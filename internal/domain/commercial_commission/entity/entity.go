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
type LedgerEntry struct {
	Code, RepresentativeCode, SalesOrderCode  int64
	FiscalExitID, ReceivableID                *int64
	EventType                                 string
	CompetenceDate                            time.Time
	BaseAmount, CommissionPct, Amount, Status string
	ReversalOf                                *int64
	OccurredAt                                time.Time
	ReconciledAt, PaidAt                      *time.Time
	PaymentReference                          *string
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
