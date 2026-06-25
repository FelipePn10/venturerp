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
}
