package production_order_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
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
type productionRouteReader interface {
	GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error)
}
type productionRouteExploder interface {
	ExplodeRoute(context.Context, int64, int64) ([]*response.ProductionOrderOperationResponse, error)
}

type CreateProductionOrderUseCase struct {
	Repo      repository.ProductionOrderRepository
	Auth      ports.AuthService
	Structure coproductReader
	// MaskVars devolve as variáveis da configuração (item + máscara) para
	// avaliar a fórmula de quantidade do componente. Opcional: sem ela vale a
	// quantidade fixa cadastrada.
	MaskVars orderMaskVariableReader
	Routing  productionRouteReader
	OrderOps productionRouteExploder
	// Items resolve o código de negócio do item (texto) para a chave legada.
	Items any
}

func (uc *CreateProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreateProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.CreatedBy = actor
	item, err := itemresolution.Resolve(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	itemCode := int64(item.Code)
	if dto.PlannedQty <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade planejada deve ser maior que zero")
	}

	var nextNum int64
	if dto.OrderNumber != nil {
		if *dto.OrderNumber <= 0 {
			return nil, errorsuc.NewValidationError("o número da ordem deve ser maior que zero")
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
			dto.EmployeeID, err = defaults.GetManualOrderPlanner(ctx, itemCode)
			if err != nil {
				return nil, err
			}
		}
	}

	order := &entity.ProductionOrder{
		OrderNumber:    nextNum,
		PlannedOrderID: dto.PlannedOrderID,
		ItemCode:       itemCode,
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
	children, err := uc.componentesDaOrdem(ctx, itemCode, dto.Mask)
	if err != nil {
		return nil, err
	}
	// A lista de materiais vale para a data em que a ordem começa. Sem data de
	// início, vale a de hoje.
	consumo := time.Now()
	if order.StartDate != nil {
		consumo = *order.StartDate
	}
	vigentes := make([]*structentity.ItemStructure, 0, len(children))
	for _, child := range children {
		if child.VigenteEm(consumo) {
			vigentes = append(vigentes, child)
		}
	}
	var vars map[string]float64
	if uc.MaskVars != nil && dto.Mask != "" {
		vars, _ = uc.MaskVars.GetMaskAnswersWithNames(ctx, itemCode, dto.Mask)
	}
	materials := []*entity.ProductionOrderMaterial{}
	rework := false
	for _, child := range structentity.SelectPrimarySubstituteComponents(vigentes) {
		perUnit, _ := child.ResolvedQuantity(vars)
		if child.IsCoproduct || perUnit <= 0 {
			continue
		}
		quantity := decimal.NewFromFloat(perUnit)
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
		rework = rework || child.ChildCode == itemCode
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
		return nil, errorsuc.NewValidationError("a criação manual de ordem em uma única transação não está disponível")
	}
	created, err := atomicRepo.CreateWithMaterials(ctx, order, materials)
	if err != nil {
		return nil, err
	}
	if uc.Routing != nil && uc.OrderOps != nil {
		route, routeErr := uc.Routing.GetRouteForItem(ctx, itemCode, dto.Mask)
		if routeErr != nil {
			return nil, errorsuc.NewValidationError("o item não possui roteiro de fabricação aprovado; cadastre o roteiro antes de criar a OF")
		}
		if _, explodeErr := uc.OrderOps.ExplodeRoute(ctx, created.ID, route.ID); explodeErr != nil {
			return nil, explodeErr
		}
	}
	return created, nil
}

// orderMaskVariableReader resolve as variáveis da configuração de um item.
type orderMaskVariableReader interface {
	GetMaskAnswersWithNames(ctx context.Context, itemCode int64, mask string) (map[string]float64, error)
}

// maskBOMReader é a leitura de estrutura que respeita a máscara da ordem.
type maskBOMReader interface {
	GetDirectChildrenForMask(ctx context.Context, parentCode int64, mask string) ([]*structentity.ItemStructure, error)
}

// componentesDaOrdem devolve os filhos diretos que valem para a configuração da
// ordem: os genéricos mais os da máscara informada.
//
// A criação da OF usava GetAllDirectChildren, que não filtra máscara: a ordem de
// um item configurado recebia os componentes de TODAS as variantes — todas as
// cores, todas as medidas — além dos genéricos. A leitura por máscara só não é
// usada quando o repositório não a oferece (os dublês de teste antigos).
func (uc *CreateProductionOrderUseCase) componentesDaOrdem(ctx context.Context, itemCode int64, mask string) ([]*structentity.ItemStructure, error) {
	if r, ok := uc.Structure.(maskBOMReader); ok {
		return r.GetDirectChildrenForMask(ctx, itemCode, mask)
	}
	return uc.Structure.GetAllDirectChildren(ctx, itemCode)
}
