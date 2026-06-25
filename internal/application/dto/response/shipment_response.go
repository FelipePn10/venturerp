package response

import (
	"time"

	"github.com/google/uuid"
)

// ShipmentResponse is the API representation of a shipment (romaneio).
type ShipmentResponse struct {
	ID                  int64   `json:"id"`
	Code                int64   `json:"code"`
	ReferenceType       *string `json:"reference_type,omitempty"`
	SalesOrderCode      *int64  `json:"sales_order_code,omitempty"`
	PurchaseOrderCode   *int64  `json:"purchase_order_code,omitempty"`
	ProductionOrderCode *int64  `json:"production_order_code,omitempty"`
	CarrierCode         *int64  `json:"carrier_code,omitempty"`
	Status              string  `json:"status"`

	TotalVolumes     int     `json:"total_volumes"`
	TotalNetWeight   float64 `json:"total_net_weight"`
	TotalGrossWeight float64 `json:"total_gross_weight"`
	TotalCubageM3    float64 `json:"total_cubage_m3"`

	// Transporte / viagem.
	FreightModality   *string    `json:"freight_modality,omitempty"`
	FreightValue      float64    `json:"freight_value"`
	InsuranceValue    float64    `json:"insurance_value"`
	VehiclePlate      *string    `json:"vehicle_plate,omitempty"`
	DriverName        *string    `json:"driver_name,omitempty"`
	DriverDocument    *string    `json:"driver_document,omitempty"`
	ANTTCode          *string    `json:"antt_code,omitempty"`
	Seals             *string    `json:"seals,omitempty"`
	EstimatedDelivery *time.Time `json:"estimated_delivery,omitempty"`

	// Vínculo fiscal.
	FiscalExitID *int64  `json:"fiscal_exit_id,omitempty"`
	NFeNumber    *int64  `json:"nfe_number,omitempty"`
	NFeKey       *string `json:"nfe_key,omitempty"`

	Notes       *string    `json:"notes,omitempty"`
	SeparatedAt *time.Time `json:"separated_at,omitempty"`
	ConferredAt *time.Time `json:"conferred_at,omitempty"`
	ShippedAt   *time.Time `json:"shipped_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   uuid.UUID  `json:"created_by"`

	Items   []ShipmentItemResponse   `json:"items,omitempty"`
	Volumes []ShipmentVolumeResponse `json:"volumes,omitempty"`
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
	HasDivergence      bool      `json:"has_divergence"`
	UnitNetWeight      float64   `json:"unit_net_weight"`
	UnitGrossWeight    float64   `json:"unit_gross_weight"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// ShipmentVolumeResponse is the API representation of one packed volume.
type ShipmentVolumeResponse struct {
	ID           int64     `json:"id"`
	ShipmentID   int64     `json:"shipment_id"`
	VolumeNumber int       `json:"volume_number"`
	PackageType  string    `json:"package_type"`
	NetWeight    float64   `json:"net_weight"`
	GrossWeight  float64   `json:"gross_weight"`
	LengthCm     float64   `json:"length_cm"`
	WidthCm      float64   `json:"width_cm"`
	HeightCm     float64   `json:"height_cm"`
	CubageM3     float64   `json:"cubage_m3"`
	Marking      *string   `json:"marking,omitempty"`
	Contents     *string   `json:"contents,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ShipmentEventResponse is one audit-trail entry of a romaneio.
type ShipmentEventResponse struct {
	ID        int64      `json:"id"`
	Event     string     `json:"event"`
	Note      *string    `json:"note,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
