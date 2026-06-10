package sales_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type CreateSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CreateSalesOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesOrderDTO,
) (*response.SalesOrderResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	orderNum, err := uc.Repo.NextOrderNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}

	emissionDate, _ := time.Parse("2006-01-02", dto.EmissionDate)

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
		OrderNumber:         orderNum,
		EnterpriseCode:      dto.EnterpriseCode,
		Status:              status,
		Origin:              origin,
		EmissionDate:        emissionDate,
		DeliveryDateFirm:    dto.DeliveryDateFirm,
		DigitDate:           time.Now(),
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
		CurrencyCode:        currency,
		PaymentTermCode:     dto.PaymentTermCode,
		AdditionalDays:      dto.AdditionalDays,
		BearerCode:          dto.BearerCode,
		Notes:               dto.Notes,
		ObsCustomer:         dto.ObsCustomer,
		CreatedBy:           dto.CreatedBy,
	}

	if dto.DeliveryDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.DeliveryDate)
		o.DeliveryDate = &t
	}
	if dto.SaleDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.SaleDate)
		o.SaleDate = &t
	}

	created, err := uc.Repo.Create(ctx, o)
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponse(created), nil
}
