package entity

import (
	"github.com/shopspring/decimal"
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
	ExpectedQty           decimal.Decimal
	ReceivedQty           decimal.Decimal
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
	ExpectedQty           decimal.Decimal
	ActualQty             decimal.Decimal
	ExpectedPrice         *decimal.Decimal
	ActualPrice           *decimal.Decimal
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
	ConfirmedQty          decimal.Decimal
	ConfirmedPrice        decimal.Decimal
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
	ExchangeRate      decimal.Decimal
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
	Quantity            decimal.Decimal
	Weight              decimal.Decimal
	FobUnitPrice        decimal.Decimal
	ApportionedExpenses decimal.Decimal
	LandedUnitCost      decimal.Decimal
	Notes               *string
}

type ImportExpense struct {
	ID          int64
	ProcessID   int64
	ExpenseType string
	Amount      decimal.Decimal
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
	if !rate.IsPositive() {
		rate = decimal.NewFromInt(1)
	}
	expensesInCost := decimal.Zero
	for _, e := range p.Expenses {
		if e.InItemCost {
			expensesInCost = expensesInCost.Add(e.Amount)
		}
	}
	bases := make([]decimal.Decimal, len(p.Items))
	totalBasis := decimal.Zero
	for i, it := range p.Items {
		switch p.ApportionBasis {
		case "WEIGHT":
			bases[i] = it.Weight
		case "QUANTITY":
			bases[i] = it.Quantity
		default: // VALUE
			bases[i] = it.Quantity.Mul(it.FobUnitPrice).Mul(rate)
		}
		totalBasis = totalBasis.Add(bases[i])
	}
	// O rateio é exato, arredondado na escala da coluna (4 casas); a última
	// linha leva a diferença, para a soma das partes fechar com o total.
	distribuido := decimal.Zero
	for i, it := range p.Items {
		var parte decimal.Decimal
		switch {
		case i == len(p.Items)-1:
			parte = expensesInCost.Sub(distribuido)
		case totalBasis.IsPositive():
			parte = expensesInCost.Mul(bases[i]).DivRound(totalBasis, 4)
		default:
			parte = expensesInCost.DivRound(decimal.NewFromInt(int64(len(p.Items))), 4)
		}
		distribuido = distribuido.Add(parte)
		it.ApportionedExpenses = parte
		landedTotal := it.Quantity.Mul(it.FobUnitPrice).Mul(rate).Add(parte)
		if it.Quantity.IsPositive() {
			it.LandedUnitCost = landedTotal.DivRound(it.Quantity, 6)
		} else {
			it.LandedUnitCost = decimal.Zero
		}
	}
}

// DetectEDILineDivergence compares a supplier-confirmed line against the purchase
// order line and returns a comma-separated list of divergence codes (QTY, PRICE,
// DATE), or an empty string when everything matches within tolerance.
func DetectEDILineDivergence(poQty, poPrice decimal.Decimal, poDate *time.Time, confirmedQty, confirmedPrice decimal.Decimal, confirmedDate *time.Time, qtyTol, priceTol decimal.Decimal) string {
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

func within(a, b, tol decimal.Decimal) bool {
	if tol.IsNegative() {
		tol = decimal.Zero
	}
	return !a.Sub(b).Abs().GreaterThan(tol)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
