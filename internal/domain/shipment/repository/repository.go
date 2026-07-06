package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/google/uuid"
)

// ShipmentFilter narrows a romaneio listing. Nil fields are ignored; Limit ≤ 0
// means "no limit".
type ShipmentFilter struct {
	Status      *entity.ShipmentStatus
	CarrierCode *int64
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

type LoadFilter struct {
	Status      *entity.LoadStatus
	CarrierCode *int64
	BoxCode     *string
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

// TransportInput carries the trip/transport data set on a romaneio.
type TransportInput struct {
	CarrierCode       *int64
	FreightModality   *string
	FreightValue      float64
	InsuranceValue    float64
	VehiclePlate      *string
	DriverName        *string
	DriverDocument    *string
	ANTTCode          *string
	Seals             *string
	EstimatedDelivery *time.Time
}

type CreateLoadInput struct {
	Description       *string
	CarrierCode       *int64
	VehiclePlate      *string
	DriverName        *string
	DriverDocument    *string
	RouteCode         *string
	Origin            *string
	Destination       *string
	DispatchBoxCode   *string
	PlannedShipDate   *time.Time
	EstimatedDelivery *time.Time
	Notes             *string
	CreatedBy         uuid.UUID
}

type AddFiscalNoteToLoadInput struct {
	LoadCode     int64
	ShipmentCode *int64
	FiscalExitID int64
	NFeNumber    *int64
	NFeKey       *string
	Sequence     int
}

type LoadMonitorRow struct {
	LoadCode           int64
	Status             entity.LoadStatus
	CarrierCode        *int64
	VehiclePlate       *string
	DriverName         *string
	DispatchBoxCode    *string
	PlannedShipDate    *time.Time
	EstimatedDelivery  *time.Time
	TotalShipments     int
	TotalFiscalNotes   int
	TotalVolumes       int
	TotalNetWeight     float64
	TotalGrossWeight   float64
	TotalCubageM3      float64
	OpenShipments      int
	SeparatedShipments int
	ConferredShipments int
	ShippedShipments   int
}

type SeparationMonitorRow struct {
	ShipmentCode     int64
	LoadCode         *int64
	ShipmentStatus   entity.ShipmentStatus
	LoadStatus       *entity.LoadStatus
	SalesOrderCode   *int64
	CarrierCode      *int64
	DispatchBoxCode  *string
	TotalItems       int
	ConferredItems   int
	DivergentItems   int
	TotalVolumes     int
	TotalGrossWeight float64
}

type LogisticPanelSummary struct {
	PlannedLoads       int
	ReleasedLoads      int
	LoadingLoads       int
	LoadedLoads        int
	ShippedLoads       int
	CancelledLoads     int
	OpenShipments      int
	SeparatedShipments int
	ConferredShipments int
	BoxesOccupied      int
	BoxesAvailable     int
	TotalVolumes       int
	TotalGrossWeight   float64
}

type ShipmentRepository interface {
	NextCode(ctx context.Context) (int64, error)
	Create(ctx context.Context, s *entity.Shipment) (*entity.Shipment, error)
	GetByCode(ctx context.Context, code int64) (*entity.Shipment, error)
	List(ctx context.Context) ([]*entity.Shipment, error)
	ListFiltered(ctx context.Context, f ShipmentFilter) ([]*entity.Shipment, error)
	ListBySalesOrder(ctx context.Context, salesOrderCode int64) ([]*entity.Shipment, error)
	ListByPurchaseOrder(ctx context.Context, purchaseOrderCode int64) ([]*entity.Shipment, error)
	ListByProductionOrder(ctx context.Context, productionOrderCode int64) ([]*entity.Shipment, error)
	ListByReference(ctx context.Context, refType entity.ShipmentReferenceType, refCode int64) ([]*entity.Shipment, error)

	// UpdateStatus moves the romaneio to a new status, stamping the matching
	// transition timestamp and recording an audit event.
	UpdateStatus(ctx context.Context, code int64, status entity.ShipmentStatus, by *uuid.UUID, note string) error
	UpdateTransport(ctx context.Context, code int64, t TransportInput, by *uuid.UUID) error
	SetFiscalExit(ctx context.Context, code int64, fiscalExitID, nfeNumber *int64, nfeKey *string, by *uuid.UUID) error
	RecalcTotals(ctx context.Context, code int64) error

	AddItem(ctx context.Context, item *entity.ShipmentItem) (*entity.ShipmentItem, error)
	ListItems(ctx context.Context, shipmentID int64) ([]*entity.ShipmentItem, error)
	GetItem(ctx context.Context, itemID int64) (*entity.ShipmentItem, error)
	ConferItem(ctx context.Context, itemID int64, conferredQty float64) error

	AddVolume(ctx context.Context, v *entity.ShipmentVolume) (*entity.ShipmentVolume, error)
	ListVolumes(ctx context.Context, shipmentID int64) ([]*entity.ShipmentVolume, error)
	DeleteVolume(ctx context.Context, volumeID int64) error

	AddEvent(ctx context.Context, e *entity.ShipmentEvent) error
	ListEvents(ctx context.Context, shipmentID int64) ([]*entity.ShipmentEvent, error)

	NextLoadCode(ctx context.Context) (int64, error)
	CreateLoad(ctx context.Context, in CreateLoadInput) (*entity.ShipmentLoad, error)
	GetLoadByCode(ctx context.Context, code int64) (*entity.ShipmentLoad, error)
	ListLoads(ctx context.Context, f LoadFilter) ([]*entity.ShipmentLoad, error)
	AddShipmentToLoad(ctx context.Context, loadCode, shipmentCode int64, sequence int) (*entity.ShipmentLoadShipment, error)
	RemoveShipmentFromLoad(ctx context.Context, loadCode, shipmentCode int64) error
	AddFiscalNoteToLoad(ctx context.Context, in AddFiscalNoteToLoadInput) (*entity.ShipmentLoadFiscalNote, error)
	UpdateLoadStatus(ctx context.Context, code int64, status entity.LoadStatus, by *uuid.UUID, note string) error
	RecalcLoadTotals(ctx context.Context, code int64) error
	CreateDeliveryInstruction(ctx context.Context, d *entity.DeliveryInstruction) (*entity.DeliveryInstruction, error)
	ListDeliveryInstructions(ctx context.Context, loadCode *int64, activeOnly bool) ([]*entity.DeliveryInstruction, error)
	CreateDispatchBox(ctx context.Context, b *entity.DispatchBox) (*entity.DispatchBox, error)
	ListDispatchBoxes(ctx context.Context, activeOnly bool) ([]*entity.DispatchBox, error)
	AssignBoxToLoad(ctx context.Context, loadCode int64, boxCode string, by *uuid.UUID) error
	LoadMonitor(ctx context.Context, f LoadFilter) ([]*LoadMonitorRow, error)
	SeparationMonitor(ctx context.Context, f LoadFilter) ([]*SeparationMonitorRow, error)
	LogisticPanel(ctx context.Context) (*LogisticPanelSummary, error)
}
