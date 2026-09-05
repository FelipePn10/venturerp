package sales_quotation_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/salespricing"
	itemtypes "github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/shopspring/decimal"
)

func (uc *UseCase) CreateItem(ctx context.Context, dto request.CreateSalesQuotationItemDTO) (*response.SalesQuotationItemResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SalesQuotationCode == 0 {
		return nil, errorsuc.NewValidationError("informe o orçamento de venda")
	}
	resolvedItem, err := itemresolution.Resolve(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	itemCode := int64(resolvedItem.Code)
	if err = itemresolution.ValidateMask(ctx, uc.Items, itemCode, dto.Mask); err != nil {
		return nil, err
	}
	if !dto.RequestedQty.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade solicitada deve ser maior que zero")
	}
	item := &entity.SalesQuotationItem{
		SalesQuotationCode: dto.SalesQuotationCode,
		Sequence:           dto.Sequence,
		ItemCode:           itemCode,
		Mask:               dto.Mask,
		SalesUOM:           dto.SalesUOM,
		WarehouseCode:      dto.WarehouseCode,
		PriceTableCode:     dto.PriceTableCode,
		RequestedQty:       dto.RequestedQty,
		UnitPrice:          dto.UnitPrice,
		DeliveryDate:       datetime.ParseDatePtr(dto.DeliveryDate),
		DeliveryDateFirm:   dto.DeliveryDateFirm,
		DiscountPct:        dto.DiscountPct,
		IPIPct:             dto.IPIPct,
		STPct:              dto.STPct,
		Status:             entity.SalesQuotationItemStatusOpen,
		Notes:              dto.Notes,
	}
	quotation, err := uc.Repo.GetByCode(ctx, dto.SalesQuotationCode)
	if err != nil {
		return nil, err
	}
	tableCode := quotation.PriceTableCode
	if dto.PriceTableCode != nil {
		tableCode = dto.PriceTableCode
	}
	if tableCode == nil {
		return nil, errorsuc.NewValidationError("o orçamento não possui tabela de preço")
	}
	pricing, err := salespricing.Resolve(ctx, uc.Customers, *tableCode, itemCode, dto.RequestedQty.InexactFloat64())
	if err != nil {
		return nil, err
	}
	item.UnitPrice = decimal.NewFromFloat(pricing.AppliedPrice)
	item.PriceTableCode = tableCode
	if quotation.IsNFCe && quotation.DeliveryWithReceipt {
		item.IPIPct = decimal.Zero
	}
	if err := uc.validateNFCeServiceItem(ctx, quotation, itemCode); err != nil {
		return nil, err
	}
	calcItemTotals(item)
	created, err := uc.Repo.CreateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.RecalculateTotals(ctx, dto.SalesQuotationCode); err != nil {
		return nil, err
	}
	if err := uc.applyCommercialPolicies(ctx, dto.SalesQuotationCode); err != nil {
		return nil, err
	}
	return toItemResponse(created), nil
}

func (uc *UseCase) UpdateItem(ctx context.Context, dto request.UpdateSalesQuotationItemDTO) (*response.SalesQuotationItemResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !dto.RequestedQty.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade solicitada deve ser maior que zero")
	}
	if dto.AttendedQty.IsNegative() || dto.CancelledQty.IsNegative() {
		return nil, errorsuc.NewValidationError("as quantidades atendida e cancelada devem ser maiores ou iguais a zero")
	}
	if dto.AttendedQty.Add(dto.CancelledQty).GreaterThan(dto.RequestedQty) {
		return nil, errorsuc.NewValidationError("a soma das quantidades atendida e cancelada não pode passar da solicitada")
	}
	current, err := uc.Repo.GetItem(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	if dto.ItemCode != nil {
		resolvedItem, resolveErr := itemresolution.Resolve(ctx, uc.Items, *dto.ItemCode)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if int64(resolvedItem.Code) != current.ItemCode {
			return nil, errorsuc.NewValidationError("o item da linha não pode ser alterado; cancele a linha e inclua outro item")
		}
	}
	quotation, err := uc.Repo.GetByCode(ctx, current.SalesQuotationCode)
	if err != nil {
		return nil, err
	}
	tableCode := current.PriceTableCode
	if tableCode == nil {
		tableCode = quotation.PriceTableCode
	}
	if tableCode == nil {
		return nil, errorsuc.NewValidationError("o orçamento não possui tabela de preço")
	}
	pricing, err := salespricing.Resolve(ctx, uc.Customers, *tableCode, current.ItemCode, dto.RequestedQty.InexactFloat64())
	if err != nil {
		return nil, err
	}
	if err := uc.validateNFCeServiceItem(ctx, quotation, current.ItemCode); err != nil {
		return nil, err
	}
	item := &entity.SalesQuotationItem{
		Code:             dto.Code,
		RequestedQty:     dto.RequestedQty,
		UnitPrice:        decimal.NewFromFloat(pricing.AppliedPrice),
		AttendedQty:      dto.AttendedQty,
		CancelledQty:     dto.CancelledQty,
		DeliveryDate:     datetime.ParseDatePtr(dto.DeliveryDate),
		DeliveryDateFirm: dto.DeliveryDateFirm,
		DiscountPct:      dto.DiscountPct,
		IPIPct:           dto.IPIPct,
		STPct:            dto.STPct,
		Notes:            dto.Notes,
	}
	if quotation.IsNFCe && quotation.DeliveryWithReceipt {
		item.IPIPct = decimal.Zero
	}
	calcItemTotals(item)
	balance := dto.RequestedQty.Sub(dto.AttendedQty).Sub(dto.CancelledQty)
	switch {
	case dto.CancelledQty.GreaterThanOrEqual(dto.RequestedQty):
		item.Status = entity.SalesQuotationItemStatusCancelled
	case dto.AttendedQty.GreaterThanOrEqual(dto.RequestedQty):
		item.Status = entity.SalesQuotationItemStatusDelivered
	case dto.AttendedQty.IsPositive() || balance.LessThan(dto.RequestedQty):
		item.Status = entity.SalesQuotationItemStatusPartial
	default:
		item.Status = entity.SalesQuotationItemStatusOpen
	}
	updated, err := uc.Repo.UpdateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	if err := uc.Repo.RecalculateTotals(ctx, updated.SalesQuotationCode); err != nil {
		return nil, err
	}
	if err := uc.applyCommercialPolicies(ctx, updated.SalesQuotationCode); err != nil {
		return nil, err
	}
	return toItemResponse(updated), nil
}

func (uc *UseCase) validateNFCeServiceItem(ctx context.Context, q *entity.SalesQuotation, itemCode int64) error {
	if !q.IsNFCe || uc.Items == nil {
		return nil
	}
	code, err := valueobject.NewItemCode(itemCode)
	if err != nil {
		return err
	}
	item, err := uc.Items.FindItemByCode(ctx, code)
	if err != nil {
		return err
	}
	if item.Engineering.Type != itemtypes.SERVICO {
		return nil
	}
	parameters, err := uc.Repo.GetParameters(ctx)
	if err != nil {
		return err
	}
	if !parameters.AllowServiceItemsNFCe || !q.DeliveryWithReceipt {
		return errorsuc.NewValidationError("itens de serviço na NFC-e exigem o parâmetro 27 e entrega com recibo")
	}
	return nil
}

func (uc *UseCase) ListItems(ctx context.Context, quotationCode int64) ([]*response.SalesQuotationItemResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	items, err := uc.Repo.ListItems(ctx, quotationCode)
	if err != nil {
		return nil, err
	}
	return toItemResponses(items), nil
}

func (uc *UseCase) CancelItem(ctx context.Context, dto request.CancelSalesQuotationItemDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	reason, err := uc.Repo.GetCancellationReason(ctx, dto.ReasonCode)
	if err != nil {
		return err
	}
	if reason.RequireComplement && (dto.Complement == nil || strings.TrimSpace(*dto.Complement) == "") {
		return errorsuc.NewValidationError("o motivo de cancelamento escolhido exige um complemento")
	}
	return uc.Repo.CancelItem(ctx, dto.Code, reason.Code, reason.Description, dto.Complement)
}

func calcItemTotals(item *entity.SalesQuotationItem) {
	gross := item.UnitPrice.Mul(item.RequestedQty)
	discount := gross.Mul(item.DiscountPct).Div(decimal.NewFromInt(100))
	item.TotalGross = gross
	item.TotalNet = gross.Sub(discount)
	item.TotalNetWithIPI = item.TotalNet.Add(item.TotalNet.Mul(item.IPIPct).Div(decimal.NewFromInt(100))).Add(item.TotalNet.Mul(item.STPct).Div(decimal.NewFromInt(100)))
	item.Balance = item.RequestedQty.Sub(item.AttendedQty).Sub(item.CancelledQty)
}
