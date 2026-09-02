package sales_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CreateSalesOrderUseCase struct {
	Repo            repository.SalesOrderRepository
	Auth            ports.AuthService
	Representatives representativevalidation.Repository
}

func (uc *CreateSalesOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesOrderDTO,
) (*response.SalesOrderResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	enterpriseCode, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if err := representativevalidation.Validate(ctx, uc.Representatives, dto.RepresentativeCode); err != nil {
		return nil, err
	}

	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}

	orderNum, err := uc.Repo.NextOrderNumber(ctx, enterpriseCode)
	if err != nil {
		return nil, err
	}

	// emission_date defaults to today (matching the CURRENT_DATE column default)
	// when omitted or unparseable, instead of silently persisting 0001-01-01.
	emissionDate := datetime.ParseDateOrDefault(dto.EmissionDate, time.Now())

	status := entity.SalesOrderStatusDraft
	if dto.Status != "" {
		status = entity.SalesOrderStatus(dto.Status)
	}
	origin := entity.SalesOrderOriginNormal
	if dto.Origin != "" {
		origin = entity.SalesOrderOrigin(dto.Origin)
	}
	currency := "BRL"
	if dto.CurrencyCode != "" {
		currency = dto.CurrencyCode
	}

	o := &entity.SalesOrder{
		OrderNumber:                 orderNum,
		EnterpriseCode:              enterpriseCode,
		Status:                      status,
		Origin:                      origin,
		EmissionDate:                emissionDate,
		DeliveryDateFirm:            dto.DeliveryDateFirm,
		DigitDate:                   time.Now(),
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
		CurrencyCode:                currency,
		PaymentTermCode:             dto.PaymentTermCode,
		AdditionalDays:              dto.AdditionalDays,
		BearerCode:                  dto.BearerCode,
		Notes:                       dto.Notes,
		ObsCustomer:                 dto.ObsCustomer,
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
		CreatedBy:                   actor,
	}

	o.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)
	o.SaleDate = datetime.ParseDatePtr(dto.SaleDate)

	created, err := uc.Repo.Create(ctx, o)
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponse(created), nil
}
