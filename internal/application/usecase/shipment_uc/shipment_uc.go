package shipment_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/google/uuid"
)

// StockReserver is the narrow slice of the stock repository the romaneio needs:
// it reserves stock on separation (a soft allocation), and releases those
// reservations when the romaneio ships (consumed — the NF-e does the real
// write-down) or is cancelled. Kept narrow so the shared StockRepository
// interface is untouched.
type StockReserver interface {
	CreateReservation(ctx context.Context, r *stockentity.StockReservation) (*stockentity.StockReservation, error)
	ListActiveReservations(ctx context.Context) ([]*stockentity.StockReservation, error)
	CancelReservation(ctx context.Context, id int64) error
	ConsumeReservation(ctx context.Context, id int64) error
}

const reservationRefShipment = "SHIPMENT"

// ShipmentUseCase handles outbound logistics: creating a dispatch note (romaneio),
// adding lines, separating (with stock reservation), conferência, packing into
// volumes and dispatch — a state machine mirroring SAP's outbound delivery.
type ShipmentUseCase struct {
	Repo  repository.ShipmentRepository
	Stock StockReserver // optional; when nil, separation skips stock reservation
}

type CreateShipmentInput struct {
	ReferenceType       *entity.ShipmentReferenceType
	SalesOrderCode      *int64
	PurchaseOrderCode   *int64
	ProductionOrderCode *int64
	CarrierCode         *int64
	TotalVolumes        int
	TotalNetWeight      float64
	TotalGrossWeight    float64
	TotalCubageM3       float64
	Notes               *string
	CreatedBy           uuid.UUID
}

func (uc *ShipmentUseCase) Create(ctx context.Context, in CreateShipmentInput) (*response.ShipmentResponse, error) {
	code, err := uc.Repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	s := &entity.Shipment{
		Code:                code,
		ReferenceType:       in.ReferenceType,
		SalesOrderCode:      in.SalesOrderCode,
		PurchaseOrderCode:   in.PurchaseOrderCode,
		ProductionOrderCode: in.ProductionOrderCode,
		CarrierCode:         in.CarrierCode,
		Status:              entity.ShipmentStatusOpen,
		TotalVolumes:        in.TotalVolumes,
		TotalNetWeight:      in.TotalNetWeight,
		TotalGrossWeight:    in.TotalGrossWeight,
		TotalCubageM3:       in.TotalCubageM3,
		Notes:               in.Notes,
		CreatedBy:           in.CreatedBy,
	}
	created, err := uc.Repo.Create(ctx, s)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(created), nil
}

type AddShipmentItemInput struct {
	ShipmentCode       int64
	Sequence           int
	ItemCode           int64
	SalesOrderItemCode *int64
	WarehouseID        *int64
	Quantity           float64
	UnitNetWeight      float64
	UnitGrossWeight    float64
	Notes              *string
}

func (uc *ShipmentUseCase) AddItem(ctx context.Context, in AddShipmentItemInput) (*response.ShipmentItemResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, in.ShipmentCode)
	if err != nil {
		return nil, err
	}
	if ship.Status != entity.ShipmentStatusOpen && ship.Status != entity.ShipmentStatusSeparated {
		return nil, fmt.Errorf("romaneio %d não aceita itens no status %s", in.ShipmentCode, ship.Status)
	}
	item := &entity.ShipmentItem{
		ShipmentID:         ship.ID,
		Sequence:           in.Sequence,
		ItemCode:           in.ItemCode,
		SalesOrderItemCode: in.SalesOrderItemCode,
		WarehouseID:        in.WarehouseID,
		Quantity:           in.Quantity,
		UnitNetWeight:      in.UnitNetWeight,
		UnitGrossWeight:    in.UnitGrossWeight,
		Notes:              in.Notes,
	}
	created, err := uc.Repo.AddItem(ctx, item)
	if err != nil {
		return nil, err
	}
	_ = uc.Repo.RecalcTotals(ctx, in.ShipmentCode)
	return toShipmentItemResponse(created), nil
}

func (uc *ShipmentUseCase) Get(ctx context.Context, code int64) (*response.ShipmentResponse, error) {
	s, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(s), nil
}

func (uc *ShipmentUseCase) List(ctx context.Context, f repository.ShipmentFilter) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListFiltered(ctx, f)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ListBySalesOrder(ctx context.Context, code int64) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListBySalesOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ListByPurchaseOrder(ctx context.Context, code int64) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListByPurchaseOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ListByProductionOrder(ctx context.Context, code int64) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.ListByProductionOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

// ConferItem records the checked (conferred) quantity of a line. A divergence
// from the planned quantity is preserved (not silently accepted) and surfaces
// when shipping.
func (uc *ShipmentUseCase) ConferItem(ctx context.Context, shipmentCode, itemID int64, conferredQty float64) error {
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return err
	}
	if ship.Status == entity.ShipmentStatusShipped || ship.Status == entity.ShipmentStatusCancelled {
		return fmt.Errorf("romaneio %d não pode ser conferido no status %s", shipmentCode, ship.Status)
	}
	if conferredQty < 0 {
		return fmt.Errorf("quantidade conferida não pode ser negativa")
	}
	return uc.Repo.ConferItem(ctx, itemID, conferredQty)
}

// Separate moves OPEN → SEPARATED, reserving the picked stock (soft allocation).
func (uc *ShipmentUseCase) Separate(ctx context.Context, code int64, actor uuid.UUID) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusSeparated); err != nil {
		return err
	}
	if len(ship.Items) == 0 {
		return fmt.Errorf("romaneio %d não possui itens para separar", code)
	}
	uc.reserveStock(ctx, ship, actor)
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusSeparated, &actor, "separação")
}

// Confer moves SEPARATED → CONFERRED. Every line must already be conferred.
func (uc *ShipmentUseCase) Confer(ctx context.Context, code int64, actor uuid.UUID) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusConferred); err != nil {
		return err
	}
	for _, it := range ship.Items {
		if !it.IsConferred {
			return fmt.Errorf("item %d (seq %d) ainda não conferido", it.ItemCode, it.Sequence)
		}
	}
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusConferred, &actor, "conferência")
}

// Ship moves CONFERRED → SHIPPED. It blocks on quantity divergences unless they
// are explicitly accepted, and releases (consumes) the stock reservations — the
// fiscal NF-e remains responsible for the actual stock write-down.
func (uc *ShipmentUseCase) Ship(ctx context.Context, code int64, actor uuid.UUID, acceptDivergences bool) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusShipped); err != nil {
		return err
	}
	if err := ship.ValidateForShipping(acceptDivergences); err != nil {
		return err
	}
	uc.releaseReservations(ctx, ship.Code, true)
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusShipped, &actor, "despacho")
}

// Cancel cancels a non-shipped romaneio and releases its stock reservations.
func (uc *ShipmentUseCase) Cancel(ctx context.Context, code int64, actor uuid.UUID, reason string) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := guardTransition(ship, entity.ShipmentStatusCancelled); err != nil {
		return err
	}
	uc.releaseReservations(ctx, ship.Code, false)
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusCancelled, &actor, reason)
}

// UpdateTransport sets the trip/transport data (carrier, freight, vehicle,
// driver, seals, estimated delivery).
func (uc *ShipmentUseCase) UpdateTransport(ctx context.Context, code int64, t repository.TransportInput, actor uuid.UUID) (*response.ShipmentResponse, error) {
	if err := uc.Repo.UpdateTransport(ctx, code, t, &actor); err != nil {
		return nil, err
	}
	return uc.Get(ctx, code)
}

// LinkFiscalExit attaches the NF-e (saída fiscal) of the load to the romaneio.
func (uc *ShipmentUseCase) LinkFiscalExit(ctx context.Context, code int64, fiscalExitID, nfeNumber *int64, nfeKey *string, actor uuid.UUID) error {
	return uc.Repo.SetFiscalExit(ctx, code, fiscalExitID, nfeNumber, nfeKey, &actor)
}

type AddVolumeInput struct {
	ShipmentCode int64
	VolumeNumber int
	PackageType  string
	NetWeight    float64
	GrossWeight  float64
	LengthCm     float64
	WidthCm      float64
	HeightCm     float64
	Marking      *string
	Contents     *string
}

func (uc *ShipmentUseCase) AddVolume(ctx context.Context, in AddVolumeInput) (*response.ShipmentVolumeResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, in.ShipmentCode)
	if err != nil {
		return nil, err
	}
	if in.PackageType == "" {
		in.PackageType = "CAIXA"
	}
	v := &entity.ShipmentVolume{
		ShipmentID:   ship.ID,
		VolumeNumber: in.VolumeNumber,
		PackageType:  in.PackageType,
		NetWeight:    in.NetWeight,
		GrossWeight:  in.GrossWeight,
		LengthCm:     in.LengthCm,
		WidthCm:      in.WidthCm,
		HeightCm:     in.HeightCm,
		Marking:      in.Marking,
		Contents:     in.Contents,
	}
	v.CubageM3 = v.Cubage()
	created, err := uc.Repo.AddVolume(ctx, v)
	if err != nil {
		return nil, err
	}
	_ = uc.Repo.RecalcTotals(ctx, in.ShipmentCode)
	return toShipmentVolumeResponse(created), nil
}

func (uc *ShipmentUseCase) ListVolumes(ctx context.Context, shipmentCode int64) ([]*response.ShipmentVolumeResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	return toShipmentVolumeResponses(ship.Volumes), nil
}

func (uc *ShipmentUseCase) DeleteVolume(ctx context.Context, shipmentCode, volumeID int64) error {
	if err := uc.Repo.DeleteVolume(ctx, volumeID); err != nil {
		return err
	}
	return uc.Repo.RecalcTotals(ctx, shipmentCode)
}

func (uc *ShipmentUseCase) ListEvents(ctx context.Context, shipmentCode int64) ([]*response.ShipmentEventResponse, error) {
	ship, err := uc.Repo.GetByCode(ctx, shipmentCode)
	if err != nil {
		return nil, err
	}
	events, err := uc.Repo.ListEvents(ctx, ship.ID)
	if err != nil {
		return nil, err
	}
	return toShipmentEventResponses(events), nil
}

// ---- helpers ----

func guardTransition(ship *entity.Shipment, next entity.ShipmentStatus) error {
	if !ship.Status.CanTransitionTo(next) {
		return fmt.Errorf("transição inválida: romaneio %d está %s e não pode ir para %s",
			ship.Code, ship.Status, next)
	}
	return nil
}

func (uc *ShipmentUseCase) reserveStock(ctx context.Context, ship *entity.Shipment, actor uuid.UUID) {
	if uc.Stock == nil {
		return
	}
	for _, it := range ship.Items {
		wh := int64(0)
		if it.WarehouseID != nil {
			wh = *it.WarehouseID
		}
		qty := it.Quantity
		if qty <= 0 {
			continue
		}
		_, _ = uc.Stock.CreateReservation(ctx, &stockentity.StockReservation{
			ItemCode:          it.ItemCode,
			WarehouseID:       wh,
			Quantity:          qty,
			ReferenceType:     reservationRefShipment,
			ReferenceCode:     ship.Code,
			ReferenceItemCode: it.SalesOrderItemCode,
			ReservationDate:   time.Now(),
			Status:            "ACTIVE",
			CreatedBy:         actor,
		})
	}
}

// releaseReservations consumes (on ship) or cancels (on cancel) every active
// reservation that this romaneio created. Best-effort: stock hiccups never block
// the romaneio transition.
func (uc *ShipmentUseCase) releaseReservations(ctx context.Context, shipmentCode int64, consume bool) {
	if uc.Stock == nil {
		return
	}
	active, err := uc.Stock.ListActiveReservations(ctx)
	if err != nil {
		return
	}
	for _, res := range active {
		if res.ReferenceType != reservationRefShipment || res.ReferenceCode != shipmentCode {
			continue
		}
		if consume {
			_ = uc.Stock.ConsumeReservation(ctx, res.ID)
		} else {
			_ = uc.Stock.CancelReservation(ctx, res.ID)
		}
	}
}
