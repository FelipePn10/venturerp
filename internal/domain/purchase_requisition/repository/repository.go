package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
)

type PurchaseRequisitionRepository interface {
	Create(ctx context.Context, r *entity.PurchaseRequisition) (*entity.PurchaseRequisition, error)
	GetByCode(ctx context.Context, code int64) (*entity.PurchaseRequisition, error)
	List(ctx context.Context, onlyOpen bool) ([]*entity.PurchaseRequisition, error)
	NextCode(ctx context.Context) (int64, error)

	AddItem(ctx context.Context, item *entity.PurchaseRequisitionItem) (*entity.PurchaseRequisitionItem, error)
	ListItems(ctx context.Context, requisitionCode int64) ([]*entity.PurchaseRequisitionItem, error)
	GetItem(ctx context.Context, id int64) (*entity.PurchaseRequisitionItem, error)
	// RegisterAttendance increments the attended quantity and recomputes status.
	RegisterAttendance(ctx context.Context, itemID int64, qty float64) (*entity.PurchaseRequisitionItem, error)
	UpdateStatus(ctx context.Context, code int64, status string) error
}
