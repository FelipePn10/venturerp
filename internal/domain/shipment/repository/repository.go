package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
)

type ShipmentRepository interface {
	NextCode(ctx context.Context) (int64, error)
	Create(ctx context.Context, s *entity.Shipment) (*entity.Shipment, error)
	GetByCode(ctx context.Context, code int64) (*entity.Shipment, error)
	List(ctx context.Context) ([]*entity.Shipment, error)
	ListBySalesOrder(ctx context.Context, salesOrderCode int64) ([]*entity.Shipment, error)
	UpdateStatus(ctx context.Context, code int64, status entity.ShipmentStatus) error

	AddItem(ctx context.Context, item *entity.ShipmentItem) (*entity.ShipmentItem, error)
	ListItems(ctx context.Context, shipmentID int64) ([]*entity.ShipmentItem, error)
	ConferItem(ctx context.Context, itemID int64, conferredQty float64) error
}
