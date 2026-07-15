package planned_order_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	paramsrepo "github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	reqentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	thirdparty "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/google/uuid"
)

var (
	ErrInvalidPlanningTransition = errors.New("invalid planned order transition")
	ErrFirmDateChange            = errors.New("firm planned order dates cannot be changed")
	ErrKanbanReleaseDisabled     = errors.New("planning parameter 25 does not allow releasing Kanban items")
	ErrOrderHasMovements         = errors.New("released order with production movements cannot return to planned")
)

// externalOpsReader is the slice of the routing repository needed to raise service
// requisitions for a firmed order's external/third-party operations.
type externalOpsReader interface {
	GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*routingentity.ExternalOp, error)
}
type serviceOrderGenerator interface {
	CreateOrdersForProduction(context.Context, int64, uuid.UUID) ([]thirdparty.ServiceOrder, error)
	LinkRequisitionToProduction(context.Context, int64, int64) error
}

type FirmPlannedOrderUseCase struct {
	Repo   repository.PlannedOrderRepository
	Auth   ports.AuthService
	Params paramsrepo.PlanningParamRepository
	// ProdOrderRepo is optional. When set, firming a PRODUCTION planned order
	// also creates the corresponding Production Order (OF), mirroring the
	// approve→purchase-order flow already in place on the purchasing side.
	ProdOrderRepo productionrepo.ProductionOrderRepository

	// Subcontracting hook (R4) — all optional. When set, firming a production order
	// whose item has external/third-party operations raises a service purchase
	// requisition (one item per external op with a service item).
	ReqRepo          reqrepo.PurchaseRequisitionRepository
	ExternalOps      externalOpsReader
	EnterpriseCode   int64 // enterprise the requisition belongs to (defaults to 1)
	ServiceLinker    ports.ProductionServiceLinker
	ReleaseValidator ports.ManufacturingReleaseValidator
	ServiceOrders    serviceOrderGenerator
}

func (uc *FirmPlannedOrderUseCase) Execute(ctx context.Context, dto request.FirmOrderDTO) (*response.PlannedOrderResponse, error) {
	result, err := uc.ExecuteTransition(ctx, request.TransitionPlannedOrderDTO{OrderCodes: []int64{dto.OrderCode}, Target: "FIRM"})
	if err != nil {
		return nil, err
	}
	return result[0], nil
}

func (uc *FirmPlannedOrderUseCase) ExecuteTransition(ctx context.Context, dto request.TransitionPlannedOrderDTO) ([]*response.PlannedOrderResponse, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if len(dto.OrderCodes) == 0 {
		return nil, fmt.Errorf("%w: order_codes is required", ErrInvalidPlanningTransition)
	}
	target := strings.ToUpper(strings.TrimSpace(dto.Target))
	if target != "PLANNED" && target != "RELEASED" && target != "FIRM" {
		return nil, fmt.Errorf("%w: target must be PLANNED, RELEASED or FIRM", ErrInvalidPlanningTransition)
	}
	if target == "FIRM" && (dto.StartDate != nil || dto.EndDate != nil) {
		return nil, ErrFirmDateChange
	}

	orders := make([]*entity.PlannedOrder, 0, len(dto.OrderCodes))
	for _, code := range dto.OrderCodes {
		order, err := uc.Repo.GetByCode(ctx, code)
		if err != nil {
			return nil, err
		}
		if err := uc.validateTransition(ctx, order, target); err != nil {
			return nil, fmt.Errorf("order %d: %w", code, err)
		}
		orders = append(orders, order)
	}

	start, end := datetime.ParseDatePtr(dto.StartDate), datetime.ParseDatePtr(dto.EndDate)
	if dto.StartDate != nil && start == nil || dto.EndDate != nil && end == nil {
		return nil, fmt.Errorf("%w: invalid start_date or end_date", ErrInvalidPlanningTransition)
	}
	result := make([]*response.PlannedOrderResponse, 0, len(orders))
	for _, previous := range orders {
		wasPlanned := previous.Status == types.StatusPlanned
		if target != "FIRM" && (dto.StartDate != nil || dto.EndDate != nil) {
			if _, err := uc.Repo.UpdateDates(ctx, previous.Code, start, end); err != nil {
				return nil, err
			}
		}
		status, firm := string(types.StatusReleased), target == "FIRM"
		if target == "PLANNED" {
			status = string(types.StatusPlanned)
		}
		order, err := uc.Repo.SetPlanningState(ctx, previous.Code, status, firm)
		if err != nil {
			return nil, err
		}
		if target != "PLANNED" && wasPlanned && uc.ProdOrderRepo != nil && order.OrderType == types.OrderProduction {
			productionOrder, err := uc.createProductionOrder(ctx, order)
			if err != nil {
				return nil, err
			}
			var requisitionCode int64
			if uc.ReqRepo != nil && uc.ExternalOps != nil {
				var reqErr error
				requisitionCode, reqErr = uc.generateServiceRequisition(ctx, order)
				if reqErr != nil {
					return nil, reqErr
				}
				if requisitionCode != 0 && uc.ServiceLinker != nil {
					if linkErr := uc.ServiceLinker.LinkServiceRequisition(ctx, productionOrder.ID, requisitionCode); linkErr != nil {
						return nil, linkErr
					}
				}
			}
			if uc.ServiceOrders != nil {
				if _, serviceErr := uc.ServiceOrders.CreateOrdersForProduction(ctx, productionOrder.ID, order.CreatedBy); serviceErr != nil {
					return nil, serviceErr
				}
				if requisitionCode != 0 {
					if linkErr := uc.ServiceOrders.LinkRequisitionToProduction(ctx, productionOrder.ID, requisitionCode); linkErr != nil {
						return nil, linkErr
					}
				}
			}
		}
		result = append(result, toPlannedOrderResponse(order))
	}
	return result, nil
}

func (uc *FirmPlannedOrderUseCase) validateTransition(ctx context.Context, order *entity.PlannedOrder, target string) error {
	if order.IsFirm {
		if target == "FIRM" {
			return nil
		}
		return ErrInvalidPlanningTransition
	}
	if target != "PLANNED" && uc.ReleaseValidator != nil {
		if err := uc.ReleaseValidator.ValidateProductionRelease(ctx, order.ItemCode); err != nil {
			return err
		}
	}
	if target == "PLANNED" {
		if order.Status != types.StatusReleased {
			return ErrInvalidPlanningTransition
		}
		moved, err := uc.Repo.HasProductionMovements(ctx, order.Code)
		if err != nil {
			return err
		}
		if moved {
			return ErrOrderHasMovements
		}
		return nil
	}
	if order.Status != types.StatusPlanned && order.Status != types.StatusReleased {
		return ErrInvalidPlanningTransition
	}
	kanban, err := uc.Repo.IsKanbanItem(ctx, order.ItemCode)
	if err != nil {
		return err
	}
	if !kanban {
		return nil
	}
	allowed := false
	if uc.Params != nil {
		if param, err := uc.Params.GetByNumber(ctx, 25); err == nil {
			v := strings.ToUpper(strings.TrimSpace(param.Value))
			allowed = v == "S" || v == "SIM" || v == "1" || v == "TRUE" || v == "YES"
		}
	}
	if !allowed {
		return ErrKanbanReleaseDisabled
	}
	return nil
}

// generateServiceRequisition raises one purchase requisition covering the service
// items of the firmed order's external/third-party operations. Returns nil when the
// item has no external operations with a service item configured.
func (uc *FirmPlannedOrderUseCase) generateServiceRequisition(ctx context.Context, order *entity.PlannedOrder) (int64, error) {
	ext, err := uc.ExternalOps.GetExternalOpsByItem(ctx, order.ItemCode)
	if err != nil || len(ext) == 0 {
		return 0, err
	}

	// Only external ops with a service item to buy generate requisition lines.
	var withService []*routingentity.ExternalOp
	for _, op := range ext {
		if op.ServiceItemCode != nil {
			withService = append(withService, op)
		}
	}
	if len(withService) == 0 {
		return 0, nil
	}

	entCode := uc.EnterpriseCode
	if uc.ServiceLinker != nil {
		entCode, err = uc.ServiceLinker.CurrentEnterpriseCode(ctx)
		if err != nil {
			return 0, err
		}
	} else if entCode == 0 {
		entCode = 1
	}
	code, err := uc.ReqRepo.NextCode(ctx)
	if err != nil {
		return 0, err
	}
	notes := fmt.Sprintf("Serviços da OF do item %d (firmada)", order.ItemCode)
	req, err := reqentity.NewPurchaseRequisition(code, entCode, order.CreatedBy)
	if err != nil {
		return 0, err
	}
	req.Notes = &notes
	created, err := uc.ReqRepo.Create(ctx, req)
	if err != nil {
		return 0, err
	}

	serv := "SERV"
	for i, op := range withService {
		app := fmt.Sprintf("Op. externa: %s (%.2fh)", op.OperationName, op.EffectiveHours)
		item := &reqentity.PurchaseRequisitionItem{
			RequisitionCode: created.Code,
			Sequence:        int32(i + 1),
			ItemCode:        *op.ServiceItemCode,
			Quantity:        order.Quantity,
			UOM:             &serv,
			SuggestedPrice:  op.CostPerUnit,
			Application:     &app,
		}
		if op.LeadTimeDays > 0 {
			d := time.Now().AddDate(0, 0, int(op.LeadTimeDays))
			item.DeliveryDate = &d
		}
		if _, err := uc.ReqRepo.AddItem(ctx, item); err != nil {
			return created.Code, err
		}
	}
	return created.Code, nil
}

// createProductionOrder builds the OF from the firmed planned order, mirroring
// the manual CreateProductionOrderUseCase. The production numbering sequence
// is independent from planned-order numbering; traceability uses the internal
// planned-order ID stored by the foreign key.
func (uc *FirmPlannedOrderUseCase) createProductionOrder(ctx context.Context, order *entity.PlannedOrder) (*productionentity.ProductionOrder, error) {
	mask := ""
	if order.Mask != nil {
		mask = *order.Mask
	}
	plannedID := order.ID
	orderNumber, err := uc.ProdOrderRepo.GetNextOrderNumber(ctx)
	if err != nil {
		return nil, err
	}

	of := &productionentity.ProductionOrder{
		OrderNumber:    orderNumber,
		PlannedOrderID: &plannedID,
		ItemCode:       order.ItemCode,
		Mask:           mask,
		PlannedQty:     order.Quantity,
		Status:         productionentity.StatusOpen,
		CostCenterID:   order.CostCenterCode,
		EmployeeID:     order.EmployeeCode,
		WarehouseID:    order.WarehouseCode,
		MachineID:      order.MachineCode,
		Priority:       order.Priority,
		Notes:          order.Notes,
		StartDate:      order.StartDate,
		EndDate:        order.EndDate,
		CreatedBy:      order.CreatedBy,
	}
	return uc.ProdOrderRepo.Create(ctx, of)
}
