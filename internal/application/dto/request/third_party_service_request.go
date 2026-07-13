package request

import "time"

type ThirdPartyPriceRuleDTO struct {
	Characteristic string  `json:"characteristic"`
	Answer         *string `json:"answer,omitempty"`
}
type ThirdPartyPriceDTO struct {
	ItemCode         int64                    `json:"item_code"`
	Mask             string                   `json:"mask,omitempty"`
	SupplierCode     int64                    `json:"supplier_code"`
	OperationID      int64                    `json:"operation_id"`
	UOM              string                   `json:"uom"`
	ReferenceDate    time.Time                `json:"reference_date"`
	Preferred        bool                     `json:"preferred"`
	UnitPrice        string                   `json:"unit_price"`
	ConversionFactor *string                  `json:"conversion_factor,omitempty"`
	FreightType      string                   `json:"freight_type"`
	FreightValue     string                   `json:"freight_value"`
	TaxPercent       string                   `json:"tax_percent"`
	Formula          string                   `json:"formula,omitempty"`
	Reason           string                   `json:"reason"`
	Rules            []ThirdPartyPriceRuleDTO `json:"rules"`
}
type ThirdPartyReadjustDTO struct {
	IDs           []int64   `json:"ids"`
	Percent       string    `json:"percent"`
	ReferenceDate time.Time `json:"reference_date"`
	Reason        string    `json:"reason"`
}
type ThirdPartyCopyMoveDTO struct {
	IDs           []int64   `json:"ids"`
	SupplierCode  int64     `json:"supplier_code"`
	OperationID   int64     `json:"operation_id"`
	Move          bool      `json:"move"`
	ReferenceDate time.Time `json:"reference_date"`
	Reason        string    `json:"reason"`
}
type ThirdPartyOrderStatusDTO struct {
	Status                  string `json:"status"`
	PurchaseRequisitionCode *int64 `json:"purchase_requisition_code,omitempty"`
	PurchaseOrderCode       *int64 `json:"purchase_order_code,omitempty"`
}
type ThirdPartyMovementDTO struct {
	MovementType   string    `json:"movement_type"`
	Quantity       string    `json:"quantity"`
	OccurredAt     time.Time `json:"occurred_at"`
	ReferenceType  string    `json:"reference_type,omitempty"`
	ReferenceCode  string    `json:"reference_code,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	IdempotencyKey string    `json:"idempotency_key"`
	WarehouseID    *int64    `json:"warehouse_id,omitempty"`
	Lot            string    `json:"lot,omitempty"`
}
type GlobalUnitConversionDTO struct {
	FromUOM string `json:"from_uom"`
	ToUOM   string `json:"to_uom"`
	Factor  string `json:"factor"`
}
