package production_order_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/shopspring/decimal"
)

type manualOrderRepository interface {
	CreateWithMaterials(context.Context, *entity.ProductionOrder, []*entity.ProductionOrderMaterial) (*entity.ProductionOrder, error)
}
type manualOrderDefaultsReader interface {
	GetManualOrderPlanner(context.Context, int64) (*int64, error)
}

type CreateProductionOrderUseCase struct {
	Repo      repository.ProductionOrderRepository
	Auth      ports.AuthService
	Structure coproductReader
}

func (uc *CreateProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreateProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode == 0 {
		return nil, errorsuc.NewValidationError("item_code is required")
	}
	if dto.PlannedQty <= 0 {
		return nil, errorsuc.NewValidationError("planned_qty must be greater than zero")
	}

	var nextNum int64
	var err error
	if dto.OrderNumber != nil {
		if *dto.OrderNumber <= 0 {
			return nil, errorsuc.NewValidationError("order_number must be positive")
		}
		nextNum = *dto.OrderNumber
	} else {
		nextNum, err = uc.Repo.GetNextOrderNumber(ctx)
		if err != nil {
			nextNum = 1
		}
	}
	if dto.EmployeeID == nil {
		if defaults, ok := uc.Repo.(manualOrderDefaultsReader); ok {
			dto.EmployeeID, err = defaults.GetManualOrderPlanner(ctx, dto.ItemCode)
			if err != nil {
				return nil, err
			}
		}
	}

	order := &entity.ProductionOrder{
		OrderNumber:    nextNum,
		PlannedOrderID: dto.PlannedOrderID,
		ItemCode:       dto.ItemCode,
		Mask:           dto.Mask,
		PlannedQty:     dto.PlannedQty,
		Status:         entity.StatusOpen,
		MachineID:      dto.MachineID,
		CostCenterID:   dto.CostCenterID,
		EmployeeID:     dto.EmployeeID,
		WarehouseID:    dto.WarehouseID,
		Priority:       dto.Priority,
		Notes:          dto.Notes,
		IsActive:       true,
		CreatedBy:      dto.CreatedBy,
	}

	order.StartDate = datetime.ParseDatePtr(dto.StartDate)
	order.EndDate = datetime.ParseDatePtr(dto.EndDate)

	if uc.Structure == nil {
		return uc.Repo.Create(ctx, order)
	}
	children, err := uc.Structure.GetAllDirectChildren(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	materials := []*entity.ProductionOrderMaterial{}
	rework := false
	for _, child := range structentity.SelectPrimarySubstituteComponents(children) {
		if child.IsCoproduct || child.Quantity <= 0 {
			continue
		}
		quantity := decimal.NewFromFloat(child.Quantity)
		if !child.IsFixedQty {
			quantity = quantity.Mul(decimal.NewFromFloat(dto.PlannedQty))
		}
		quantity = quantity.Mul(decimal.NewFromFloat(1 + child.LossPercentage/100))
		automatic, warehouse, infoErr := uc.Repo.GetItemAutomaticIssue(ctx, child.ChildCode)
		if infoErr != nil {
			return nil, infoErr
		}
		materials = append(materials, &entity.ProductionOrderMaterial{Kind: entity.MaterialDemand,
			ItemCode: child.ChildCode, Quantity: quantity, WarehouseID: warehouse,
			AutomaticIssue: automatic, CreatedBy: dto.CreatedBy})
		rework = rework || child.ChildCode == dto.ItemCode
	}
	if rework {
		message := "ORDEM DE RETRABALHO"
		if order.Notes == nil || strings.TrimSpace(*order.Notes) == "" {
			order.Notes = &message
		} else if !strings.Contains(*order.Notes, message) {
			joined := strings.TrimSpace(*order.Notes) + "\n" + message
			order.Notes = &joined
		}
	}
	atomicRepo, ok := uc.Repo.(manualOrderRepository)
	if !ok {
		return nil, errorsuc.NewValidationError("production repository does not support atomic manual order creation")
	}
	return atomicRepo.CreateWithMaterials(ctx, order, materials)
}
