package sales_quotation_uc

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type ConvertUseCase struct {
	Quotes *UseCase
	UOW    ports.SalesQuotationConversionUnitOfWork
}

func (uc *ConvertUseCase) Execute(ctx context.Context, dto request.ConvertSalesQuotationDTO) (*response.SalesOrderResponse, error) {
	if !uc.Quotes.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	createdBy, err := uc.Quotes.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	q, err := uc.Quotes.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	if err := representativevalidation.Validate(ctx, uc.Quotes.Representatives, q.RepresentativeCode); err != nil {
		return nil, err
	}
	if q.ConvertedSalesOrderCode != nil {
		return nil, errorsuc.NewValidationError("o orçamento já foi convertido")
	}
	if q.Status == quoteentity.SalesQuotationStatusCancelled || q.Status == quoteentity.SalesQuotationStatusExpired {
		return nil, errorsuc.NewValidationError("orçamento cancelado ou expirado não pode ser convertido")
	}
	if q.Status == quoteentity.SalesQuotationStatusAttended {
		return nil, errorsuc.NewValidationError("orçamento atendido não pode ser convertido novamente")
	}
	if q.QuotationType == quoteentity.SalesQuotationTypeConsult {
		return nil, errorsuc.NewValidationError("orçamento de consulta não pode ser convertido em pedido")
	}
	if q.CommercialBlocked || q.ReleaseStatus == quoteentity.SalesQuotationReleaseBlocked {
		return nil, errorsuc.NewValidationError("orçamento bloqueado comercialmente não pode ser convertido")
	}
	items, err := uc.Quotes.Repo.ListItems(ctx, q.Code)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errorsuc.NewValidationError("o orçamento não possui itens para conversão")
	}
	if uc.UOW == nil {
		return nil, errorsuc.NewValidationError("a conversão do orçamento não está configurada")
	}
	status := orderentity.SalesOrderStatusOrder
	if dto.Status != "" {
		status = orderentity.SalesOrderStatus(dto.Status)
	}
	origin := orderentity.SalesOrderOriginNormal
	if dto.Origin != "" {
		origin = orderentity.SalesOrderOrigin(dto.Origin)
	}
	created, err := uc.UOW.Execute(ctx, q.Code, func(orders orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error) {
		orderNumber, err := orders.NextOrderNumber(ctx, q.EnterpriseCode)
		if err != nil {
			return nil, err
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
			CommissionPct:       q.CommissionPct.InexactFloat64(),
			PriceTableCode:      q.PriceTableCode,
			CurrencyCode:        q.CurrencyCode,
			PaymentTermCode:     q.PaymentTermCode,
			IsNFCe:              q.IsNFCe,
			Street:              q.Street,
			StreetNumber:        q.StreetNumber,
			ForeignDocument:     q.ForeignDocument,
			CarrierCode:         q.CarrierCode,
			FreightType:         q.FreightType,
			FreightValue:        q.FreightValue.InexactFloat64(),
			InsuranceValue:      q.InsuranceValue.InexactFloat64(),
			DiscountValue:       q.DiscountValue.InexactFloat64(),
			SurchargeValue:      q.SurchargeValue.InexactFloat64(),
			// No pedido, total_gross e o valor dos produtos JA com desconto de
			// item (o orcamento guarda o bruto antes do desconto). Mandar o
			// bruto do orcamento inflava o pedido convertido.
			//
			// O valor nasce com o total do orcamento e e corrigido depois dos
			// itens: quando parte do orcamento ja foi atendida ou cancelada, o
			// pedido leva so o saldo, e a capa tem de fechar com a soma das
			// linhas.
			TotalGross:  q.TotalNet.InexactFloat64(),
			TotalNet:    q.TotalNet.InexactFloat64(),
			Notes:       q.Notes,
			ObsCustomer: q.ObsCustomer,
			CreatedBy:   createdBy,
		}
		created, err := orders.Create(ctx, order)
		if err != nil {
			return nil, err
		}
		somaProdutos, somaIPI, somaST := decimal.Zero, decimal.Zero, decimal.Zero
		for _, quoteItem := range items {
			if !quoteItem.IsActive || quoteItem.Status == quoteentity.SalesQuotationItemStatusCancelled {
				continue
			}
			balance := quoteItem.RequestedQty.Sub(quoteItem.AttendedQty).Sub(quoteItem.CancelledQty)
			if !balance.IsPositive() {
				continue
			}
			// O saldo convertido pode ser menor que o pedido original, então os
			// totais são recalculados na proporção do saldo — copiar os totais
			// do orçamento levaria o valor da quantidade inteira.
			cem := decimal.NewFromInt(100)
			precoLiquido := quoteItem.UnitPrice.Mul(cem.Sub(quoteItem.DiscountPct)).Div(cem)
			valorProdutos := precoLiquido.Mul(balance)
			valorIPI := valorProdutos.Mul(quoteItem.IPIPct).Div(cem)
			valorST := valorProdutos.Mul(quoteItem.STPct).Div(cem)
			orderItem := &orderentity.SalesOrderItem{
				SalesOrderCode:   created.Code,
				Sequence:         quoteItem.Sequence,
				ItemCode:         quoteItem.ItemCode,
				Mask:             quoteItem.Mask,
				DigitDate:        time.Now(),
				SalesUOM:         quoteItem.SalesUOM,
				WarehouseCode:    quoteItem.WarehouseCode,
				PriceTableCode:   quoteItem.PriceTableCode,
				RequestedQty:     balance.InexactFloat64(),
				UnitPrice:        quoteItem.UnitPrice.InexactFloat64(),
				DeliveryDate:     quoteItem.DeliveryDate,
				DeliveryDateFirm: quoteItem.DeliveryDateFirm,
				IPIPct:           quoteItem.IPIPct.InexactFloat64(),
				STPct:            quoteItem.STPct.InexactFloat64(),
				DiscountPct:      quoteItem.DiscountPct.InexactFloat64(),
				TotalGross:       valorProdutos.InexactFloat64(),
				TotalNet:         valorProdutos.InexactFloat64(),
				TotalIPI:         valorIPI.InexactFloat64(),
				TotalST:          valorST.InexactFloat64(),
				TotalNetWithIPI:  valorProdutos.Add(valorIPI).InexactFloat64(),
				Status:           orderentity.SalesOrderItemStatusOpen,
				Notes:            quoteItem.Notes,
			}
			if _, err := orders.CreateItem(ctx, orderItem); err != nil {
				return nil, err
			}
			somaProdutos = somaProdutos.Add(valorProdutos)
			somaIPI = somaIPI.Add(valorIPI)
			somaST = somaST.Add(valorST)
		}

		// A capa fecha com a soma das linhas convertidas. Antes ela levava o
		// total do orcamento inteiro: convertendo o saldo de um orcamento
		// parcialmente atendido, o pedido nascia valendo mais do que os itens
		// que ele tem, e esse numero ia para comissao, credito e faturamento.
		created.TotalGross = somaProdutos.InexactFloat64()
		created.TotalNet = somaProdutos.InexactFloat64()
		created.TotalNetNoST = somaProdutos.Add(somaIPI).InexactFloat64()
		created.TotalWithIPIWithST = somaProdutos.Add(somaIPI).Add(somaST).InexactFloat64()
		atualizado, err := orders.Update(ctx, created)
		if err != nil {
			return nil, err
		}
		return atualizado, nil
	})
	if err != nil {
		return nil, err
	}
	return &response.SalesOrderResponse{Code: created.Code, OrderNumber: created.OrderNumber, EnterpriseCode: created.EnterpriseCode, Status: string(created.Status)}, nil
}
