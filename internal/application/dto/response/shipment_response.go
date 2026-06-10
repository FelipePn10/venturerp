package response

import (
	"time"

	"github.com/google/uuid"
)

// ShipmentResponse is the API representation of a shipment (romaneio).
type ShipmentResponse struct {
	ID             int64                  `json:"id"`
	Code           int64                  `json:"code"`
	SalesOrderCode *int64                 `json:"sales_order_code,omitempty"`
	CarrierCode    *int64                 `json:"carrier_code,omitempty"`
	Status         string                 `json:"status"`
	TotalVolumes   int                    `json:"total_volumes"`
	TotalWeight    float64                `json:"total_weight"`
	Notes          *string                `json:"notes,omitempty"`
	ShippedAt      *time.Time             `json:"shipped_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CreatedBy      uuid.UUID              `json:"created_by"`
	Items          []ShipmentItemResponse `json:"items,omitempty"`
}

// ShipmentItemResponse is the API representation of a shipment line.
type ShipmentItemResponse struct {
	ID                 int64     `json:"id"`
	ShipmentID         int64     `json:"shipment_id"`
	Sequence           int       `json:"sequence"`
	ItemCode           int64     `json:"item_code"`
	SalesOrderItemCode *int64    `json:"sales_order_item_code,omitempty"`
	WarehouseID        *int64    `json:"warehouse_id,omitempty"`
	Quantity           float64   `json:"quantity"`
	ConferredQty       float64   `json:"conferred_qty"`
	IsConferred        bool      `json:"is_conferred"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}
