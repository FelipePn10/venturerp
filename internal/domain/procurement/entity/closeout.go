package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---- Receiving notice + divergences (FAVR) ----

type ReceivingNotice struct {
	ID                int64
	EnterpriseCode    int64
	NoticeNumber      int64
	SupplierCode      *int64
	PurchaseOrderCode *int64
	CarrierCode       *int64
	Status            string
	Dock              *string
	ScheduledAt       *time.Time
	ArrivedAt         *time.Time
	InvoiceNumber     *string
	Blocked           bool
	Notes             *string
	CreatedBy         *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Items             []*ReceivingNoticeItem
}

type ReceivingNoticeItem struct {
	ID                    int64
	NoticeID              int64
	PurchaseOrderItemCode *int64
	ItemCode              int64
	Mask                  string
	ExpectedQty           float64
	ReceivedQty           float64
	Unit                  *string
	Notes                 *string
}

type ReceivingDivergence struct {
	ID                    int64
	NoticeID              *int64
	PurchaseOrderCode     *int64
	PurchaseOrderItemCode *int64
	SupplierCode          *int64
	ItemCode              *int64
	Mask                  string
	DivergenceType        string
	ExpectedQty           float64
	ActualQty             float64
	ExpectedPrice         *float64
	ActualPrice           *float64
	Resolution            string
	AffectsSupplierScore  bool
	Notes                 *string
	CreatedBy             *uuid.UUID
	CreatedAt             time.Time
	ResolvedAt            *time.Time
}

// ---- Supplier EDI (structured) (FEDS) ----

type SupplierEDIMessage struct {
	ID                int64
	EnterpriseCode    int64
	SupplierCode      *int64
	Direction         string
	MessageType       string
	PurchaseOrderCode *int64
	ExternalReference *string
	Status            string
	DivergenceCount   int
	Payload           []byte
	Notes             *string
	CreatedBy         *uuid.UUID
	CreatedAt         time.Time
	ProcessedAt       *time.Time
	Lines             []*SupplierEDILine
}

type SupplierEDILine struct {
	ID                    int64
	MessageID             int64
	PurchaseOrderItemCode *int64
	ItemCode              *int64
	Mask                  string
	ConfirmedQty          float64
	ConfirmedPrice        float64
	ConfirmedDate         *time.Time
	Divergence            *string
	Notes                 *string
}

// ---- Import landed cost (FREC0203 / FIMP) ----

type ImportProcess struct {
	ID                int64
	EnterpriseCode    int64
	ProcessNumber     int64
	SupplierCode      *int64
	PurchaseOrderCode *int64
	Reference         *string
	Incoterm          *string
	Currency          string
	ExchangeRate      float64
	ApportionBasis    string
	Status            string
	Notes             *string
	CreatedBy         *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	NationalizedAt    *time.Time
	Items             []*ImportProcessItem
	Expenses          []*ImportExpense
}

type ImportProcessItem struct {
	ID                  int64
	ProcessID           int64
	ItemCode            int64
	Mask                string
	Quantity            float64
	Weight              float64
	FobUnitPrice        float64
	ApportionedExpenses float64
	LandedUnitCost      float64
	Notes               *string
}

type ImportExpense struct {
	ID          int64
	ProcessID   int64
	ExpenseType string
	Amount      float64
	InItemCost  bool
	Notes       *string
}

// ---- Procurement parameters (FUTL0125) ----

type ProcurementParameter struct {
	ID             int64
	EnterpriseCode int64
	Domain         string
	Key            string
	Value          string
	ValueType      string
	Description    *string
	UpdatedBy      *uuid.UUID
	UpdatedAt      time.Time
}

// ---- Supplier homologation (FAVF0203) ----

type SupplierHomologation struct {
	ID           int64
	SupplierCode int64
	Status       string
	IQFScore     *float64
	Category     *string
	ValidUntil   *time.Time
	Notes        *string
	DecidedBy    *uuid.UUID
	DecidedAt    time.Time
}

// ---- Pure business logic (unit-tested, no I/O) ----

// ComputeLandedCosts apportions the cost-composing expenses across the process
// items by the chosen basis and computes each item's nationalized (landed) unit
// cost in local currency. FOB prices are in the process currency and converted by
// ExchangeRate. It mutates ApportionedExpenses and LandedUnitCost on each item.
func ComputeLandedCosts(p *ImportProcess) {
	if p == nil || len(p.Items) == 0 {
		return
	}
	rate := p.ExchangeRate
	if rate <= 0 {
		rate = 1
	}
	var expensesInCost float64
	for _, e := range p.Expenses {
		if e.InItemCost {
			expensesInCost += e.Amount
		}
	}
	bases := make([]float64, len(p.Items))
	var totalBasis float64
	for i, it := range p.Items {
		switch p.ApportionBasis {
		case "WEIGHT":
			bases[i] = it.Weight
		case "QUANTITY":
			bases[i] = it.Quantity
		default: // VALUE
			bases[i] = it.Quantity * it.FobUnitPrice * rate
		}
		totalBasis += bases[i]
	}
	for i, it := range p.Items {
		var share float64
		switch {
		case totalBasis > 0:
			share = bases[i] / totalBasis
		case len(p.Items) > 0:
			share = 1.0 / float64(len(p.Items))
		}
		it.ApportionedExpenses = expensesInCost * share
		fobLocalTotal := it.Quantity * it.FobUnitPrice * rate
		landedTotal := fobLocalTotal + it.ApportionedExpenses
		if it.Quantity > 0 {
			it.LandedUnitCost = landedTotal / it.Quantity
		} else {
			it.LandedUnitCost = 0
		}
	}
}

// DetectEDILineDivergence compares a supplier-confirmed line against the purchase
// order line and returns a comma-separated list of divergence codes (QTY, PRICE,
// DATE), or an empty string when everything matches within tolerance.
func DetectEDILineDivergence(poQty, poPrice float64, poDate *time.Time, confirmedQty, confirmedPrice float64, confirmedDate *time.Time, qtyTol, priceTol float64) string {
	var codes []string
	if !within(poQty, confirmedQty, qtyTol) {
		codes = append(codes, "QTY")
	}
	if !within(poPrice, confirmedPrice, priceTol) {
		codes = append(codes, "PRICE")
	}
	if poDate != nil && confirmedDate != nil && !sameDay(*poDate, *confirmedDate) {
		codes = append(codes, "DATE")
	}
	return strings.Join(codes, ",")
}

// HomologationStatusForIQF maps an IQF score to a homologation status using the
// configured thresholds (homologated >= homologatedMin; conditional >=
// conditionalMin; otherwise rejected).
func HomologationStatusForIQF(iqf, homologatedMin, conditionalMin float64) string {
	switch {
	case iqf >= homologatedMin:
		return "HOMOLOGATED"
	case iqf >= conditionalMin:
		return "CONDITIONAL"
	default:
		return "REJECTED"
	}
}

func within(a, b, tol float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	if tol < 0 {
		tol = 0
	}
	return diff <= tol+1e-9
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
