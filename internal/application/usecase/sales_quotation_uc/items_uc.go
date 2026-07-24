package sales_quotation_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
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
		return nil, errorsuc.NewValidationError("sales_quotation_code is required")
	}
	if dto.ItemCode == 0 {
		return nil, errorsuc.NewValidationError("item_code is required")
	}
	if !dto.RequestedQty.IsPositive() {
		return nil, errorsuc.NewValidationError("requested_qty must be greater than zero")
	}
	item := &entity.SalesQuotationItem{
		SalesQuotationCode: dto.SalesQuotationCode,
		Sequence:           dto.Sequence,
		ItemCode:           dto.ItemCode,
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
	if quotation.IsNFCe && quotation.DeliveryWithReceipt {
		item.IPIPct = decimal.Zero
	}
	if err := uc.validateNFCeServiceItem(ctx, quotation, dto.ItemCode); err != nil {
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
		return nil, errorsuc.NewValidationError("requested_qty must be greater than zero")
	}
	if dto.AttendedQty.IsNegative() || dto.CancelledQty.IsNegative() {
		return nil, errorsuc.NewValidationError("attended_qty and cancelled_qty must be greater than or equal to zero")
	}
	if dto.AttendedQty.Add(dto.CancelledQty).GreaterThan(dto.RequestedQty) {
		return nil, errorsuc.NewValidationError("attended_qty plus cancelled_qty cannot exceed requested_qty")
	}
	current, err := uc.Repo.GetItem(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	quotation, err := uc.Repo.GetByCode(ctx, current.SalesQuotationCode)
	if err != nil {
		return nil, err
	}
	if err := uc.validateNFCeServiceItem(ctx, quotation, current.ItemCode); err != nil {
		return nil, err
	}
	item := &entity.SalesQuotationItem{
		Code:             dto.Code,
		RequestedQty:     dto.RequestedQty,
		UnitPrice:        dto.UnitPrice,
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
		return errorsuc.NewValidationError("service items in NFC-e require parameter 27 and delivery_with_receipt")
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
		return errorsuc.NewValidationError("complement is required for the selected cancellation reason")
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
