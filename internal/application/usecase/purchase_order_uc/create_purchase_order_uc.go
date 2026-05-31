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
	// SupplierDefaults is optional. When set, missing header fields (payment
	// term) are defaulted from the supplier's registration. Nil disables it.
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
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

	// Default header fields from the supplier registration when not provided.
	paymentTerm := dto.PaymentTermCode
	priceTable := dto.PriceTableCode
	invoiceType := dto.InvoiceTypeCode
	financialAccount := dto.FinancialAccount
	freightType := dto.FreightType
	if uc.SupplierDefaults != nil && dto.SupplierCode != nil {
		if def, derr := uc.SupplierDefaults.GetPurchasingDefaults(ctx, *dto.SupplierCode, dto.EnterpriseCode); derr == nil && def != nil {
			if paymentTerm == nil {
				paymentTerm = def.PaymentConditionID
			}
			if priceTable == nil {
				priceTable = def.PurchasePriceTableID
			}
			if invoiceType == nil {
				invoiceType = def.DefaultInvoiceTypeID
			}
			if financialAccount == nil {
				financialAccount = def.FinancialAccount
			}
			if freightType == "" {
				freightType = def.FreightType
			}
		}
	}

	o := &entity.PurchaseOrder{
		OrderNumber:         orderNum,
		EnterpriseCode:      dto.EnterpriseCode,
		Status:              status,
		Origin:              origin,
		EmissionDate:        emissionDate,
		SupplierCode:        dto.SupplierCode,
		PaymentTermCode:     paymentTerm,
		CurrencyCode:        currency,
		ShippingAddressCode: dto.ShippingAddressCode,
		Notes:               dto.Notes,
		TotalGross:          dto.TotalGross,
		TotalNet:            dto.TotalNet,
		TotalDiscount:       dto.TotalDiscount,
		IsFirm:              dto.IsFirm,
		CreatedBy:           dto.CreatedBy,
		// extended header
		PriceTableCode:         priceTable,
		InvoiceTypeCode:        invoiceType,
		FinancialAccount:       financialAccount,
		RequestTypeCode:        dto.RequestTypeCode,
		FreightType:            freightType,
		FreightValueType:       dto.FreightValueType,
		FreightValueMode:       dto.FreightValueMode,
		FreightValue:           dto.FreightValue,
		CarrierCode:            dto.CarrierCode,
		RedispatchCarrierCode:  dto.RedispatchCarrierCode,
		RedispatchFreightType:  dto.RedispatchFreightType,
		RedispatchFreightValue: dto.RedispatchFreightValue,
		AdvanceValue:           dto.AdvanceValue,
		IncotermCode:           dto.IncotermCode,
		TalaoNumber:            dto.TalaoNumber,
	}

	if dto.DeliveryDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.DeliveryDate)
		o.DeliveryDate = &t
	}
	o.CurrencyDate = parsePODate(dto.CurrencyDate)
	o.AdvanceDate = parsePODate(dto.AdvanceDate)
	o.ShipmentDate = parsePODate(dto.ShipmentDate)

	return uc.Repo.Create(ctx, o)
}

func parsePODate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}
