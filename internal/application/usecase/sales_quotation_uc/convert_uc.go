package sales_quotation_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type ConvertUseCase struct {
	Quotes *UseCase
	Orders orderrepo.SalesOrderRepository
}

func (uc *ConvertUseCase) Execute(ctx context.Context, dto request.ConvertSalesQuotationDTO) (*response.SalesOrderResponse, error) {
	if !uc.Quotes.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	q, err := uc.Quotes.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	if q.ConvertedSalesOrderCode != nil {
		return nil, errorsuc.NewValidationError("quotation already converted")
	}
	if q.Status == quoteentity.SalesQuotationStatusCancelled || q.Status == quoteentity.SalesQuotationStatusExpired {
		return nil, errorsuc.NewValidationError("cancelled or expired quotation cannot be converted")
	}
	if q.Status == quoteentity.SalesQuotationStatusAttended {
		return nil, errorsuc.NewValidationError("attended quotation cannot be converted again")
	}
	if q.QuotationType == quoteentity.SalesQuotationTypeConsult {
		return nil, errorsuc.NewValidationError("consultation quotation cannot be converted to sales order")
	}
	if q.CommercialBlocked || q.ReleaseStatus == quoteentity.SalesQuotationReleaseBlocked {
		return nil, errorsuc.NewValidationError("commercially blocked quotation cannot be converted")
	}
	items, err := uc.Quotes.Repo.ListItems(ctx, q.Code)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errorsuc.NewValidationError("quotation has no items")
	}
	orderNumber, err := uc.Orders.NextOrderNumber(ctx, q.EnterpriseCode)
	if err != nil {
		return nil, err
	}
	status := orderentity.SalesOrderStatusOrder
	if dto.Status != "" {
		status = orderentity.SalesOrderStatus(dto.Status)
	}
	origin := orderentity.SalesOrderOriginNormal
	if dto.Origin != "" {
		origin = orderentity.SalesOrderOrigin(dto.Origin)
	}
	createdBy := dto.CreatedBy
	if createdBy == [16]byte{} {
		createdBy = q.CreatedBy
	}
	order := &orderentity.SalesOrder{
		OrderNumber:         orderNumber,
		EnterpriseCode:      q.EnterpriseCode,
		Status:              status,
		Origin:              origin,
		EmissionDate:        time.Now(),
		DeliveryDate:        q.DeliveryDate,
		DeliveryDateFirm:    q.DeliveryDateFirm,
		DigitDate:           time.Now(),
		CustomerCode:        q.CustomerCode,
		BillingAddressCode:  q.BillingAddressCode,
		ShippingAddressCode: q.ShippingAddressCode,
		RepresentativeCode:  q.RepresentativeCode,
		SalesDivisionCode:   q.SalesDivisionCode,
		CommissionPct:       q.CommissionPct,
		PriceTableCode:      q.PriceTableCode,
		CurrencyCode:        q.CurrencyCode,
		PaymentTermCode:     q.PaymentTermCode,
		IsNFCe:              q.IsNFCe,
		Street:              q.Street,
		StreetNumber:        q.StreetNumber,
		ForeignDocument:     q.ForeignDocument,
		CarrierCode:         q.CarrierCode,
		FreightType:         q.FreightType,
		FreightValue:        q.FreightValue,
		InsuranceValue:      q.InsuranceValue,
		DiscountValue:       q.DiscountValue,
		SurchargeValue:      q.SurchargeValue,
		TotalGross:          q.TotalGross,
		TotalNet:            q.TotalNet,
		Notes:               q.Notes,
		ObsCustomer:         q.ObsCustomer,
		CreatedBy:           createdBy,
	}
	created, err := uc.Orders.Create(ctx, order)
	if err != nil {
		return nil, err
	}
	for _, quoteItem := range items {
		if !quoteItem.IsActive || quoteItem.Status == quoteentity.SalesQuotationItemStatusCancelled {
			continue
		}
		balance := quoteItem.RequestedQty - quoteItem.AttendedQty - quoteItem.CancelledQty
		if balance <= 0 {
			continue
		}
		orderItem := &orderentity.SalesOrderItem{
			SalesOrderCode:   created.Code,
			Sequence:         quoteItem.Sequence,
			ItemCode:         quoteItem.ItemCode,
			Mask:             quoteItem.Mask,
			DigitDate:        time.Now(),
			SalesUOM:         quoteItem.SalesUOM,
			WarehouseCode:    quoteItem.WarehouseCode,
			PriceTableCode:   quoteItem.PriceTableCode,
			RequestedQty:     balance,
			UnitPrice:        quoteItem.UnitPrice,
			DeliveryDate:     quoteItem.DeliveryDate,
			DeliveryDateFirm: quoteItem.DeliveryDateFirm,
			IPIPct:           quoteItem.IPIPct,
			STPct:            quoteItem.STPct,
			DiscountPct:      quoteItem.DiscountPct,
			TotalGross:       quoteItem.TotalGross,
			TotalNet:         quoteItem.TotalNet,
			TotalNetWithIPI:  quoteItem.TotalNetWithIPI,
			Status:           orderentity.SalesOrderItemStatusOpen,
			Notes:            quoteItem.Notes,
		}
		_, _ = uc.Orders.CreateItem(ctx, orderItem)
	}
	if err := uc.Quotes.Repo.MarkConverted(ctx, q.Code, created.Code); err != nil {
		return nil, err
	}
	return &response.SalesOrderResponse{Code: created.Code, OrderNumber: created.OrderNumber, EnterpriseCode: created.EnterpriseCode, Status: string(created.Status)}, nil
}
