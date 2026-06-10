package shipment_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/google/uuid"
)

// ShipmentUseCase handles outbound logistics: creating a dispatch note (romaneio),
// adding lines, confirming separation (conferência) and shipping.
type ShipmentUseCase struct {
	Repo repository.ShipmentRepository
}

type CreateShipmentInput struct {
	SalesOrderCode *int64
	CarrierCode    *int64
	TotalVolumes   int
	TotalWeight    float64
	Notes          *string
	CreatedBy      uuid.UUID
}

func (uc *ShipmentUseCase) Create(ctx context.Context, in CreateShipmentInput) (*response.ShipmentResponse, error) {
	code, err := uc.Repo.NextCode(ctx)
	if err != nil {
		return nil, err
	}
	s := &entity.Shipment{
		Code:           code,
		SalesOrderCode: in.SalesOrderCode,
		CarrierCode:    in.CarrierCode,
		Status:         entity.ShipmentStatusOpen,
		TotalVolumes:   in.TotalVolumes,
		TotalWeight:    in.TotalWeight,
		Notes:          in.Notes,
		CreatedBy:      in.CreatedBy,
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
	}
	item.Notes = in.Notes
	created, err := uc.Repo.AddItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return toShipmentItemResponse(created), nil
}

func (uc *ShipmentUseCase) Get(ctx context.Context, code int64) (*response.ShipmentResponse, error) {
	s, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toShipmentResponse(s), nil
}

func (uc *ShipmentUseCase) List(ctx context.Context) ([]*response.ShipmentResponse, error) {
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toShipmentResponses(list), nil
}

func (uc *ShipmentUseCase) ConferItem(ctx context.Context, itemID int64, conferredQty float64) error {
	return uc.Repo.ConferItem(ctx, itemID, conferredQty)
}

// Confer marks the whole shipment as conferred (after separation/checking).
func (uc *ShipmentUseCase) Confer(ctx context.Context, code int64) error {
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusConferred)
}

// Ship marks the shipment as dispatched. Requires every line to be conferred.
func (uc *ShipmentUseCase) Ship(ctx context.Context, code int64) error {
	ship, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	for _, it := range ship.Items {
		if !it.IsConferred {
			return fmt.Errorf("item %d (seq %d) ainda não conferido", it.ItemCode, it.Sequence)
		}
	}
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusShipped)
}

func (uc *ShipmentUseCase) Cancel(ctx context.Context, code int64) error {
	return uc.Repo.UpdateStatus(ctx, code, entity.ShipmentStatusCancelled)
}
