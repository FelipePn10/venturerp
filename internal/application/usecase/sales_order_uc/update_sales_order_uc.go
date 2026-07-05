package sales_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type UpdateSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *UpdateSalesOrderUseCase) Execute(
	ctx context.Context,
	dto request.UpdateSalesOrderDTO,
) (*response.SalesOrderResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	o := &entity.SalesOrder{
		Code:                        dto.Code,
		Status:                      entity.SalesOrderStatus(dto.Status),
		Origin:                      entity.SalesOrderOrigin(dto.Origin),
		DeliveryDateFirm:            dto.DeliveryDateFirm,
		CustomerCode:                dto.CustomerCode,
		BillingAddressCode:          dto.BillingAddressCode,
		ShippingAddressCode:         dto.ShippingAddressCode,
		RepresentativeCode:          dto.RepresentativeCode,
		PlanCode:                    dto.PlanCode,
		SalesDivisionCode:           dto.SalesDivisionCode,
		CommissionPct:               dto.CommissionPct,
		TaxTypeCode:                 dto.TaxTypeCode,
		PresenceIndicator:           dto.PresenceIndicator,
		SalesChannel:                dto.SalesChannel,
		DefaultNFType:               dto.DefaultNFType,
		PriceTableCode:              dto.PriceTableCode,
		CurrencyCode:                dto.CurrencyCode,
		PaymentTermCode:             dto.PaymentTermCode,
		AdditionalDays:              dto.AdditionalDays,
		BearerCode:                  dto.BearerCode,
		Notes:                       dto.Notes,
		ObsCustomer:                 dto.ObsCustomer,
		IsFirm:                      dto.IsFirm,
		RepresentativeOrderNumber:   dto.RepresentativeOrderNumber,
		IsNFCe:                      dto.IsNFCe,
		Street:                      dto.Street,
		StreetNumber:                dto.StreetNumber,
		ForeignDocument:             dto.ForeignDocument,
		CollectionEstablishmentCode: dto.CollectionEstablishmentCode,
		NFTypeDescription:           dto.NFTypeDescription,
		CarrierCode:                 dto.CarrierCode,
		FreightType:                 dto.FreightType,
		FreightValue:                dto.FreightValue,
		InsuranceValue:              dto.InsuranceValue,
		VolumeQuantity:              dto.VolumeQuantity,
		VolumeType:                  dto.VolumeType,
		NetWeight:                   dto.NetWeight,
		GrossWeight:                 dto.GrossWeight,
		DiscountValue:               dto.DiscountValue,
		SurchargeValue:              dto.SurchargeValue,
		ProjectCode:                 dto.ProjectCode,
		ProjectName:                 dto.ProjectName,
	}

	o.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)
	o.SaleDate = datetime.ParseDatePtr(dto.SaleDate)

	updated, err := uc.Repo.Update(ctx, o)
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponse(updated), nil
}
