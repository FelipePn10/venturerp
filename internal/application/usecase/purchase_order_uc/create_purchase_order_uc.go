package purchase_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type CreatePurchaseOrderUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *CreatePurchaseOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreatePurchaseOrderDTO,
) (*entity.PurchaseOrder, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	orderNum, err := uc.Repo.NextOrderNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}

	emissionDate, _ := time.Parse("2006-01-02", dto.EmissionDate)

	status := entity.PurchaseOrderStatusDRAFT
	if dto.Status != "" {
		status = entity.PurchaseOrderStatus(dto.Status)
	}
	origin := entity.PurchaseOrderOriginNORMAL
	if dto.Origin != "" {
		origin = entity.PurchaseOrderOrigin(dto.Origin)
	}
	currency := "BRL"
	if dto.CurrencyCode != "" {
		currency = dto.CurrencyCode
	}

	o := &entity.PurchaseOrder{
		OrderNumber:         orderNum,
		EnterpriseCode:      dto.EnterpriseCode,
		Status:              status,
		Origin:              origin,
		EmissionDate:        emissionDate,
		SupplierCode:        dto.SupplierCode,
		PaymentTermCode:     dto.PaymentTermCode,
		CurrencyCode:        currency,
		ShippingAddressCode: dto.ShippingAddressCode,
		Notes:               dto.Notes,
		TotalGross:          dto.TotalGross,
		TotalNet:            dto.TotalNet,
		TotalDiscount:       dto.TotalDiscount,
		IsFirm:              dto.IsFirm,
		CreatedBy:           dto.CreatedBy,
	}

	if dto.DeliveryDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.DeliveryDate)
		o.DeliveryDate = &t
	}

	return uc.Repo.Create(ctx, o)
}
