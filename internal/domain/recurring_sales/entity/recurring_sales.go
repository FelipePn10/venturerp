package entity

import (
	"encoding/json"
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

type MovementType string

const (
	MovementSale          MovementType = "SALE"
	MovementUpgrade       MovementType = "UPGRADE"
	MovementDowngrade     MovementType = "DOWNGRADE"
	MovementAdjustment    MovementType = "ADJUSTMENT"
	MovementRecalculation MovementType = "RECALCULATION"
	MovementCancellation  MovementType = "CANCELLATION"
)

type TermType string

const (
	TermIndefinite TermType = "INDEFINITE"
	TermFixed      TermType = "FIXED"
)

type CommissionBase string

type LifecycleStatus string

const (
	LifecycleActive    LifecycleStatus = "ATIVA"
	LifecycleSuspended LifecycleStatus = "SUSPENSA"
	LifecycleCancelled LifecycleStatus = "CANCELADA"
	LifecycleEnded     LifecycleStatus = "ENCERRADA"
)

const (
	CommissionBaseOriginal CommissionBase = "ORIGINAL"
	CommissionBaseAdjusted CommissionBase = "ADJUSTED"
)

type Parameters struct {
	EnterpriseCode              int64
	CurrentMonthBillingLimitDay int
	GroupOrderItemTotal         bool
	IndefiniteDeliveryDay       int
	FixedTermDeliveryDay        int
	ConsiderDiscountsAdditions  bool
	GenericRepresentativeCode   *int64
	GenericSalesPlanCode        *int64
	UpdatedAt                   time.Time
	UpdatedBy                   uuid.UUID
}

type AdjustmentDate struct {
	Code              int64
	EnterpriseCode    int64
	CustomerCode      int64
	EstablishmentCode *int64
	AdjustmentDate    time.Time
	Notes             *string
	CreatedAt         time.Time
	CreatedBy         uuid.UUID
}

type RecurringSale struct {
	Code                      int64
	EnterpriseCode            int64
	CustomerCode              int64
	EstablishmentCode         *int64
	ItemCode                  int64
	ItemMask                  *string
	SalesPlanCode             *int64
	MovementType              MovementType
	TermType                  TermType
	SaleDate                  time.Time
	NextAdjustmentDate        *time.Time
	MonthsQuantity            *int
	PaymentsQuantity          *int
	GraceMonths               int
	PaymentValue              *decimal.Decimal
	Quantity                  decimal.Decimal
	UnitValue                 decimal.Decimal
	Reason                    *string
	GeneratedOrderCode        *int64
	GeneratedOrderAt          *time.Time
	SourceRecurringSaleCode   *int64
	OriginalAdjustmentCode    *int64
	AdjustmentPercent         *decimal.Decimal
	IsActive                  bool
	LifecycleStatus           LifecycleStatus
	EffectiveFrom             *time.Time
	EffectiveUntil            *time.Time
	CancellationEffectiveDate *time.Time
	FutureOrdersPolicy        *string
	CancelledBy               *uuid.UUID
	Frequency                 string
	PriceTableCode            *int64
	CurrencyCode              string
	AdjustmentIndex           *string
	AdjustmentPeriodMonths    *int
	AdjustmentFloorPct        *decimal.Decimal
	AdjustmentCapPct          *decimal.Decimal
	BillingPolicy             json.RawMessage
	DeliveryPolicy            json.RawMessage
	TaxPolicy                 json.RawMessage
	CostCenterCode            *int64
	RenewalPolicy             string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	CreatedBy                 uuid.UUID
	Representatives           []*Representative
}

type Representative struct {
	Code                   int64
	RecurringSaleCode      int64
	RepresentativeCode     int64
	IsPrimary              bool
	CommissionPercent      decimal.Decimal
	CommissionBase         CommissionBase
	IsLifetime             bool
	CommissionInstallments *int
	CreatedAt              time.Time
}

type ProjectionRow struct {
	Month             time.Time       `json:"month"`
	EnterpriseCode    int64           `json:"enterprise_code"`
	CustomerCode      int64           `json:"customer_code"`
	EstablishmentCode *int64          `json:"establishment_code,omitempty"`
	ItemCode          int64           `json:"item_code"`
	ItemMask          *string         `json:"item_mask,omitempty"`
	RecurringSaleCode int64           `json:"recurring_sale_code"`
	Quantity          decimal.Decimal `json:"quantity"`
	UnitValue         decimal.Decimal `json:"unit_value"`
	ProjectedValue    decimal.Decimal `json:"projected_value"`
	AppliedAdjustment bool            `json:"applied_adjustment"`
}

type CommissionProjectionRow struct {
	ProjectionRow
	RepresentativeCode int64           `json:"representative_code"`
	CommissionPercent  decimal.Decimal `json:"commission_percent"`
	CommissionValue    decimal.Decimal `json:"commission_value"`
}

type Operation struct {
	RecurringSaleCode int64
	OperationType     string
	Competence        string
	IdempotencyKey    string
	RequestHash       string
	ResultCode        *int64
	Status            string
	ActorID           uuid.UUID
}
