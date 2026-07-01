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
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

type CreateSalesOrderItemUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CreateSalesOrderItemUseCase) Execute(
	ctx context.Context,
	dto request.CreateSalesOrderItemDTO,
) (*response.SalesOrderItemResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if dto.SalesOrderCode == 0 {
		return nil, errorsuc.NewValidationError("sales_order_code is required")
	}
	if dto.ItemCode == 0 {
		return nil, errorsuc.NewValidationError("item_code is required")
	}

	digitDate := datetime.ParseDateOrDefault(dto.DigitDate, time.Now())

	item := &entity.SalesOrderItem{
		SalesOrderCode:   dto.SalesOrderCode,
		Sequence:         dto.Sequence,
		ItemCode:         dto.ItemCode,
		Mask:             dto.Mask,
		DigitDate:        digitDate,
		NFType:           dto.NFType,
		SalesUOM:         dto.SalesUOM,
		WarehouseCode:    dto.WarehouseCode,
		PriceTableCode:   dto.PriceTableCode,
		RequestedQty:     dto.RequestedQty,
		UnitPrice:        dto.UnitPrice,
		DeliveryDateFirm: dto.DeliveryDateFirm,
		CustomerDelivery: dto.CustomerDelivery,
		Lot:              dto.Lot,
		CouponDelivery:   dto.CouponDelivery,
		PaidAtCashier:    dto.PaidAtCashier,
		IPIPct:           dto.IPIPct,
		ICMSPct:          dto.ICMSPct,
		PISPct:           dto.PISPct,
		COFINSPct:        dto.COFINSPct,
		STPct:            dto.STPct,
		DiscountPct:      dto.DiscountPct,
		UnitWeightNet:    dto.UnitWeightNet,
		UnitWeightGross:  dto.UnitWeightGross,
		Notes:            dto.Notes,
		Status:           entity.SalesOrderItemStatusOpen,
	}

	item.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)

	// Compute totals
	grossUnit := dto.UnitPrice * (1 - dto.DiscountPct/100)
	item.TotalGross = grossUnit * dto.RequestedQty
	ipiAmt := item.TotalGross * dto.IPIPct / 100
	item.TotalIPI = ipiAmt
	item.TotalNet = item.TotalGross
	item.TotalNetWithIPI = item.TotalGross + ipiAmt
	item.TotalST = item.TotalGross * dto.STPct / 100

	item.UnitWeightNet = dto.UnitWeightNet
	item.UnitWeightGross = dto.UnitWeightGross

	created, err := uc.Repo.CreateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return toSalesOrderItemResponse(created), nil
}

type UpdateSalesOrderItemUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *UpdateSalesOrderItemUseCase) Execute(
	ctx context.Context,
	dto request.UpdateSalesOrderItemDTO,
) (*response.SalesOrderItemResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	item := &entity.SalesOrderItem{
		Code:             dto.Code,
		RequestedQty:     dto.RequestedQty,
		UnitPrice:        dto.UnitPrice,
		AttendedQty:      dto.AttendedQty,
		CancelledQty:     dto.CancelledQty,
		DeliveryDateFirm: dto.DeliveryDateFirm,
		CustomerDelivery: dto.CustomerDelivery,
		Lot:              dto.Lot,
		CouponDelivery:   dto.CouponDelivery,
		PaidAtCashier:    dto.PaidAtCashier,
		IPIPct:           dto.IPIPct,
		ICMSPct:          dto.ICMSPct,
		PISPct:           dto.PISPct,
		COFINSPct:        dto.COFINSPct,
		STPct:            dto.STPct,
		DiscountPct:      dto.DiscountPct,
		UnitWeightNet:    dto.UnitWeightNet,
		UnitWeightGross:  dto.UnitWeightGross,
		Notes:            dto.Notes,
	}

	item.DeliveryDate = datetime.ParseDatePtr(dto.DeliveryDate)

	// Determine status from quantities
	balance := dto.RequestedQty - dto.AttendedQty - dto.CancelledQty
	switch {
	case dto.CancelledQty >= dto.RequestedQty:
		item.Status = entity.SalesOrderItemStatusCancelled
	case dto.AttendedQty >= dto.RequestedQty:
		item.Status = entity.SalesOrderItemStatusDelivered
	case dto.AttendedQty > 0 || balance < dto.RequestedQty:
		item.Status = entity.SalesOrderItemStatusPartial
	default:
		item.Status = entity.SalesOrderItemStatusOpen
	}

	// Recompute totals
	grossUnit := dto.UnitPrice * (1 - dto.DiscountPct/100)
	item.TotalGross = grossUnit * dto.RequestedQty
	ipiAmt := item.TotalGross * dto.IPIPct / 100
	item.TotalIPI = ipiAmt
	item.TotalNet = item.TotalGross
	item.TotalNetWithIPI = item.TotalGross + ipiAmt
	item.TotalST = item.TotalGross * dto.STPct / 100

	updated, err := uc.Repo.UpdateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return toSalesOrderItemResponse(updated), nil
}

type ListSalesOrderItemsUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrderItemsUseCase) Execute(ctx context.Context, salesOrderCode int64) ([]*response.SalesOrderItemResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	items, err := uc.Repo.ListItems(ctx, salesOrderCode)
	if err != nil {
		return nil, err
	}
	return toSalesOrderItemResponsePtrs(items), nil
}

type CancelSalesOrderItemUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CancelSalesOrderItemUseCase) Execute(ctx context.Context, itemCode int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.CancelItem(ctx, itemCode)
}
