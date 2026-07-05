package sales_quotation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
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
	if dto.RequestedQty <= 0 {
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
	calcItemTotals(item)
	created, err := uc.Repo.CreateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	_ = uc.Repo.RecalculateTotals(ctx, dto.SalesQuotationCode)
	return toItemResponse(created), nil
}

func (uc *UseCase) UpdateItem(ctx context.Context, dto request.UpdateSalesQuotationItemDTO) (*response.SalesQuotationItemResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.RequestedQty <= 0 {
		return nil, errorsuc.NewValidationError("requested_qty must be greater than zero")
	}
	if dto.AttendedQty < 0 || dto.CancelledQty < 0 {
		return nil, errorsuc.NewValidationError("attended_qty and cancelled_qty must be greater than or equal to zero")
	}
	if dto.AttendedQty+dto.CancelledQty > dto.RequestedQty {
		return nil, errorsuc.NewValidationError("attended_qty plus cancelled_qty cannot exceed requested_qty")
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
	calcItemTotals(item)
	balance := dto.RequestedQty - dto.AttendedQty - dto.CancelledQty
	switch {
	case dto.CancelledQty >= dto.RequestedQty:
		item.Status = entity.SalesQuotationItemStatusCancelled
	case dto.AttendedQty >= dto.RequestedQty:
		item.Status = entity.SalesQuotationItemStatusDelivered
	case dto.AttendedQty > 0 || balance < dto.RequestedQty:
		item.Status = entity.SalesQuotationItemStatusPartial
	default:
		item.Status = entity.SalesQuotationItemStatusOpen
	}
	updated, err := uc.Repo.UpdateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	_ = uc.Repo.RecalculateTotals(ctx, updated.SalesQuotationCode)
	return toItemResponse(updated), nil
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

func (uc *UseCase) CancelItem(ctx context.Context, itemCode int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.CancelItem(ctx, itemCode)
}

func calcItemTotals(item *entity.SalesQuotationItem) {
	gross := item.UnitPrice * item.RequestedQty
	discount := gross * item.DiscountPct / 100
	item.TotalGross = gross
	item.TotalNet = gross - discount
	item.TotalNetWithIPI = item.TotalNet + item.TotalNet*item.IPIPct/100 + item.TotalNet*item.STPct/100
	item.Balance = item.RequestedQty - item.AttendedQty - item.CancelledQty
}
