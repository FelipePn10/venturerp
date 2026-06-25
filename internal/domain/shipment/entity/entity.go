package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	ShipmentStatusOpen      ShipmentStatus = "OPEN"
	ShipmentStatusSeparated ShipmentStatus = "SEPARATED"
	ShipmentStatusConferred ShipmentStatus = "CONFERRED"
	ShipmentStatusShipped   ShipmentStatus = "SHIPPED"
	ShipmentStatusCancelled ShipmentStatus = "CANCELLED"
)

// allowedTransitions is the romaneio state machine. A romaneio flows
// OPEN → SEPARATED → CONFERRED → SHIPPED; it can be cancelled from any state
// before it is shipped. SHIPPED and CANCELLED are terminal.
var allowedTransitions = map[ShipmentStatus][]ShipmentStatus{
	ShipmentStatusOpen:      {ShipmentStatusSeparated, ShipmentStatusCancelled},
	ShipmentStatusSeparated: {ShipmentStatusConferred, ShipmentStatusOpen, ShipmentStatusCancelled},
	ShipmentStatusConferred: {ShipmentStatusShipped, ShipmentStatusSeparated, ShipmentStatusCancelled},
	ShipmentStatusShipped:   {},
	ShipmentStatusCancelled: {},
}

// CanTransitionTo reports whether moving from the current status to next is a
// legal romaneio transition.
func (s ShipmentStatus) CanTransitionTo(next ShipmentStatus) bool {
	for _, allowed := range allowedTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

type ShipmentReferenceType string

const (
	ShipmentRefSalesOrder      ShipmentReferenceType = "SALES_ORDER"
	ShipmentRefPurchaseOrder   ShipmentReferenceType = "PURCHASE_ORDER"
	ShipmentRefProductionOrder ShipmentReferenceType = "PRODUCTION_ORDER"
)

// Freight modality (responsável pelo frete).
const (
	FreightCIF       = "CIF"       // emitente paga
	FreightFOB       = "FOB"       // destinatário paga
	FreightThirdParty = "TERCEIROS" // terceiros
	FreightNone      = "SEM_FRETE"
)

// Shipment is a dispatch note (romaneio de carregamento / expedição) that
// supports polymorphic references to sales orders, purchase orders, and
// production orders. It models the physical outbound logistics (separation,
// conferência, packing into volumes and dispatch) — the fiscal stock write-down
// stays on the NF-e (saída fiscal); the romaneio only reserves stock.
type Shipment struct {
	ID                  int64
	Code                int64
	ReferenceType       *ShipmentReferenceType
	SalesOrderCode      *int64
	PurchaseOrderCode   *int64
	ProductionOrderCode *int64
	CarrierCode         *int64
	Status              ShipmentStatus
	TotalVolumes        int

	// Pesos e cubagem totais (líquido e bruto são distintos).
	TotalNetWeight   float64
	TotalGrossWeight float64
	TotalCubageM3    float64

	// Dados da viagem / transporte.
	FreightModality   *string
	FreightValue      float64
	InsuranceValue    float64
	VehiclePlate      *string
	DriverName        *string
	DriverDocument    *string
	ANTTCode          *string
	Seals             *string
	EstimatedDelivery *time.Time

	// Vínculo com a NF-e (saída fiscal) deste carregamento.
	FiscalExitID *int64
	NFeNumber    *int64
	NFeKey       *string

	Notes       *string
	SeparatedAt *time.Time
	ConferredAt *time.Time
	ShippedAt   *time.Time
	CancelledAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
	UpdatedBy   *uuid.UUID

	Items   []*ShipmentItem
	Volumes []*ShipmentVolume
}

type ShipmentItem struct {
	ID                 int64
	ShipmentID         int64
	Sequence           int
	ItemCode           int64
	SalesOrderItemCode *int64
	WarehouseID        *int64
	Quantity           float64 // quantidade planejada a expedir
	ConferredQty       float64 // quantidade conferida (real)
	IsConferred        bool
	UnitNetWeight      float64
	UnitGrossWeight    float64
	Notes              *string
	CreatedAt          time.Time
}

// HasDivergence reports whether a conferred line's checked quantity differs from
// the planned quantity (sobra ou falta na separação).
func (it *ShipmentItem) HasDivergence() bool {
	return it.IsConferred && it.ConferredQty != it.Quantity
}

// ShipmentVolume is one physical handling unit (volume) of the load, with its
// packaging, weights and dimensions — the romaneio's packing detail.
type ShipmentVolume struct {
	ID           int64
	ShipmentID   int64
	VolumeNumber int
	PackageType  string
	NetWeight    float64
	GrossWeight  float64
	LengthCm     float64
	WidthCm      float64
	HeightCm     float64
	CubageM3     float64
	Marking      *string
	Contents     *string
	CreatedAt    time.Time
}

// Cubage computes the volume's cubage in m³ from its dimensions (cm), used when
// the caller does not supply one explicitly.
func (v *ShipmentVolume) Cubage() float64 {
	return (v.LengthCm / 100.0) * (v.WidthCm / 100.0) * (v.HeightCm / 100.0)
}

// ShipmentEvent is one audit entry in the romaneio's lifecycle.
type ShipmentEvent struct {
	ID         int64
	ShipmentID int64
	Event      string
	Note       *string
	CreatedBy  *uuid.UUID
	CreatedAt  time.Time
}

// ValidateForShipping checks the romaneio can be dispatched: every line conferred
// and, unless divergences are explicitly accepted, no quantity divergences.
func (s *Shipment) ValidateForShipping(acceptDivergences bool) error {
	if len(s.Items) == 0 {
		return fmt.Errorf("romaneio %d não possui itens", s.Code)
	}
	for _, it := range s.Items {
		if !it.IsConferred {
			return fmt.Errorf("item %d (seq %d) ainda não conferido", it.ItemCode, it.Sequence)
		}
		if it.HasDivergence() && !acceptDivergences {
			return fmt.Errorf("item %d (seq %d) com divergência: planejado %.4f, conferido %.4f",
				it.ItemCode, it.Sequence, it.Quantity, it.ConferredQty)
		}
	}
	return nil
}
