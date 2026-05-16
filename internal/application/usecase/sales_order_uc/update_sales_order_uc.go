package sales_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type UpdateSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *UpdateSalesOrderUseCase) Execute(
	ctx context.Context,
	dto request.UpdateSalesOrderDTO,
) (*entity.SalesOrder, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	o := &entity.SalesOrder{
		Code:                dto.Code,
		Status:              entity.SalesOrderStatus(dto.Status),
		Origin:              entity.SalesOrderOrigin(dto.Origin),
		DeliveryDateFirm:    dto.DeliveryDateFirm,
		CustomerCode:        dto.CustomerCode,
		BillingAddressCode:  dto.BillingAddressCode,
		ShippingAddressCode: dto.ShippingAddressCode,
		RepresentativeCode:  dto.RepresentativeCode,
		PlanCode:            dto.PlanCode,
		SalesDivisionCode:   dto.SalesDivisionCode,
		CommissionPct:       dto.CommissionPct,
		TaxTypeCode:         dto.TaxTypeCode,
		PresenceIndicator:   dto.PresenceIndicator,
		SalesChannel:        dto.SalesChannel,
		DefaultNFType:       dto.DefaultNFType,
		PriceTableCode:      dto.PriceTableCode,
		CurrencyCode:        dto.CurrencyCode,
		PaymentTermCode:     dto.PaymentTermCode,
		AdditionalDays:      dto.AdditionalDays,
		BearerCode:          dto.BearerCode,
		Notes:               dto.Notes,
		ObsCustomer:         dto.ObsCustomer,
		IsFirm:              dto.IsFirm,
	}

	if dto.DeliveryDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.DeliveryDate)
		o.DeliveryDate = &t
	}
	if dto.SaleDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.SaleDate)
		o.SaleDate = &t
	}

	return uc.Repo.Update(ctx, o)
}
