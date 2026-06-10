package purchase_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type UpdatePurchaseOrderUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *UpdatePurchaseOrderUseCase) Execute(
	ctx context.Context,
	dto request.UpdatePurchaseOrderDTO,
) (*response.PurchaseOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	o := &entity.PurchaseOrder{
		Code:                dto.Code,
		Status:              entity.PurchaseOrderStatus(dto.Status),
		Origin:              entity.PurchaseOrderOrigin(dto.Origin),
		SupplierCode:        dto.SupplierCode,
		PaymentTermCode:     dto.PaymentTermCode,
		CurrencyCode:        dto.CurrencyCode,
		ShippingAddressCode: dto.ShippingAddressCode,
		Notes:               dto.Notes,
		TotalGross:          dto.TotalGross,
		TotalNet:            dto.TotalNet,
		TotalDiscount:       dto.TotalDiscount,
		IsFirm:              dto.IsFirm,
	}

	if dto.DeliveryDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.DeliveryDate)
		o.DeliveryDate = &t
	}

	updated, err := uc.Repo.Update(ctx, o)
	if err != nil {
		return nil, err
	}
	return toPurchaseOrderResponse(updated), nil
}
