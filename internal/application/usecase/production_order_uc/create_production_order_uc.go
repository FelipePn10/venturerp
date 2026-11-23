package production_order_uc

import (
	"context"
	"fmt"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/employeeresolution"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type manualOrderRepository interface {
	CreateWithMaterials(context.Context, *entity.ProductionOrder, []*entity.ProductionOrderMaterial) (*entity.ProductionOrder, error)
}
type manualOrderDefaultsReader interface {
	GetManualOrderPlanner(context.Context, int64) (*int64, error)
}
type warehouseChecker interface {
	AlmoxarifadoDaEmpresa(context.Context, int64) (bool, error)
}
type productionRouteReader interface {
	GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error)
}
type productionRouteExploder interface {
	ExplodeRoute(context.Context, int64, int64) ([]*response.ProductionOrderOperationResponse, error)
}

type AtomicOrderCreator func(context.Context, *entity.ProductionOrder, []*entity.ProductionOrderMaterial, int64) (*entity.ProductionOrder, error)

type CreateProductionOrderUseCase struct {
	// Perdas lê a fórmula de perdas (parâmetro 20), a mesma do MRP. Opcional.
	Perdas           LeitorDaFormulaDePerda
	CreateAtomically AtomicOrderCreator
	Repo             repository.ProductionOrderRepository
	Auth             ports.AuthService
	Structure        coproductReader
	// MaskVars devolve as variáveis da configuração (item + máscara) para
	// avaliar a fórmula de quantidade do componente. Opcional: sem ela vale a
	// quantidade fixa cadastrada.
	MaskVars orderMaskVariableReader
	Routing  productionRouteReader
	OrderOps productionRouteExploder
	// Items resolve o código de negócio do item (texto) para a chave legada.
	Items any
	// Projetos acha o projeto informado na OF manual. Opcional.
	Projetos LeitorDeProjetos
	// Employees traduz o código do funcionário para a chave interna que
	// `production_orders.employee_id` exige — a tela manda o código.
	Employees employeeresolution.Finder
	// BomVersions: item sob controle de versão da estrutura só gera OF com a
	// versão de fabricação (MBOM) aprovada e vigente. Opcional.
	BomVersions VersaoDaEstrutura
	// Fatores põe a estrutura do item na unidade de estoque atual dos
	// componentes antes de copiá-la (structure_uc.RecalcularFatores).
	// Opcional: sem ela vale o que está gravado na linha.
	Fatores EstruturaEmDia
}

// EstruturaEmDia regrava a quantidade das linhas da estrutura de um item na
// unidade de estoque atual de cada componente e devolve um aviso por linha que
// não converte.
type EstruturaEmDia interface {
	RecalcularDoPai(ctx context.Context, pai int64, motivo string) (atualizadas int, semConversao []string, err error)
}

// VersaoDaEstrutura diz se o item (na máscara) está sob controle de versão da
// estrutura e se tem MBOM aprovada e vigente.
type VersaoDaEstrutura interface {
	SituacaoParaProducao(ctx context.Context, itemCode int64, mask string) (controla, aprovada bool, err error)
}

func (uc *CreateProductionOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreateProductionOrderDTO,
) (*entity.ProductionOrder, error) {
	if msg := datetime.PrimeiraInvalida(datetime.Campo("data de início", dto.StartDate), datetime.Campo("data de fim", dto.EndDate)); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	}
	if !uc.Auth.CanCreatePlannedOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	dto.CreatedBy = actor
	item, err := itemresolution.ResolveActive(ctx, uc.Items, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	itemCode := int64(item.Code)
	if !dto.PlannedQty.IsPositive() {
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
	// O planejador padrão vem do cadastro do item já como chave interna; o que
	// chega da tela vem como CÓDIGO e precisa ser traduzido.
	if dto.EmployeeID == nil {
		if defaults, ok := uc.Repo.(manualOrderDefaultsReader); ok {
			dto.EmployeeID, err = defaults.GetManualOrderPlanner(ctx, itemCode)
			if err != nil {
				return nil, err
			}
		}
	}
	employeeResolvido, err := employeeresolution.ResolveOptionalID(ctx, uc.Employees, dto.EmployeeID)
	if err != nil {
		return nil, err
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
		EmployeeID:     employeeResolvido,
		WarehouseID:    dto.WarehouseID,
		Priority:       dto.Priority,
		Notes:          dto.Notes,
		IsActive:       true,
		CreatedBy:      dto.CreatedBy,
		VinculoDemanda: dto.VinculoDemanda,
	}

	order.StartDate = datetime.ParseDatePtr(dto.StartDate)
	order.EndDate = datetime.ParseDatePtr(dto.EndDate)
	if order.StartDate != nil && order.EndDate != nil && order.StartDate.After(*order.EndDate) {
		return nil, errorsuc.NewValidationError("o início da ordem não pode ser depois do término")
	}
	order.StartDate, order.EndDate = inicioAPartirDeHoje(order.StartDate, order.EndDate, time.Now())
	if dto.CustomerCode != nil || dto.ProjectCode != nil {
		repo, ok := uc.Repo.(origemDaOFRepo)
		if !ok {
			return nil, errorsuc.NewValidationError("cliente e projeto da OF não estão disponíveis")
		}
		if order.CustomerCode, order.ProjectID, err = resolverOrigem(ctx, repo, uc.Projetos, dto.CustomerCode, dto.ProjectCode); err != nil {
			return nil, err
		}
	}
	// O almoxarifado que recebe a produção é da empresa da sessão: sem isso a
	// OF nasceria com um destino que só falharia na entrega.
	if dto.WarehouseID != nil {
		if conf, ok := uc.Repo.(warehouseChecker); ok {
			daEmpresa, err := conf.AlmoxarifadoDaEmpresa(ctx, *dto.WarehouseID)
			if err != nil {
				return nil, err
			}
			if !daEmpresa {
				return nil, errorsuc.NewValidationError(fmt.Sprintf("almoxarifado %d não é desta empresa", *dto.WarehouseID))
			}
		}
	}

	if uc.Structure == nil {
		if order.VinculoDemanda != nil {
			return nil, errorsuc.NewValidationError("a OF ligada à demanda exige a criação com materiais numa transação")
		}
		return uc.Repo.Create(ctx, order)
	}
	// A lista de materiais vale para a data em que a ordem começa. Sem data de
	// início, vale a de hoje.
	consumo := time.Now()
	if order.StartDate != nil {
		consumo = *order.StartDate
	}
	materials, rework, err := uc.MateriaisDaEstrutura(ctx, itemCode, dto.Mask, dto.PlannedQty, consumo, dto.CreatedBy)
	if err != nil {
		return nil, err
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
	if uc.BomVersions != nil {
		controla, aprovada, errVersao := uc.BomVersions.SituacaoParaProducao(ctx, itemCode, dto.Mask)
		if errVersao != nil {
			return nil, errVersao
		}
		if controla && !aprovada {
			return nil, errorsuc.NewValidationError("a estrutura do item está sob controle de versão e não tem versão de fabricação (MBOM) aprovada e vigente; aprove na VENT0210 (Versões) antes de criar a OF")
		}
	}
	var route *routingentity.ManufacturingRoute
	if uc.Routing != nil && uc.OrderOps != nil {
		var routeErr error
		// A quantidade entra na escolha: roteiro com faixa de lote que casa vence.
		route, routeErr = routingrepo.RoteiroParaQuantidade(ctx, uc.Routing, itemCode, dto.Mask, dto.PlannedQty)
		if routeErr != nil {
			return nil, errorsuc.NewValidationError("o item não possui roteiro de fabricação aprovado; cadastre o roteiro antes de criar a OF")
		}
	}
	if route != nil && uc.CreateAtomically != nil {
		return uc.CreateAtomically(ctx, order, materials, route.ID)
	}
	created, err := atomicRepo.CreateWithMaterials(ctx, order, materials)
	if err != nil {
		return nil, err
	}
	if route != nil {
		if _, explodeErr := uc.OrderOps.ExplodeRoute(ctx, created.ID, route.ID); explodeErr != nil {
			return nil, explodeErr
		}
	}
	return created, nil
}

// MateriaisDaEstrutura monta a lista de materiais da OF pela estrutura do item
// (genéricos + os da máscara), vigente na data de consumo: primário dos
// substitutos, fórmula da quantidade, perda e quantidade fixa por ordem — a
// mesma conta do MRP. Co-produto e material do cliente ficam de fora. rework
// diz se o próprio item entra (ordem de retrabalho). É usada na criação da OF
// e para completar a lista da OF que nasceu sem ela.
func (uc *CreateProductionOrderUseCase) MateriaisDaEstrutura(ctx context.Context, itemCode int64, mask string, quantidade decimal.Decimal, consumo time.Time, ator uuid.UUID) ([]*entity.ProductionOrderMaterial, bool, error) {
	// A linha guarda a quantidade já convertida; se a unidade de estoque ou a
	// conversão do componente mudou e o MRP ainda não rodou, o número gravado
	// é o velho (o tubo de 104 mm virava 104 m). Acerta antes de copiar.
	if uc.Fatores != nil {
		_, sem, err := uc.Fatores.RecalcularDoPai(ctx, itemCode, "conferência na criação da OF")
		if err != nil {
			return nil, false, fmt.Errorf("conferindo as unidades da estrutura do item: %w", err)
		}
		if len(sem) > 0 {
			return nil, false, errorsuc.NewValidationError("a OF não pode copiar a estrutura com quantidade sem conversão: " + strings.Join(sem, "; "))
		}
	}
	children, err := uc.componentesDaOrdem(ctx, itemCode, mask)
	if err != nil {
		return nil, false, err
	}
	vigentes := make([]*structentity.ItemStructure, 0, len(children))
	for _, child := range children {
		if child.VigenteEm(consumo) {
			vigentes = append(vigentes, child)
		}
	}
	var vars map[string]decimal.Decimal
	if uc.MaskVars != nil && mask != "" {
		var errMascara error
		vars, errMascara = uc.MaskVars.GetMaskAnswersWithNames(ctx, itemCode, mask)
		if errMascara != nil {
			return nil, false, fmt.Errorf("lendo as respostas da máscara do item: %w", errMascara)
		}
	}
	formula, err := formulaDePerda(ctx, uc.Perdas)
	if err != nil {
		return nil, false, err
	}
	materials := []*entity.ProductionOrderMaterial{}
	rework := false
	for _, child := range structentity.SelectPrimarySubstituteComponents(vigentes) {
		perUnit, _ := child.ResolvedQuantity(vars)
		if child.IsCoproduct || !perUnit.IsPositive() {
			continue
		}
		// Material do cliente (beneficiamento) não é demanda do estoque próprio:
		// entra pela NF-e de remessa, no razão de terceiros. Como DEMAND ele
		// apareceria como falta, entraria no MRP e seria requisitado do estoque.
		if child.CustomerSupplied {
			continue
		}
		base := quantidade
		if child.IsFixedQty {
			base = decimal.NewFromInt(1)
		}
		// Mesma conta e mesma escala do MRP: a compra que ele sugeriu cobre a OF.
		quantity := structentity.QuantidadeDoComponente(base, perUnit, child.LossPercentage, formula)
		automatic, warehouse, infoErr := uc.Repo.GetItemAutomaticIssue(ctx, child.ChildCode)
		if infoErr != nil {
			return nil, false, infoErr
		}
		materials = append(materials, &entity.ProductionOrderMaterial{Kind: entity.MaterialDemand,
			ItemCode: child.ChildCode, Quantity: quantity, WarehouseID: warehouse,
			AutomaticIssue: automatic, CreatedBy: ator})
		rework = rework || child.ChildCode == itemCode
	}
	return materials, rework, nil
}

// orderMaskVariableReader resolve as variáveis da configuração de um item.
type orderMaskVariableReader interface {
	GetMaskAnswersWithNames(ctx context.Context, itemCode int64, mask string) (map[string]decimal.Decimal, error)
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

// inicioAPartirDeHoje: a OF nasce hoje ou depois — início vazio ou no passado
// (MRP com prazo vencido, data digitada para trás) vira hoje, e o término nunca
// fica antes do início.
func inicioAPartirDeHoje(inicio, fim *time.Time, agora time.Time) (*time.Time, *time.Time) {
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.UTC)
	if inicio == nil || inicio.Before(hoje) {
		inicio = &hoje
	}
	if fim != nil && fim.Before(*inicio) {
		f := *inicio
		fim = &f
	}
	return inicio, fim
}
