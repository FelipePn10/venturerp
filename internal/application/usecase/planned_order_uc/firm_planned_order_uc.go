package planned_order_uc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	productionuc "github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	paramsrepo "github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository"
	productionentity "github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	productionrepo "github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	reqentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	structureentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	thirdparty "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/google/uuid"
)

var (
	ErrInvalidPlanningTransition = errors.New("mudança de situação da ordem planejada não permitida")
	ErrFirmDateChange            = errors.New("as datas de uma ordem planejada firme não podem ser alteradas")
	ErrKanbanReleaseDisabled     = errors.New("o parâmetro de planejamento 25 não permite liberar itens de Kanban")
	ErrOrderHasMovements         = errors.New("ordem liberada com movimentos de produção não volta para planejada")
)

// externalOpsReader is the slice of the routing repository needed to raise service
// requisitions for a firmed order's external/third-party operations.
type externalOpsReader interface {
	GetExternalOpsByItem(ctx context.Context, itemCode int64) ([]*routingentity.ExternalOp, error)
}
type productionRouteReader interface {
	GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error)
}
type productionRouteExploder interface {
	ExplodeRoute(context.Context, int64, int64) ([]*response.ProductionOrderOperationResponse, error)
}
type productionStructureReader interface {
	GetAllDirectChildren(context.Context, int64) ([]*structureentity.ItemStructure, error)
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
	ServiceLinker    ports.ProductionServiceLinker
	ReleaseValidator ports.ManufacturingReleaseValidator
	ServiceOrders    serviceOrderGenerator
	Routing          productionRouteReader
	OrderOps         productionRouteExploder
	Structure        productionStructureReader
	// Items resolve o código de negócio do item para a criação da OF.
	Items any
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
		return nil, fmt.Errorf("%w: informe as ordens", ErrInvalidPlanningTransition)
	}
	target := strings.ToUpper(strings.TrimSpace(dto.Target))
	if target != "PLANNED" && target != "RELEASED" && target != "FIRM" {
		return nil, fmt.Errorf("%w: o alvo deve ser planejada, liberada ou firme", ErrInvalidPlanningTransition)
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
		if target != "PLANNED" && order.OrderType == types.OrderProduction && uc.Routing != nil {
			mask := ""
			if order.Mask != nil {
				mask = *order.Mask
			}
			if _, routeErr := uc.Routing.GetRouteForItem(ctx, order.ItemCode, mask); routeErr != nil {
				return nil, errorsuc.NewValidationError("o item planejado não possui roteiro de fabricação aprovado")
			}
		}
		orders = append(orders, order)
	}

	start, end := datetime.ParseDatePtr(dto.StartDate), datetime.ParseDatePtr(dto.EndDate)
	if dto.StartDate != nil && start == nil || dto.EndDate != nil && end == nil {
		return nil, fmt.Errorf("%w: data inicial ou final inválida", ErrInvalidPlanningTransition)
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

	entCode, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return 0, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return 0, err
	}
	code, err := uc.ReqRepo.NextCode(ctx)
	if err != nil {
		return 0, err
	}
	notes := fmt.Sprintf("Serviços da OF do item %d (firmada)", order.ItemCode)
	req, err := reqentity.NewPurchaseRequisition(code, entCode, actor)
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
	if uc.Structure != nil {
		var startDate, endDate *string
		if order.StartDate != nil {
			value := order.StartDate.Format("2006-01-02")
			startDate = &value
		}
		if order.EndDate != nil {
			value := order.EndDate.Format("2006-01-02")
			endDate = &value
		}
		manual := &productionuc.CreateProductionOrderUseCase{Repo: uc.ProdOrderRepo, Auth: uc.Auth, Structure: uc.Structure, Routing: uc.Routing, OrderOps: uc.OrderOps, Items: uc.Items}
		// A sugestão do MRP guarda a chave legada do item; o contrato do caso de
		// uso é textual e a resolução aceita esse formato numérico.
		return manual.Execute(ctx, request.CreateProductionOrderDTO{
			PlannedOrderID: &plannedID, ItemCode: request.TextCode(strconv.FormatInt(order.ItemCode, 10)), Mask: mask, PlannedQty: order.Quantity,
			StartDate: startDate, EndDate: endDate, CostCenterID: order.CostCenterCode,
			EmployeeID: order.EmployeeCode, WarehouseID: order.WarehouseCode, MachineID: order.MachineCode,
			Priority: order.Priority, Notes: order.Notes,
		})
	}
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
	created, err := uc.ProdOrderRepo.Create(ctx, of)
	if err != nil {
		return nil, err
	}
	if uc.Routing != nil && uc.OrderOps != nil {
		route, routeErr := uc.Routing.GetRouteForItem(ctx, order.ItemCode, mask)
		if routeErr != nil {
			return nil, errorsuc.NewValidationError("o item planejado não possui roteiro de fabricação aprovado")
		}
		if _, explodeErr := uc.OrderOps.ExplodeRoute(ctx, created.ID, route.ID); explodeErr != nil {
			return nil, explodeErr
		}
	}
	return created, nil
}
