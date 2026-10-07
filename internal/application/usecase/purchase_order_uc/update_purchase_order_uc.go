package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"
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
	// SupplierDefaults confere que o fornecedor é desta empresa (opcional:
	// nil desliga a conferência, como nos testes de unidade).
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
}

// Execute altera a capa de um pedido ainda não aprovado. Antes, o PUT gravava
// a situação que viesse no corpo (um "APPROVED" passava por cima da alçada),
// zerava a origem e os totais e ignorava frete, transportadora e adiantamento.
func (uc *UpdatePurchaseOrderUseCase) Execute(
	ctx context.Context,
	dto request.UpdatePurchaseOrderDTO,
) (*response.PurchaseOrderResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	o, err := uc.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	if !o.IsActive {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", o.Code))
	}
	if err := o.Editavel(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	if s := strings.ToUpper(strings.TrimSpace(dto.Status)); s != "" && entity.PurchaseOrderStatus(s) != o.Status {
		return nil, errorsuc.NewValidationError("a situação do pedido muda por Aprovar, Autorizar alçada ou Cancelar, não pela alteração da capa")
	}
	if dto.SupplierCode == nil || *dto.SupplierCode <= 0 {
		return nil, errorsuc.NewValidationError("fornecedor é obrigatório")
	}
	if uc.SupplierDefaults != nil {
		if _, serr := resolverFornecedorDoTenant(ctx, uc.SupplierDefaults, *dto.SupplierCode, o.EnterpriseCode); serr != nil {
			return nil, serr
		}
	}

	if dto.EmissionDate != "" {
		t, perr := time.Parse("2006-01-02", dto.EmissionDate)
		if perr != nil {
			return nil, errorsuc.NewValidationError("data de emissão inválida; use AAAA-MM-DD")
		}
		o.EmissionDate = t
	}
	o.DeliveryDate = nil
	if dto.DeliveryDate != nil && *dto.DeliveryDate != "" {
		t, perr := time.Parse("2006-01-02", *dto.DeliveryDate)
		if perr != nil {
			return nil, errorsuc.NewValidationError("data de entrega inválida; use AAAA-MM-DD")
		}
		o.DeliveryDate = &t
	}
	for nome, campo := range map[string]*string{"adiantamento": dto.AdvanceDate, "embarque": dto.ShipmentDate} {
		if campo != nil && *campo != "" {
			if _, perr := time.Parse("2006-01-02", *campo); perr != nil {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("data de %s inválida; use AAAA-MM-DD", nome))
			}
		}
	}
	if dto.FreightValue < 0 || dto.AdvanceValue < 0 || dto.RedispatchFreightValue < 0 {
		return nil, errorsuc.NewValidationError("frete e adiantamento não podem ser negativos")
	}

	o.SupplierCode = dto.SupplierCode
	o.PaymentTermCode = dto.PaymentTermCode
	if dto.CurrencyCode != "" {
		o.CurrencyCode = strings.ToUpper(dto.CurrencyCode)
	}
	o.ShippingAddressCode = dto.ShippingAddressCode
	o.Notes = dto.Notes
	o.IsFirm = dto.IsFirm
	o.PriceTableCode = dto.PriceTableCode
	o.InvoiceTypeCode = dto.InvoiceTypeCode
	o.FinancialAccount = dto.FinancialAccount
	if dto.FreightType != "" {
		o.FreightType = dto.FreightType
	}
	o.FreightValueType = dto.FreightValueType
	o.FreightValueMode = dto.FreightValueMode
	o.FreightValue = dto.FreightValue
	o.CarrierCode = dto.CarrierCode
	o.RedispatchCarrierCode = dto.RedispatchCarrierCode
	o.RedispatchFreightType = dto.RedispatchFreightType
	o.RedispatchFreightValue = dto.RedispatchFreightValue
	o.AdvanceDate = parsePODate(dto.AdvanceDate)
	o.AdvanceValue = dto.AdvanceValue
	o.IncotermCode = dto.IncotermCode
	o.ShipmentDate = parsePODate(dto.ShipmentDate)
	o.TalaoNumber = dto.TalaoNumber

	return regravarComTotais(ctx, uc.Repo, o)
}

// regravarComTotais recalcula os totais pelas linhas, devolve à alçada o
// pedido cujo valor pode ter mudado e grava a capa.
func regravarComTotais(ctx context.Context, repo repository.PurchaseOrderRepository, o *entity.PurchaseOrder) (*response.PurchaseOrderResponse, error) {
	items, err := repo.ListItems(ctx, o.Code)
	if err != nil {
		return nil, err
	}
	antes := o.TotalNet
	o.AplicarTotais(entity.CalcularTotais(o, items))
	if o.Status == entity.PurchaseOrderStatusREQUESTED || o.TotalNet != antes {
		o.VoltarParaRascunho()
	}
	updated, err := repo.Update(ctx, o)
	if err != nil {
		return nil, err
	}
	updated.Items = items
	return toPurchaseOrderResponse(updated), nil
}
