package response

import (
	"encoding/json"
	"time"
)

type ThirdPartyPriceRuleResponse struct {
	ID             int64   `json:"id"`
	Characteristic string  `json:"characteristic"`
	Answer         *string `json:"answer,omitempty"`
}
type ThirdPartyPriceResponse struct {
	ID               int64                         `json:"id"`
	ItemCode         int64                         `json:"item_code"`
	Mask             string                        `json:"mask,omitempty"`
	SupplierCode     int64                         `json:"supplier_code"`
	OperationID      int64                         `json:"operation_id"`
	ItemDescription  string                        `json:"item_description"`
	SupplierName     string                        `json:"supplier_name"`
	OperationName    string                        `json:"operation_name"`
	UOM              string                        `json:"uom"`
	ReferenceDate    time.Time                     `json:"reference_date"`
	Preferred        bool                          `json:"preferred"`
	UnitPrice        string                        `json:"unit_price"`
	ConversionFactor *string                       `json:"conversion_factor,omitempty"`
	FreightType      string                        `json:"freight_type"`
	FreightValue     string                        `json:"freight_value"`
	TaxPercent       string                        `json:"tax_percent"`
	Formula          string                        `json:"formula,omitempty"`
	IsActive         bool                          `json:"is_active"`
	Rules            []ThirdPartyPriceRuleResponse `json:"rules"`
}
type ThirdPartyHistoryResponse struct {
	ID        int64           `json:"id"`
	PriceID   int64           `json:"price_id"`
	Action    string          `json:"action"`
	Reason    string          `json:"reason"`
	Snapshot  json.RawMessage `json:"snapshot"`
	ChangedBy string          `json:"changed_by"`
	ChangedAt time.Time       `json:"changed_at"`
}
type ThirdPartyOrderResponse struct {
	ID                      int64                            `json:"id"`
	Code                    int64                            `json:"code"`
	PlannedSuggestionCode   *int64                           `json:"planned_suggestion_code,omitempty"`
	PlanCode                *int64                           `json:"plan_code,omitempty"`
	ProductionOrderID       int64                            `json:"production_order_id"`
	RouteOperationID        int64                            `json:"route_operation_id"`
	OperationID             int64                            `json:"operation_id"`
	ItemCode                int64                            `json:"item_code"`
	ItemDescription         string                           `json:"item_description"`
	SupplierName            string                           `json:"supplier_name"`
	OperationName           string                           `json:"operation_name"`
	Mask                    string                           `json:"mask,omitempty"`
	SupplierCode            *int64                           `json:"supplier_code,omitempty"`
	ServiceItemCode         *int64                           `json:"service_item_code,omitempty"`
	UOM                     string                           `json:"uom"`
	Quantity                string                           `json:"quantity"`
	FulfilledQuantity       string                           `json:"fulfilled_quantity"`
	PendingQuantity         string                           `json:"pending_quantity"`
	StartDate               time.Time                        `json:"start_date"`
	DueDate                 time.Time                        `json:"due_date"`
	Status                  string                           `json:"status"`
	PurchaseRequisitionCode *int64                           `json:"purchase_requisition_code,omitempty"`
	PurchaseOrderCode       *int64                           `json:"purchase_order_code,omitempty"`
	RemittanceType          string                           `json:"remittance_type"`
	Kanban                  bool                             `json:"kanban"`
	Notes                   string                           `json:"notes,omitempty"`
	History                 []ThirdPartyOrderHistoryResponse `json:"history,omitempty"`
}
type ThirdPartyMovementResponse struct {
	ID             int64     `json:"id"`
	ServiceOrderID int64     `json:"service_order_id"`
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
type ThirdPartyCostResponse struct {
	Mode              string `json:"mode"`
	ItemCode          int64  `json:"item_code"`
	OperationID       int64  `json:"operation_id"`
	GrossUnitCost     string `json:"gross_unit_cost"`
	Freight           string `json:"freight"`
	RecoverableTaxes  string `json:"recoverable_taxes"`
	ConversionFactor  string `json:"conversion_factor"`
	EffectiveUnitCost string `json:"effective_unit_cost"`
}
type GlobalUnitConversionResponse struct {
	ID       int64  `json:"id"`
	FromUOM  string `json:"from_uom"`
	ToUOM    string `json:"to_uom"`
	Factor   string `json:"factor"`
	IsActive bool   `json:"is_active"`
}
type ThirdPartyOrderHistoryResponse struct {
	ID             int64     `json:"id"`
	ServiceOrderID int64     `json:"service_order_id"`
	EventType      string    `json:"event_type"`
	PreviousStatus *string   `json:"previous_status,omitempty"`
	NewStatus      *string   `json:"new_status,omitempty"`
	Quantity       *string   `json:"quantity,omitempty"`
	ReferenceType  string    `json:"reference_type,omitempty"`
	ReferenceCode  string    `json:"reference_code,omitempty"`
	ActorID        string    `json:"actor_id"`
	OccurredAt     time.Time `json:"occurred_at"`
}
