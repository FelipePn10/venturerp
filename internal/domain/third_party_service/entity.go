package third_party_service

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrNotFound = errors.New("third-party service record not found")

type Price struct {
	ID, EnterpriseID, ItemCode, SupplierCode, OperationID int64
	Mask, UOM, FreightType, Formula                       string
	ItemDescription, SupplierName, OperationName          string
	ReferenceDate                                         time.Time
	Preferred, IsActive                                   bool
	UnitPrice, FreightValue, TaxPercent                   decimal.Decimal
	ConversionFactor                                      *decimal.Decimal
	CreatedBy                                             uuid.UUID
	CreatedAt, UpdatedAt                                  time.Time
	Rules                                                 []PriceRule
}
type PriceRule struct {
	ID             int64
	Characteristic string
	Answer         *string
}
type PriceFilter struct {
	ItemFrom, ItemTo, SupplierFrom, SupplierTo, OperationID *int64
	Mask                                                    *string
	ReferenceDate                                           *time.Time
	Preferred                                               *bool
	PriceType, OrderBy, ItemSearch, SupplierSearch          string
	ClassificationMaskCode                                  *int64
	ClassificationCodes                                     []string
	Limit, Offset                                           int
}
type History struct {
	ID, PriceID    int64
	Action, Reason string
	Snapshot       []byte
	ChangedBy      uuid.UUID
	ChangedAt      time.Time
}

type ServiceOrder struct {
	ID, Code, EnterpriseID, ProductionOrderID, RouteOperationID, OperationID, ItemCode int64
	PlannedSuggestionCode, PlanCode                                                    *int64
	Mask, UOM, Status, RemittanceType, Notes                                           string
	ItemDescription, SupplierName, OperationName                                       string
	SupplierCode, ServiceItemCode, PurchaseRequisitionCode, PurchaseOrderCode          *int64
	Quantity, FulfilledQuantity                                                        decimal.Decimal
	StartDate, DueDate                                                                 time.Time
	Kanban                                                                             bool
	CreatedBy                                                                          uuid.UUID
	CreatedAt, UpdatedAt                                                               time.Time
}
type OrderFilter struct {
	PlanCode                                                                                            *int64
	ItemFrom, ItemTo, ProductionOrderID, ServiceOrderCode, OperationID, SupplierCode, PurchaseOrderCode *int64
	ProductionOrderIDs, ServiceOrderCodes, OperationIDs, SupplierCodes, PurchaseOrderCodes              []int64
	From, To                                                                                            *time.Time
	EmittedFrom, EmittedTo, DeliveryFrom, DeliveryTo                                                    *time.Time
	Statuses                                                                                            []string
	Position, ItemSearch, SupplierSearch, OrderBy                                                       string
	ClassificationMaskCode                                                                              *int64
	ClassificationCodes                                                                                 []string
	OnlyKanban                                                                                          bool
	Limit, Offset                                                                                       int
}
type Movement struct {
	ID, ServiceOrderID                  int64
	MovementType                        string
	Quantity                            decimal.Decimal
	OccurredAt                          time.Time
	ReferenceType, ReferenceCode, Notes string
	IdempotencyKey                      string
	WarehouseID                         *int64
	Lot                                 string
	CreatedBy                           uuid.UUID
}
type GlobalConversion struct {
	ID, EnterpriseID     int64
	FromUOM, ToUOM       string
	Factor               decimal.Decimal
	IsActive             bool
	CreatedBy            uuid.UUID
	CreatedAt, UpdatedAt time.Time
}
type OrderHistory struct {
	ID, ServiceOrderID           int64
	EventType                    string
	PreviousStatus, NewStatus    *string
	Quantity                     *decimal.Decimal
	ReferenceType, ReferenceCode string
	ActorID                      uuid.UUID
	OccurredAt                   time.Time
}

func (p *Price) Validate() error {
	p.Mask = strings.TrimSpace(p.Mask)
	p.UOM = strings.ToUpper(strings.TrimSpace(p.UOM))
	p.FreightType = strings.ToUpper(strings.TrimSpace(p.FreightType))
	if p.ItemCode <= 0 || p.SupplierCode <= 0 || p.OperationID <= 0 || p.UOM == "" || p.ReferenceDate.IsZero() {
		return errors.New("item, supplier, operation, uom and reference_date are required")
	}
	if p.UnitPrice.IsNegative() || p.FreightValue.IsNegative() || p.TaxPercent.IsNegative() || p.TaxPercent.GreaterThan(decimal.NewFromInt(100)) {
		return errors.New("price, freight and tax values are invalid")
	}
	if p.FreightType != "FIXED" && p.FreightType != "PERCENT" {
		return errors.New("freight_type must be FIXED or PERCENT")
	}
	if p.ConversionFactor != nil && !p.ConversionFactor.IsPositive() {
		return errors.New("conversion_factor must be positive")
	}
	return nil
}
func (o *ServiceOrder) Pending() decimal.Decimal {
	v := o.Quantity.Sub(o.FulfilledQuantity)
	if v.IsNegative() {
		return decimal.Zero
	}
	return v
}
