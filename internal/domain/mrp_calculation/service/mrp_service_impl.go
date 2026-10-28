package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/shopspring/decimal"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	inddemandrepo "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	calrepo "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	orderpriority "github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	planentity "github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	planrepo "github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
	restrictionrepo "github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	forecastrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structrepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// tooEarlyDays is the threshold (in calendar days) beyond which a firm supply
// arriving before the need date triggers a RESCHEDULE_OUT exception.
const tooEarlyDays = 30

// expediteThreshold is the maximum number of late days for which an EXPEDITE
// exception is generated instead of RESCHEDULE_IN.
const expediteThreshold = 5

// excessThreshold is the fraction above net requirement that is tolerated
// before an EXCESS_PROJECTED exception is raised (1 % tolerance).
const excessThreshold = 0.01

type MRPServiceImpl struct {
	MRPRepo         mrprepo.MRPCalculationRepository
	StructRepo      structrepo.ItemStructureRepository
	DemandRepo      inddemandrepo.IndependentDemandRepository
	CalRepo         calrepo.IndustrialCalendarRepository
	ItemRepo        itemrepo.ItemRepository
	PlanRepo        planrepo.ProductionPlanRepository
	ForecastRepo    forecastrepo.SalesForecastRepository
	RestrictionRepo restrictionrepo.RestrictionRepository
	SupplyPort      ports.PlannedOrderSupplyPort
	RoutingRepo     routingrepo.RoutingRepository
	// Scenarios lê os ajustes do cenário quando o plano é de simulação.
	Scenarios ScenarioReader // nil = fallback to configured lead time
	// MaskVars resolve as variáveis da configuração (as "perguntas" respondidas)
	// para avaliar fórmulas de quantidade na explosão. Opcional: sem ela, vale a
	// quantidade fixa cadastrada no componente.
	MaskVars MaskVariableReader
}

// MaskVariableReader devolve variável → valor da configuração de um item,
// alimentando as fórmulas de quantidade da estrutura (FENG0210).
type MaskVariableReader interface {
	GetMaskAnswersWithNames(ctx context.Context, itemCode int64, mask string) (map[string]float64, error)
}

// maskVarCache evita reconsultar as respostas do mesmo par item/máscara durante
// uma explosão inteira.
type maskVarCache struct {
	reader MaskVariableReader
	cache  map[string]map[string]float64
}

func newMaskVarCache(reader MaskVariableReader) *maskVarCache {
	return &maskVarCache{reader: reader, cache: map[string]map[string]float64{}}
}

func (c *maskVarCache) vars(ctx context.Context, itemCode int64, mask string) map[string]float64 {
	if c == nil || c.reader == nil || mask == "" {
		return nil
	}
	key := fmt.Sprintf("%d|%s", itemCode, mask)
	if cached, ok := c.cache[key]; ok {
		return cached
	}
	vars, err := c.reader.GetMaskAnswersWithNames(ctx, itemCode, mask)
	if err != nil {
		vars = nil
	}
	c.cache[key] = vars
	return vars
}

func NewMRPService(
	mrpRepo mrprepo.MRPCalculationRepository,
	structRepo structrepo.ItemStructureRepository,
	demandRepo inddemandrepo.IndependentDemandRepository,
	calRepo calrepo.IndustrialCalendarRepository,
	itemRepo itemrepo.ItemRepository,
	supplyPort ports.PlannedOrderSupplyPort,
	planRepo planrepo.ProductionPlanRepository,
	forecastRepo forecastrepo.SalesForecastRepository,
	restrictionRepo restrictionrepo.RestrictionRepository,
	routingRepo routingrepo.RoutingRepository,
	maskVars ...MaskVariableReader,
) MRPService {
	svc := &MRPServiceImpl{
		MRPRepo:         mrpRepo,
		StructRepo:      structRepo,
		DemandRepo:      demandRepo,
		CalRepo:         calRepo,
		ItemRepo:        itemRepo,
		SupplyPort:      supplyPort,
		PlanRepo:        planRepo,
		ForecastRepo:    forecastRepo,
		RestrictionRepo: restrictionRepo,
		RoutingRepo:     routingRepo,
	}
	if len(maskVars) > 0 {
		svc.MaskVars = maskVars[0]
	}
	return svc
}

// cachedItemMRP holds the item fields the MRP needs, looked up once per item.
type cachedItemMRP struct {
	engineeringType int
	typeMRP         int
	ghost           bool
	reorderPoint    *valueobject.ReorderPoint
	// Política de lote do próprio cadastro do item. Eram gravados na VENT0200 e
	// o MRP nunca os lia: só uma "regra configurada" separada aplicava lote
	// mínimo, então os campos da tela não mudavam o planejamento.
	minimumLot  decimal.Decimal
	multipleLot decimal.Decimal
}

// =============================================================================
// Calculate — orquestração principal com dispatch por modo de planejamento
// =============================================================================

// Calculate orchestrates the full MRP run. Strategy:
//  1. Load TypedPlanningParams and dispatch by PlanningTypes.
//  2. One recursive CTE loads the entire BOM tree (no N+1).
//  3. Two bulk queries load all snapshots and configured rules.
//  4. If SupplyPort is wired: one query loads all firm supply for time-phased netting.
//  5. The main loop processes items level by level; each item costs zero extra queries
//     (item-type lookups are cached; workday subtraction is a single PG function call).
//  6. After the main loop: exception messages are generated comparing firm supply
//     against the computed net requirements.
//  7. Post-processing: machine integration, priority generation.
func (s *MRPServiceImpl) Calculate(ctx context.Context, planCode, initialOrderNumber int64, generateLLC bool) (*entity.MRPCalculationLog, error) {
	log, err := s.MRPRepo.StartCalculation(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("starting calculation log: %w", err)
	}

	errs := make(map[string]interface{})

	// Sobra do cálculo anterior que não sai fica somada ao novo: a falha vai
	// para o log (COMPLETED_WITH_ERRORS) em vez de sumir.
	registrarErro(errs, "limpar_sugestoes", s.MRPRepo.DeleteSuggestionsByPlan(ctx, planCode))
	registrarErro(errs, "limpar_perfis", s.MRPRepo.DeleteProfilesByPlan(ctx, planCode))
	registrarErro(errs, "limpar_detalhes_perfil", s.MRPRepo.DeleteProfileDetailsByPlan(ctx, planCode))
	registrarErro(errs, "limpar_excecoes", s.MRPRepo.DeleteExceptionsByPlan(ctx, planCode))
	// O pegging aponta sugestões que acabaram de ser apagadas: sai junto, para
	// um cálculo que pare no meio não deixar vínculo com sugestão inexistente.
	if w, ok := s.MRPRepo.(peggingWriter); ok {
		if err := w.ReplacePegging(ctx, planCode, nil); err != nil {
			errs["pegging"] = err.Error()
		}
	}

	// Carrega parâmetros de planejamento tipados.
	params, err := s.MRPRepo.LoadTypedPlanningParams(ctx)
	if err != nil {
		errs["load_params"] = err.Error()
		return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
	}

	// Load the production plan to drive demand filtering and item scope.
	plan, err := s.PlanRepo.GetByCode(ctx, planCode)
	if err != nil {
		errs["load_plan"] = err.Error()
		return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
	}

	// A specific sales-order item is an exclusive demand source. Otherwise the
	// plan combines open sales orders, independent demands and forecasts.
	var demands []*inddemandentity.IndependentDemand
	var demandInputs []*entity.MRPInput
	if plan.OrderItemCode != nil {
		demandInputs, err = s.MRPRepo.ListOpenSalesOrderDemands(ctx, planCode, plan.OrderItemCode)
		if err != nil || len(demandInputs) == 0 {
			if err != nil {
				errs["load_sales_order_item"] = err.Error()
			} else {
				errs["load_sales_order_item"] = "sales order item is not an open demand for this enterprise"
			}
			return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
		}
	} else if plan.IndependentDemands != planentity.IndependentDemandsNo {
		demands, err = s.DemandRepo.List(ctx)
		if err != nil {
			errs["load_demands"] = err.Error()
			return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
		}
		if plan.IndependentDemands == planentity.IndependentDemandsFromDate {
			demands = s.filterDemandsFromDate(demands, plan.Parameters)
		}
	}

	if plan.OrderItemCode == nil {
		salesInputs, loadErr := s.MRPRepo.ListOpenSalesOrderDemands(ctx, planCode, nil)
		if loadErr != nil {
			errs["load_sales_orders"] = loadErr.Error()
			return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
		}
		demandInputs = append(demandInputs, salesInputs...)
		previsao, errPrevisao := s.loadForecastDemands(ctx, planCode)
		registrarErro(errs, "load_forecasts", errPrevisao)
		demandInputs = append(demandInputs, previsao...)
	}

	// Plano de cenário: ajustes de demanda, estoque e prazo entram em memória.
	var ajustes *entity.ScenarioAdjustments
	if plan.IsScenario {
		if s.Scenarios == nil {
			errs["scenario"] = "cálculo de cenário indisponível nesta instalação"
			return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
		}
		ajustes, err = s.Scenarios.ScenarioAdjustments(ctx, planCode)
		if err != nil {
			errs["scenario"] = err.Error()
			return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
		}
		demands, demandInputs = applyScenarioDemands(ajustes, demands, demandInputs, planCode)
	}

	allowedItems, err := s.resolveAllowedItemSet(ctx, plan)
	if err != nil {
		errs["resolve_classification"] = err.Error()
		return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
	}

	// Collect root items from demands + forecasts (respecting plan scope).
	seen := make(map[int64]bool)
	var rootItems []int64
	for _, d := range demands {
		if !s.itemAllowed(allowedItems, d.ItemCode) {
			continue
		}
		if !seen[d.ItemCode] {
			seen[d.ItemCode] = true
			rootItems = append(rootItems, d.ItemCode)
		}
	}
	for _, fi := range demandInputs {
		if !s.itemAllowed(allowedItems, fi.ItemCode) {
			continue
		}
		if !seen[fi.ItemCode] {
			seen[fi.ItemCode] = true
			rootItems = append(rootItems, fi.ItemCode)
		}
	}

	// Se nenhum item tem demanda, termina cedo.
	noMRPDemand := len(demands) == 0 && len(demandInputs) == 0

	// Bulk load 1: entire BOM tree.
	bomMap, err := s.StructRepo.LoadBOMForRoots(ctx, rootItems)
	if err != nil {
		bomMap = make(map[int64][]*structentity.ItemStructure)
	}
	llcMap := buildLLCFromBOM(bomMap, rootItems)

	// Bulk load 2 & 3: snapshots and configured rules.
	snapshotMap, err := s.MRPRepo.ListAllStockSnapshots(ctx)
	if err != nil {
		snapshotMap = make(map[int64]*entity.StockSnapshot)
	}
	rulesMap, err := s.MRPRepo.ListAllConfiguredRules(ctx)
	if err != nil {
		rulesMap = make(map[int64][]*entity.ConfiguredItemRule)
	}
	applyScenarioStock(ajustes, snapshotMap)
	applyScenarioLeadTimes(ajustes, rulesMap)

	// Expande allCodes para incluir itens de todas as fontes.
	allCodesFromBOM := collectAllItemCodes(bomMap, rootItems)
	allCodes := make(map[int64]bool, len(allCodesFromBOM))
	for _, c := range allCodesFromBOM {
		allCodes[c] = true
	}
	for _, d := range demands {
		allCodes[d.ItemCode] = true
	}
	for _, fi := range demandInputs {
		allCodes[fi.ItemCode] = true
	}
	var allCodeSlice []int64
	for c := range allCodes {
		allCodeSlice = append(allCodeSlice, c)
	}

	// Bulk load 4: firm supply.
	var supplyMap map[int64][]ports.SupplyEntry
	if s.SupplyPort != nil {
		var errSuprimento error
		supplyMap, errSuprimento = s.SupplyPort.ListFirmSupplyForItems(ctx, allCodeSlice)
		registrarErro(errs, "load_firm_supply", errSuprimento)
	}
	if supplyMap == nil {
		supplyMap = make(map[int64][]ports.SupplyEntry)
	}

	// Bulk load 5: restricted items.
	restrictedItems := make(map[int64]struct{})
	if s.RestrictionRepo != nil {
		lidos, errRestricao := s.RestrictionRepo.ListRestrictedItemCodes(ctx, allCodeSlice)
		registrarErro(errs, "load_restrictions", errRestricao)
		if lidos != nil {
			restrictedItems = lidos
		}
	}

	// Item cache — lazy load from DB.
	itemCache := make(map[int64]*cachedItemMRP)

	// Base das mensagens de exceção: o netting consome o estoque e o
	// suprimento firme (advanceNettingState), então a análise precisa da
	// posição ANTES do cálculo e de cada necessidade bruta registrada.
	excecoes := newExceptionBasis(snapshotMap, supplyMap)
	totalItems := 0
	totalOrders := 0
	nextOrderNumber := initialOrderNumber
	// Sugestão de cenário não recebe número de ordem: o número é único na
	// empresa e a simulação não pode ocupar o que o plano real vai usar.
	numeracao := &nextOrderNumber
	if plan.IsScenario {
		numeracao = nil
	}

	// Dispara modos conforme PlanningTypes do plano.
	if len(plan.PlanningTypes) == 0 {
		plan.PlanningTypes = []string{"MRP"}
	}

	for _, pt := range plan.PlanningTypes {
		switch strings.ToUpper(strings.TrimSpace(pt)) {
		case "MIN_MAX":
			tItems, tOrders := s.calculateMinMax(ctx, planCode, params, snapshotMap, supplyMap, rulesMap, llcMap, itemCache, allowedItems, restrictedItems, numeracao, errs)
			totalItems += tItems
			totalOrders += tOrders
		case "REORDER_POINT":
			tItems, tOrders := s.calculateReorderPoint(ctx, planCode, params, snapshotMap, supplyMap, rulesMap, llcMap, itemCache, allowedItems, restrictedItems, numeracao, errs)
			totalItems += tItems
			totalOrders += tOrders
		case "KANBAN":
			tItems, tOrders := s.calculateKanban(ctx, planCode, params, snapshotMap, supplyMap, llcMap, itemCache, allowedItems, restrictedItems, numeracao, errs)
			totalItems += tItems
			totalOrders += tOrders
		case "MPS":
			if !noMRPDemand {
				tItems, tOrders := s.calculateMPS(ctx, planCode, params, plan, snapshotMap, rulesMap, supplyMap, llcMap, bomMap, itemCache, allowedItems, restrictedItems, demands, demandInputs, numeracao, excecoes, errs)
				totalItems += tItems
				totalOrders += tOrders
			}
		default: // MRP
			if !noMRPDemand {
				tItems, tOrders := s.calculateMRP(ctx, planCode, params, plan, snapshotMap, rulesMap, supplyMap, llcMap, bomMap, itemCache, allowedItems, restrictedItems, demands, demandInputs, numeracao, excecoes, errs)
				totalItems += tItems
				totalOrders += tOrders
			}
		}
	}

	// Pós-processamento: integração com máquinas.
	if err := s.processMachineIntegration(ctx, planCode, allCodeSlice); err != nil {
		errs["machine_integration"] = err.Error()
	}

	// Pós-processamento: geração de prioridades automáticas.
	if params.GerarPrioridadesOrdens {
		registrarErro(errs, "prioridades", s.processAutoPriority(ctx, planCode, params))
	}

	// Geração de mensagens de exceção.
	if s.SupplyPort != nil {
		if err := s.generateExceptionMessages(ctx, planCode, excecoes); err != nil {
			errs["exception_messages"] = err.Error()
		}
	}
	// Pegging fixo: de onde vem cada necessidade e o que a cobriu.
	if w, ok := s.MRPRepo.(peggingWriter); ok {
		if err := w.ReplacePegging(ctx, planCode, excecoes.pegging()); err != nil {
			errs["pegging"] = err.Error()
		}
	}
	if generateLLC && !plan.IsScenario {
		if err := s.MRPRepo.UpdateItemLLCs(ctx, llcMap); err != nil {
			errs["update_item_llc"] = err.Error()
		}
	}

	status := "COMPLETED"
	if len(errs) > 0 {
		status = "COMPLETED_WITH_ERRORS"
	}
	if err := s.PlanRepo.UpdateLastCalculated(ctx, planCode); err != nil {
		errs["update_last_calculated"] = err.Error()
		status = "COMPLETED_WITH_ERRORS"
	}

	return s.MRPRepo.FinishCalculation(ctx, log.Code, status, errs, totalItems, totalOrders)
}

// =============================================================================
// Interface pública — compatível com MRPService
// =============================================================================

func (s *MRPServiceImpl) CalculateNetRequirements(ctx context.Context, input *entity.MRPInput) (*entity.MRPOutput, error) {
	params, err := s.MRPRepo.LoadTypedPlanningParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("lendo os parâmetros de planejamento: %w", err)
	}
	if params == nil {
		params = entity.DefaultTypedPlanningParams()
	}
	snapshotMap := make(map[int64]*entity.StockSnapshot)
	snapshot, err := s.MRPRepo.GetStockSnapshot(ctx, input.ItemCode)
	if err != nil && !domainerr.EhNaoEncontrado(err) {
		return nil, fmt.Errorf("lendo o estoque do item %d: %w", input.ItemCode, err)
	}
	if snapshot != nil {
		snapshotMap[input.ItemCode] = snapshot
	}
	rulesMap := make(map[int64][]*entity.ConfiguredItemRule)
	rules, err := s.MRPRepo.GetConfiguredItemRules(ctx, input.ItemCode)
	if err != nil {
		return nil, fmt.Errorf("lendo as regras de item configurado %d: %w", input.ItemCode, err)
	}
	rulesMap[input.ItemCode] = rules
	return s.calcNetReqFast(ctx, input, snapshotMap, rulesMap, nil, make(map[int64]*cachedItemMRP), params)
}

func (s *MRPServiceImpl) ExplodeStructure(ctx context.Context, parentCode int64, mask string, quantity decimal.Decimal, level int) ([]*entity.MRPInput, error) {
	if level > 20 {
		return nil, nil
	}
	children, err := s.StructRepo.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("exploding structure for item %d: %w", parentCode, err)
	}
	params, err := s.MRPRepo.LoadTypedPlanningParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("lendo os parâmetros de planejamento: %w", err)
	}
	if params == nil {
		params = entity.DefaultTypedPlanningParams()
	}
	// Variáveis da configuração do pai, para as fórmulas de quantidade.
	vars := newMaskVarCache(s.MaskVars).vars(ctx, parentCode, mask)
	inputs := make([]*entity.MRPInput, 0, len(children))
	for _, child := range children {
		if !child.IsActive {
			continue
		}
		if child.ParentMask != nil && (mask == "" || *child.ParentMask != mask) {
			continue
		}
		perUnit, _ := child.ResolvedQuantity(vars)
		adjustedQty := applyLossFormula(quantity, perUnit, child.LossPercentage, params.FormulaPerdasEstrutura)
		inputs = append(inputs, &entity.MRPInput{
			ItemCode: child.ChildCode,
			Quantity: adjustedQty,
			LLC:      level,
		})
	}
	return inputs, nil
}

func (s *MRPServiceImpl) CalculateItemLLC(ctx context.Context, itemCode int64) (int, error) {
	llcMap, err := s.buildLLCMap(ctx, []int64{itemCode})
	if err != nil {
		return 0, err
	}
	return llcMap[itemCode], nil
}

func (s *MRPServiceImpl) GenerateLLC(ctx context.Context) error {
	return nil
}

func (s *MRPServiceImpl) buildLLCMap(ctx context.Context, rootItems []int64) (map[int64]int, error) {
	bomMap, err := s.StructRepo.LoadBOMForRoots(ctx, rootItems)
	if err != nil {
		return nil, err
	}
	return buildLLCFromBOM(bomMap, rootItems), nil
}

func (s *MRPServiceImpl) subtractWorkdays(ctx context.Context, from time.Time, days int) (time.Time, error) {
	if days <= 0 {
		return from, nil
	}
	return s.CalRepo.SubtractWorkdays(ctx, from, days)
}

func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// =============================================================================
// Modos de planejamento
// =============================================================================

// calculateMRP executa o cálculo MRP clássico (BFS nível por nível).
func (s *MRPServiceImpl) calculateMRP(
	ctx context.Context,
	planCode int64,
	params *entity.TypedPlanningParams,
	plan *planentity.ProductionPlan,
	snapshotMap map[int64]*entity.StockSnapshot,
	rulesMap map[int64][]*entity.ConfiguredItemRule,
	supplyMap map[int64][]ports.SupplyEntry,
	llcMap map[int64]int,
	bomMap map[int64][]*structentity.ItemStructure,
	itemCache map[int64]*cachedItemMRP,
	allowedItems map[int64]struct{},
	restrictedItems map[int64]struct{},
	demands []*inddemandentity.IndependentDemand,
	forecastInputs []*entity.MRPInput,
	nextOrderNumber *int64,
	excecoes *exceptionBasis,
	errs map[string]interface{},
) (int, int) {
	totalItems := 0
	totalOrders := 0
	// Cache das variáveis de configuração usadas pelas fórmulas de quantidade,
	// compartilhado por toda a explosão deste plano.
	maskVars := newMaskVarCache(s.MaskVars)

	// An order-item calculation must not introduce any additional independent source.
	var safetyInputs []*entity.MRPInput
	if plan.OrderItemCode == nil {
		var errSeguranca error
		safetyInputs, errSeguranca = s.generateSafetyStockDemands(ctx, params, snapshotMap, supplyMap, llcMap, itemCache, allowedItems, restrictedItems)
		registrarErro(errs, "estoque_seguranca", errSeguranca)
	}
	for _, si := range safetyInputs {
		if params.AgrupaDemandaEstoque {
			// Agrega à primeira demanda existente do item.
			merged := false
			for _, fi := range forecastInputs {
				if fi.ItemCode == si.ItemCode && fi.Mask == si.Mask && !fi.TechnicalAssistance {
					fi.Quantity = fi.Quantity.Add(si.Quantity)
					if si.NeedDate.Before(fi.NeedDate) {
						fi.NeedDate = si.NeedDate
					}
					merged = true
					break
				}
			}
			if !merged {
				forecastInputs = append(forecastInputs, si)
			}
		} else {
			forecastInputs = append(forecastInputs, si)
		}
	}

	// Recoleta root items (com safety stock adicionado).
	seen := make(map[int64]bool)
	var rootItems []int64
	for _, d := range demands {
		if !s.itemAllowed(allowedItems, d.ItemCode) {
			continue
		}
		if !seen[d.ItemCode] {
			seen[d.ItemCode] = true
			rootItems = append(rootItems, d.ItemCode)
		}
	}
	for _, fi := range forecastInputs {
		if !s.itemAllowed(allowedItems, fi.ItemCode) {
			continue
		}
		if !seen[fi.ItemCode] {
			seen[fi.ItemCode] = true
			rootItems = append(rootItems, fi.ItemCode)
		}
	}

	// Seed level 0.
	levelQueues := make(map[int][]*entity.MRPInput)
	for _, d := range demands {
		if !s.itemAllowed(allowedItems, d.ItemCode) {
			continue
		}
		mask := ""
		if d.Mask != nil {
			mask = *d.Mask
		}
		llc := llcMap[d.ItemCode]
		sourceCode := d.CodeDemand
		levelQueues[llc] = append(levelQueues[llc], &entity.MRPInput{
			PlanCode:   planCode,
			ItemCode:   d.ItemCode,
			Mask:       mask,
			Quantity:   d.Quantity,
			NeedDate:   d.DemandDate,
			LLC:        llc,
			DemandType: "INDEPENDENT_DEMAND",
			SourceCode: &sourceCode,
		})
	}
	for _, fi := range forecastInputs {
		if !s.itemAllowed(allowedItems, fi.ItemCode) {
			continue
		}
		llc := llcMap[fi.ItemCode]
		fi.LLC = llc
		levelQueues[llc] = append(levelQueues[llc], fi)
	}

	maxLevel := maxLLC(llcMap)

	for level := 0; level <= maxLevel; level++ {
		inputs, ok := levelQueues[level]
		if !ok {
			continue
		}
		for _, detailInput := range inputs {
			detailType := detailInput.DemandType
			if detailType == "" && detailInput.ParentItemCode != nil {
				detailType = "DEPENDENT_DEMAND"
			}
			if detailType == "" {
				detailType = "DEMAND"
			}
			// Cada necessidade bruta, com a origem, antes de agrupar por data:
			// é a base do pegging e das mensagens de exceção.
			pegKind, pegCode := pegOrigin(detailInput)
			excecoes.addRequirement(detailInput.ItemCode, detailInput.NeedDate, detailInput.Quantity, pegKind, pegCode)
			if err := s.MRPRepo.CreateProfileDetail(ctx, &entity.MRPProfileDetail{PlanCode: planCode, ItemCode: detailInput.ItemCode,
				NeedDate: detailInput.NeedDate, DetailType: detailType, SourceCode: detailInput.SourceCode,
				ParentItemCode: detailInput.ParentItemCode, Quantity: detailInput.Quantity}); err != nil {
				errs[fmt.Sprintf("profile_detail_%d_%s", detailInput.ItemCode, detailInput.NeedDate.Format("2006-01-02"))] = err.Error()
			}
		}
		for _, input := range aggregateInputs(inputs, plan.GroupSameDateOrders) {
			input.PlanCode = planCode
			input.LLC = level

			if _, blocked := restrictedItems[input.ItemCode]; blocked {
				errs[fmt.Sprintf("item_%d_restricted", input.ItemCode)] = "item has active restriction — skipped"
				continue
			}

			output, err := s.calcNetReqFast(ctx, input, snapshotMap, rulesMap, supplyMap, itemCache, params)
			if err != nil {
				errs[fmt.Sprintf("item_%d", input.ItemCode)] = err.Error()
				continue
			}
			firmSupplyUsed := firmSupplyForItem(supplyMap, input.ItemCode, input.NeedDate)
			advanceNettingState(input, output, snapshotMap, supplyMap)

			if _, errPerfil := s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        input.ItemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          output.Demand,
				// Necessidade líquida negativa = o estoque/firmes sobram: não há
				// ordem planejada. Antes o perfil gravava o negativo e o
				// "planejado" do item aparecia como −400.
				OrdersPlanned:  decimal.Max(decimal.Zero, output.NetRequirement),
				OrdersFirm:     firmSupplyUsed,
				StockProjected: output.StockProjected,
				LLC:            level,
				NeedDate:       input.NeedDate,
			}); errPerfil != nil {
				registrarErro(errs, "perfil", errPerfil)
			}
			totalItems++

			for _, suggestion := range output.PlannedOrders {
				suggestion.PlanCode = planCode

				cached, errItem := s.ensureItemCache(ctx, itemCache, input.ItemCode)
				registrarErro(errs, "cadastro_item", errItem)

				if !cached.ghost || params.ItensFantasmasGravar {
					created, createErr := s.createNumberedSuggestion(ctx, suggestion, nextOrderNumber)
					if createErr != nil {
						errs[fmt.Sprintf("suggestion_%d_%s", suggestion.ItemCode, suggestion.NeedDate.Format("2006-01-02"))] = createErr.Error()
						continue
					}
					totalOrders++
					if created != nil {
						suggestion.Code = created.Code
					}
					excecoes.addSuggestion(suggestion)
				}

				if suggestion.StartDate == nil || !suppliesItem(suggestion) {
					continue
				}

				if cached.ghost && !params.ItensFantasmasGravar {
					// Fantasma não gravado: a necessidade passa direto aos
					// filhos, que herdam a origem do pai no pegging.
					excecoes.skip[input.ItemCode] = true
					origemKind, origemCode := pegOrigin(input)
					children := explodeFromBOMWithVars(bomMap, input.ItemCode, input.Mask, suggestion.Quantity, level+1, params.FormulaPerdasEstrutura, maskVars.vars(ctx, input.ItemCode, input.Mask), *suggestion.StartDate)
					for _, child := range children {
						child.PlanCode = planCode
						child.NeedDate = *suggestion.StartDate
						child.ParentItemCode = &input.ItemCode
						child.PegKind, child.PegCode = origemKind, origemCode
						levelQueues[level+1] = append(levelQueues[level+1], child)
					}
					continue
				}

				children := explodeFromBOMWithVars(bomMap, input.ItemCode, input.Mask, suggestion.Quantity, level+1, params.FormulaPerdasEstrutura, maskVars.vars(ctx, input.ItemCode, input.Mask), *suggestion.StartDate)
				pai := suggestion.Code
				for _, child := range children {
					child.PlanCode = planCode
					child.NeedDate = *suggestion.StartDate
					child.ParentItemCode = &input.ItemCode
					// A necessidade do componente nasce desta sugestão: é a
					// ponte do pegging entre os níveis.
					child.PegKind, child.PegCode = entity.PegDemandSuggestion, &pai
					levelQueues[level+1] = append(levelQueues[level+1], child)
				}
			}
		}
	}

	return totalItems, totalOrders
}

// calculateMinMax implementa lógica de estoque mínimo-máximo.
func (s *MRPServiceImpl) calculateMinMax(
	ctx context.Context,
	planCode int64,
	params *entity.TypedPlanningParams,
	snapshotMap map[int64]*entity.StockSnapshot,
	supplyMap map[int64][]ports.SupplyEntry,
	rulesMap map[int64][]*entity.ConfiguredItemRule,
	llcMap map[int64]int,
	itemCache map[int64]*cachedItemMRP,
	allowedItems map[int64]struct{},
	restrictedItems map[int64]struct{},
	nextOrderNumber *int64,
	errs map[string]interface{},
) (int, int) {
	totalItems := 0
	totalOrders := 0

	planningExtras, errExtras := s.MRPRepo.ListItemPlanningExtras(ctx)
	registrarErro(errs, "load_planning_extras", errExtras)
	if planningExtras == nil {
		planningExtras = make(map[int64]*entity.ItemPlanningExtra)
	}

	for itemCode, snapshot := range snapshotMap {
		if !s.itemAllowed(allowedItems, itemCode) {
			continue
		}
		if _, blocked := restrictedItems[itemCode]; blocked {
			continue
		}

		// LMI = estoque mínimo do snapshot
		lmi := snapshot.SafetyStock
		if !lmi.IsPositive() {
			continue
		}

		// LMA = maximum_stock (ou minimum_stock * 3 como default)
		lma := lmi.Mul(decimal.NewFromInt(3))
		if extra, ok := planningExtras[itemCode]; ok && extra.MaximumStock.IsPositive() {
			lma = extra.MaximumStock
		}

		// QTDE = stock disponível (quantity - reserved)
		qtde := decimal.Max(decimal.Zero, snapshot.Quantity.Sub(snapshot.ReservedQty))

		// QTDP = firm supply (ordens firmes de compra)
		qtdp := firmSupplyForItem(supplyMap, itemCode, maxTime())

		qtdeTotal := qtde.Add(qtdp)

		if qtdeTotal.LessThanOrEqual(lmi) {
			qtc := lma.Sub(qtdeTotal)
			if !qtc.IsPositive() {
				continue
			}

			cached, errItem := s.ensureItemCache(ctx, itemCache, itemCode)
			registrarErro(errs, "cadastro_item", errItem)
			if cached == nil || cached.engineeringType == 2 { // DE_TERCEIRO
				continue
			}

			leadTimeDays, errPrazo := s.getLeadTimeDays(ctx, rulesMap, itemCode)
			registrarErro(errs, "prazo", errPrazo)
			needDate := today().AddDate(0, 0, leadTimeDays)
			startDate, errCalendario := s.subtractWorkdays(ctx, needDate, leadTimeDays)
			registrarErro(errs, "calendario", errCalendario)
			if startDate.IsZero() {
				startDate = needDate.AddDate(0, 0, -leadTimeDays)
			}

			llc := llcMap[itemCode]
			suggestion := &entity.PlannedOrderSuggestion{
				PlanCode:   planCode,
				ItemCode:   itemCode,
				Quantity:   qtc,
				NeedDate:   needDate,
				StartDate:  &startDate,
				OrderType:  "COMPRA",
				DemandType: "MIN_MAX",
				LLC:        llc,
			}
			if _, err := s.createNumberedSuggestion(ctx, suggestion, nextOrderNumber); err != nil {
				errs[fmt.Sprintf("min_max_suggestion_%d", itemCode)] = err.Error()
			} else {
				totalOrders++
			}

			if _, errPerfil := s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        itemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          qtc,
				OrdersPlanned:   qtc,
				OrdersFirm:      qtdp,
				StockProjected:  qtdeTotal.Sub(qtc),
				LLC:             llc,
				NeedDate:        needDate,
			}); errPerfil != nil {
				registrarErro(errs, "perfil", errPerfil)
			}
			totalItems++
		}
	}

	return totalItems, totalOrders
}

// calculateReorderPoint implementa lógica de ponto de reposição.
func (s *MRPServiceImpl) calculateReorderPoint(
	ctx context.Context,
	planCode int64,
	params *entity.TypedPlanningParams,
	snapshotMap map[int64]*entity.StockSnapshot,
	supplyMap map[int64][]ports.SupplyEntry,
	rulesMap map[int64][]*entity.ConfiguredItemRule,
	llcMap map[int64]int,
	itemCache map[int64]*cachedItemMRP,
	allowedItems map[int64]struct{},
	restrictedItems map[int64]struct{},
	nextOrderNumber *int64,
	errs map[string]interface{},
) (int, int) {
	totalItems := 0
	totalOrders := 0

	for itemCode, snapshot := range snapshotMap {
		if !s.itemAllowed(allowedItems, itemCode) {
			continue
		}
		if _, blocked := restrictedItems[itemCode]; blocked {
			continue
		}

		cached, errItem := s.ensureItemCache(ctx, itemCache, itemCode)
		registrarErro(errs, "cadastro_item", errItem)
		if cached == nil || cached.engineeringType == 2 {
			continue
		}
		if cached.reorderPoint == nil {
			continue
		}

		pr, err := cached.reorderPoint.Calculate()
		if err != nil || !pr.IsPositive() {
			continue
		}

		qtde := decimal.Max(decimal.Zero, snapshot.Quantity.Sub(snapshot.ReservedQty))

		if qtde.LessThanOrEqual(pr) {
			loteEconomico := pr.Mul(decimal.NewFromInt(2))
			leadTimeDays, errPrazo := s.getLeadTimeDays(ctx, rulesMap, itemCode)
			registrarErro(errs, "prazo", errPrazo)
			needDate := today().AddDate(0, 0, leadTimeDays)
			startDate, errCalendario := s.subtractWorkdays(ctx, needDate, leadTimeDays)
			registrarErro(errs, "calendario", errCalendario)
			if startDate.IsZero() {
				startDate = needDate.AddDate(0, 0, -leadTimeDays)
			}
			llc := llcMap[itemCode]

			suggestion := &entity.PlannedOrderSuggestion{
				PlanCode:   planCode,
				ItemCode:   itemCode,
				Quantity:   loteEconomico,
				NeedDate:   needDate,
				StartDate:  &startDate,
				OrderType:  "COMPRA",
				DemandType: "REORDER_POINT",
				LLC:        llc,
			}
			if _, err := s.createNumberedSuggestion(ctx, suggestion, nextOrderNumber); err != nil {
				errs[fmt.Sprintf("reorder_suggestion_%d", itemCode)] = err.Error()
			} else {
				totalOrders++
			}

			if _, errPerfil := s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        itemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          loteEconomico,
				OrdersPlanned:   loteEconomico,
				OrdersFirm:      decimal.Zero,
				StockProjected:  qtde.Sub(loteEconomico),
				LLC:             llc,
				NeedDate:        needDate,
			}); errPerfil != nil {
				registrarErro(errs, "perfil", errPerfil)
			}
			totalItems++
		}
	}

	return totalItems, totalOrders
}

// calculateKanban implementa lógica de cartões kanban.
func (s *MRPServiceImpl) calculateKanban(
	ctx context.Context,
	planCode int64,
	params *entity.TypedPlanningParams,
	snapshotMap map[int64]*entity.StockSnapshot,
	supplyMap map[int64][]ports.SupplyEntry,
	llcMap map[int64]int,
	itemCache map[int64]*cachedItemMRP,
	allowedItems map[int64]struct{},
	restrictedItems map[int64]struct{},
	nextOrderNumber *int64,
	errs map[string]interface{},
) (int, int) {
	totalItems := 0
	totalOrders := 0

	kanbanCards, err := s.MRPRepo.ListKanbanCards(ctx)
	if err != nil || len(kanbanCards) == 0 {
		return 0, 0
	}

	// Agrupa cartões por item.
	type cardGroup struct {
		cards []*entity.KanbanCardInfo
	}
	grouped := make(map[int64]*cardGroup)
	for _, k := range kanbanCards {
		if group, ok := grouped[k.ItemCode]; ok {
			group.cards = append(group.cards, k)
		} else {
			grouped[k.ItemCode] = &cardGroup{cards: []*entity.KanbanCardInfo{k}}
		}
	}

	for itemCode, group := range grouped {
		if !s.itemAllowed(allowedItems, itemCode) {
			continue
		}
		if _, blocked := restrictedItems[itemCode]; blocked {
			continue
		}

		qtde := decimal.Zero
		if snapshot, ok := snapshotMap[itemCode]; ok && snapshot != nil {
			qtde = decimal.Max(decimal.Zero, snapshot.Quantity.Sub(snapshot.ReservedQty))
		}

		for _, card := range group.cards {
			if qtde.GreaterThan(card.ReorderPoint) {
				continue
			}

			cached, errItem := s.ensureItemCache(ctx, itemCache, itemCode)
			registrarErro(errs, "cadastro_item", errItem)
			if cached == nil || cached.engineeringType == 2 {
				continue
			}

			qtc := card.QuantityPerCard.Mul(decimal.NewFromInt(int64(card.CardCount)))
			leadTimeDays, errPrazo := s.getLeadTimeDays(ctx, nil, itemCode)
			registrarErro(errs, "prazo", errPrazo)
			needDate := today().AddDate(0, 0, leadTimeDays)
			startDate, errCalendario := s.subtractWorkdays(ctx, needDate, leadTimeDays)
			registrarErro(errs, "calendario", errCalendario)
			if startDate.IsZero() {
				startDate = needDate.AddDate(0, 0, -leadTimeDays)
			}
			llc := llcMap[itemCode]

			suggestion := &entity.PlannedOrderSuggestion{
				PlanCode:   planCode,
				ItemCode:   itemCode,
				Quantity:   qtc,
				NeedDate:   needDate,
				StartDate:  &startDate,
				OrderType:  "COMPRA",
				DemandType: "KANBAN",
				LLC:        llc,
			}
			if _, err := s.createNumberedSuggestion(ctx, suggestion, nextOrderNumber); err != nil {
				errs[fmt.Sprintf("kanban_suggestion_%d", itemCode)] = err.Error()
			} else {
				totalOrders++
			}

			if _, errPerfil := s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        itemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          qtc,
				OrdersPlanned:   qtc,
				OrdersFirm:      decimal.Zero,
				StockProjected:  qtde.Sub(qtc),
				LLC:             llc,
				NeedDate:        needDate,
			}); errPerfil != nil {
				registrarErro(errs, "perfil", errPerfil)
			}
			totalItems++
			qtde = qtde.Sub(qtc)
		}
	}

	return totalItems, totalOrders
}

// calculateMPS carrega entradas do MPS como demandas e processa via MRP normal.
func (s *MRPServiceImpl) calculateMPS(
	ctx context.Context,
	planCode int64,
	params *entity.TypedPlanningParams,
	plan *planentity.ProductionPlan,
	snapshotMap map[int64]*entity.StockSnapshot,
	rulesMap map[int64][]*entity.ConfiguredItemRule,
	supplyMap map[int64][]ports.SupplyEntry,
	llcMap map[int64]int,
	bomMap map[int64][]*structentity.ItemStructure,
	itemCache map[int64]*cachedItemMRP,
	allowedItems map[int64]struct{},
	restrictedItems map[int64]struct{},
	demands []*inddemandentity.IndependentDemand,
	forecastInputs []*entity.MRPInput,
	nextOrderNumber *int64,
	excecoes *exceptionBasis,
	errs map[string]interface{},
) (int, int) {
	mpsItems, err := s.MRPRepo.ListMPSItems(ctx, planCode)
	if err != nil || len(mpsItems) == 0 {
		return 0, 0
	}

	// Converte entradas MPS não firmadas em MRPInput e adiciona ao forecast.
	for _, mi := range mpsItems {
		if !s.itemAllowed(allowedItems, mi.ItemCode) {
			continue
		}
		needDate := mpsPeriodToDate(mi.PeriodType, mi.PeriodValue, mi.Year)
		forecastInputs = append(forecastInputs, &entity.MRPInput{
			PlanCode: planCode,
			ItemCode: mi.ItemCode,
			Mask:     mi.Mask,
			Quantity: mi.Quantity,
			NeedDate: needDate,
			PegKind:  entity.PegDemandMPS,
		})
	}

	return s.calculateMRP(ctx, planCode, params, plan, snapshotMap, rulesMap, supplyMap,
		llcMap, bomMap, itemCache, allowedItems, restrictedItems,
		demands, forecastInputs, nextOrderNumber, excecoes, errs)
}

// =============================================================================
// calcNetReqFast — cálculo de necessidade líquida otimizado
// =============================================================================

// calcNetReqFast computes net requirements using pre-loaded in-memory maps.
// Time-phased netting: firm supply entries with ArrivalDate <= NeedDate are
// subtracted from the gross requirement before generating a new planned order.
func (s *MRPServiceImpl) calcNetReqFast(
	ctx context.Context,
	input *entity.MRPInput,
	snapshotMap map[int64]*entity.StockSnapshot,
	rulesMap map[int64][]*entity.ConfiguredItemRule,
	supplyMap map[int64][]ports.SupplyEntry,
	itemCache map[int64]*cachedItemMRP,
	params *entity.TypedPlanningParams,
) (*entity.MRPOutput, error) {

	// 1. Available stock from snapshot.
	availableStock := decimal.Zero
	if snapshot, ok := snapshotMap[input.ItemCode]; ok && snapshot != nil {
		avail := snapshot.Quantity.Sub(snapshot.ReservedQty).Sub(snapshot.SafetyStock)
		if avail.IsPositive() {
			availableStock = avail
		}
	}

	// 2. Time-phased netting.
	firmSupply := decimal.Zero
	if supplyMap != nil {
		firmSupply = firmSupplyForItem(supplyMap, input.ItemCode, input.NeedDate)
	}

	// 3. Configured rules (lead time, minimum lot) — suporta EQUAL, DIFFERENT, RANGE.
	leadTimeDays := 0
	minLot := decimal.Zero
	for _, rule := range rulesMap[input.ItemCode] {
		if !s.ruleMatchesMask(rule, input.Mask) {
			continue
		}
		switch rule.FieldName {
		case "lead_time":
			if v, err := strconv.Atoi(rule.RuleValue); err == nil {
				leadTimeDays = v
			}
		case "lote_minimo":
			if v, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(rule.RuleValue), ",", ".")); err == nil {
				minLot = v
			}
		}
	}

	// 4. Net requirement.
	totalCoverage := availableStock.Add(firmSupply)
	netReq := input.Quantity.Sub(totalCoverage)
	stockProjected := totalCoverage.Sub(input.Quantity)

	output := &entity.MRPOutput{
		ItemCode:       input.ItemCode,
		LLC:            input.LLC,
		Demand:         input.Quantity,
		StockProjected: stockProjected,
		NetRequirement: netReq,
	}

	if !netReq.IsPositive() {
		return output, nil
	}

	// 5. Item-type lookup — cached (também traz a política de lote do item).
	cachedItem, err := s.ensureItemCache(ctx, itemCache, input.ItemCode)
	if err != nil {
		return nil, err
	}
	if cachedItem == nil {
		cachedItem = &cachedItemMRP{}
	}

	// Política de lote: a regra configurada tem precedência sobre o cadastro do
	// item, porque é o ajuste fino por item/máscara.
	if !minLot.IsPositive() {
		minLot = cachedItem.minimumLot
	}
	netReq = aplicaPoliticaDeLote(netReq, minLot, cachedItem.multipleLot)

	orderType := "FABRICACAO"
	demandType := input.DemandType
	if demandType == "" {
		demandType = "INDEPENDENTE"
	}
	if input.ParentItemCode != nil {
		demandType = "DEPENDENTE"
	} else if input.InterFactory {
		demandType = "INTER_FACTORY"
	}

	cached := cachedItem

	switch cached.engineeringType {
	case 1: // COMPRADO
		orderType = "COMPRA"
	case 2: // DE_TERCEIRO — no order generated
		return output, nil
	}
	if cached.typeMRP != 0 { // PROJETO — planned manually
		return output, nil
	}

	// 6. Start date = need date minus lead time.
	startDate, err := s.subtractWorkdays(ctx, input.NeedDate, leadTimeDays)
	if err != nil {
		startDate = input.NeedDate.AddDate(0, 0, -leadTimeDays)
	}

	output.NetRequirement = netReq
	mainOrder := &entity.PlannedOrderSuggestion{
		PlanCode:             input.PlanCode,
		ItemCode:             input.ItemCode,
		Mask:                 input.Mask,
		Quantity:             netReq,
		NeedDate:             input.NeedDate,
		StartDate:            &startDate,
		OrderType:            orderType,
		DemandType:           demandType,
		ParentItemCode:       input.ParentItemCode,
		LLC:                  input.LLC,
		WarehouseCode:        input.WarehouseCode,
		InterFactory:         input.InterFactory,
		SourceEnterpriseCode: input.SourceEnterpriseCode,
		AutoRelease:          input.AutoRelease,
	}
	if shouldPlanTechnicalAssistance(input, params) {
		mainOrder.OrderType = "TECHNICAL_ASSISTANCE"
	}
	output.PlannedOrders = []*entity.PlannedOrderSuggestion{mainOrder}

	// When a FABRICACAO order is planned and the item has external/third-party
	// route operations, generate a SERVICO order for each external op so the
	// planner knows a purchase order for the service is required.
	if orderType == "FABRICACAO" && s.RoutingRepo != nil {
		// Falha ao ler as operações externas não pode virar "não há operação
		// externa": o MRP deixaria de sugerir a ordem de serviço em silêncio.
		extOps, err := s.RoutingRepo.GetExternalOpsByItem(ctx, input.ItemCode)
		if err != nil {
			return nil, fmt.Errorf("lendo as operações externas do item %d: %w", input.ItemCode, err)
		}
		for _, op := range extOps {
			note := fmt.Sprintf("Op. externa: %s (%.2fh)", op.OperationName, op.EffectiveHours)
			remittance := op.RemittanceType
			if remittance == "" {
				remittance = "DEMAND_ITEMS"
			}
			serviceOrder := &entity.PlannedOrderSuggestion{
				PlanCode:         input.PlanCode,
				ItemCode:         input.ItemCode,
				Mask:             input.Mask,
				Quantity:         netReq,
				NeedDate:         input.NeedDate,
				StartDate:        &startDate,
				OrderType:        "SERVICO",
				DemandType:       "EXTERNA",
				ParentItemCode:   input.ParentItemCode,
				LLC:              input.LLC,
				Notes:            &note,
				RouteOperationID: &op.RouteOpID,
				OperationID:      &op.OperationID,
				SupplierCode:     op.SupplierID,
				ServiceItemCode:  op.ServiceItemCode,
				RemittanceType:   &remittance,
			}
			output.PlannedOrders = append(output.PlannedOrders, serviceOrder)
		}
	}

	return output, nil
}

// itemLabeler devolve o código de negócio dos itens da empresa da sessão, para
// o texto das mensagens de exceção.
type itemLabeler interface {
	ItemBusinessCodes(ctx context.Context, codes []int64) (map[int64]string, error)
}

// generateExceptionMessages compara cada ordem firme com a necessidade que ela
// de fato cobre (pegging) e grava as mensagens de ação. Ver exceptions.go.
// Devolve o primeiro erro de gravação (as demais mensagens seguem gravando).
func (s *MRPServiceImpl) generateExceptionMessages(ctx context.Context, planCode int64, basis *exceptionBasis) error {
	var primeiro error
	if l, ok := s.MRPRepo.(itemLabeler); ok {
		codes := make([]int64, 0, len(basis.supply))
		for item := range basis.supply {
			codes = append(codes, item)
		}
		// Sem o rótulo a mensagem sai com o código interno; o erro é devolvido
		// mesmo assim, como os de gravação.
		if basis.labels, primeiro = l.ItemBusinessCodes(ctx, codes); primeiro != nil {
			primeiro = fmt.Errorf("lendo os códigos dos itens das exceções: %w", primeiro)
		}
	}
	for _, m := range basis.messages() {
		m.PlanCode = planCode
		if err := s.MRPRepo.CreateExceptionMessage(ctx, m); err != nil && primeiro == nil {
			primeiro = err
		}
	}
	return primeiro
}

// =============================================================================
// Pós-processamento: integração com máquinas e prioridades
// =============================================================================

// processMachineIntegration assigns machines and creates machine_schedule
// entries for FABRICACAO-type planned orders.
// processAutoPriority assigns priority codes to planned orders based on
// order_priorities rules and the DiasPrioridades window.
func (s *MRPServiceImpl) processAutoPriority(ctx context.Context, planCode int64, params *entity.TypedPlanningParams) error {
	priorities, err := s.MRPRepo.ListAllOrderPriorities(ctx)
	if err != nil {
		return fmt.Errorf("lendo as prioridades de ordem: %w", err)
	}
	if len(priorities) == 0 {
		return nil
	}

	suggestions, err := s.MRPRepo.ListSuggestionsByPlan(ctx, planCode)
	if err != nil {
		return fmt.Errorf("lendo as sugestões para priorizar: %w", err)
	}
	if len(suggestions) == 0 {
		return nil
	}

	cutoffDate := today().AddDate(0, 0, params.DiasPrioridades)

	for _, sug := range suggestions {
		if sug.StartDate == nil {
			continue
		}
		if sug.StartDate.After(cutoffDate) {
			continue
		}
		priority := findPriorityForQuantity(priorities, sug.Quantity)
		if priority != "" {
			sug.Priority = &priority
			if err := s.MRPRepo.UpdatePlannedOrderPriority(ctx, sug.Code, priority); err != nil {
				return fmt.Errorf("gravando a prioridade da sugestão %d: %w", sug.Code, err)
			}
		}
	}
	return nil
}

func findPriorityForQuantity(priorities []*orderpriority.OrderPriority, quantity decimal.Decimal) string {
	for _, p := range priorities {
		if quantity.GreaterThanOrEqual(dq(p.IntervalStart)) && quantity.LessThanOrEqual(dq(p.IntervalEnd)) {
			return p.Priority
		}
	}
	return ""
}

// =============================================================================
// Helpers — regras configuradas (EQUAL, DIFFERENT, RANGE)
// =============================================================================

// ruleMatchesMask avalia se uma regra configurada se aplica ao mask informado.
// Suporta EQUAL, DIFFERENT e RANGE.
func (s *MRPServiceImpl) ruleMatchesMask(rule *entity.ConfiguredItemRule, mask string) bool {
	switch strings.ToUpper(rule.RuleType) {
	case "EQUAL":
		return mask == rule.RuleValue
	case "DIFFERENT":
		return mask != rule.RuleValue
	case "RANGE":
		parts := strings.SplitN(rule.RuleValue, "-", 2)
		if len(parts) != 2 {
			return false
		}
		minVal, errMin := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		maxVal, errMax := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errMin != nil || errMax != nil {
			return false
		}
		maskVal, err := strconv.ParseFloat(mask, 64)
		if err != nil {
			return false
		}
		return maskVal >= minVal && maskVal <= maxVal
	default:
		return true // rule without type restriction applies always
	}
}

// getLeadTimeDays returns the lead time for an item in working days.
// Priority: routing critical-path (hours → days at 8h/day) > configured rule > 0.
func (s *MRPServiceImpl) getLeadTimeDays(ctx context.Context, rulesMap map[int64][]*entity.ConfiguredItemRule, itemCode int64) (int, error) {
	// 0. Prazo do cenário de simulação substitui roteiro e regras.
	for _, rule := range rulesMap[itemCode] {
		if rule.TableType == scenarioRuleTable && rule.FieldName == "lead_time" {
			if v, err := strconv.Atoi(rule.RuleValue); err == nil {
				return v, nil
			}
		}
	}
	// 1. Try routing lead time (critical-path through operation network).
	if s.RoutingRepo != nil {
		// Item sem roteiro cai na regra; falha de leitura é erro (antes caía
		// na regra em silêncio e o prazo saía errado).
		route, err := s.RoutingRepo.GetRouteForItem(ctx, itemCode, "")
		if err != nil && !domainerr.EhNaoEncontrado(err) {
			return 0, fmt.Errorf("lendo o roteiro do item %d: %w", itemCode, err)
		}
		if err == nil && route != nil {
			ops, err := s.RoutingRepo.GetRouteOperations(ctx, route.ID)
			if err != nil {
				return 0, fmt.Errorf("lendo as etapas do roteiro do item %d: %w", itemCode, err)
			}
			if len(ops) > 0 {
				edges, err := s.RoutingRepo.GetNetworkEdges(ctx, route.ID)
				if err != nil {
					return 0, fmt.Errorf("lendo a rede do roteiro do item %d: %w", itemCode, err)
				}
				// qty=1: item-level planning lead time (per unit / per base batch).
				// Shared CPM implementation with the routing lead-time use case.
				totalHours := routingentity.CriticalPath(ops, edges, 1.0).TotalHours
				if totalHours > 0 {
					days := int(math.Ceil(totalHours / 8.0)) // 8h = 1 dia útil
					if days > 0 {
						return days, nil
					}
				}
			}
		}
	}
	// 2. Fallback: configured rule.
	maxLead := 0
	for _, rule := range rulesMap[itemCode] {
		if rule.FieldName == "lead_time" {
			if v, err := strconv.Atoi(rule.RuleValue); err == nil && v > maxLead {
				maxLead = v
			}
		}
	}
	return maxLead, nil
}

// =============================================================================
// Helpers — estoque de segurança
// =============================================================================

// generateSafetyStockDemands cria MRPInput para demanda de estoque de segurança
// conforme param 4 (GERAR_DEMANDA_SEGURANCA_TODOS) e param 6.
func (s *MRPServiceImpl) generateSafetyStockDemands(
	ctx context.Context,
	params *entity.TypedPlanningParams,
	snapshotMap map[int64]*entity.StockSnapshot,
	supplyMap map[int64][]ports.SupplyEntry,
	llcMap map[int64]int,
	itemCache map[int64]*cachedItemMRP,
	allowedItems map[int64]struct{},
	restrictedItems map[int64]struct{},
) ([]*entity.MRPInput, error) {
	var result []*entity.MRPInput
	baseDate := today()

	for itemCode, snapshot := range snapshotMap {
		if !s.itemAllowed(allowedItems, itemCode) {
			continue
		}
		if _, blocked := restrictedItems[itemCode]; blocked {
			continue
		}
		if !snapshot.SafetyStock.IsPositive() {
			continue
		}

		if !params.GerarDemandaSegurancaTodos {
			// Only for items with movimentação: verifica consumo médio mensal.
			cached, errItem := s.ensureItemCache(ctx, itemCache, itemCode)
			if errItem != nil {
				return nil, errItem
			}
			if cached == nil {
				continue
			}
		}

		// Calcula data da necessidade.
		cached, errItem := s.ensureItemCache(ctx, itemCache, itemCode)
		if errItem != nil {
			return nil, errItem
		}
		leadTimeDays := 0
		needDate := baseDate
		if params.DataNecessidadeEstoqueFuturo {
			needDate = baseDate.AddDate(0, 0, leadTimeDays+1)
		} else {
			needDate = baseDate.AddDate(0, 0, -leadTimeDays-1)
		}
		_ = cached // cached may be used for future enhancements

		llc := llcMap[itemCode]
		result = append(result, &entity.MRPInput{
			PlanCode: 0,
			ItemCode: itemCode,
			Quantity: snapshot.SafetyStock,
			NeedDate: needDate,
			LLC:      llc,
			PegKind:  entity.PegDemandSafetyStock,
		})
	}

	return result, nil
}

// =============================================================================
// Helpers — item cache e divisões
// =============================================================================

// ensureItemCache garante que o item esteja no cache, carregando do DB se necessário.
func (s *MRPServiceImpl) ensureItemCache(ctx context.Context, itemCache map[int64]*cachedItemMRP, itemCode int64) (*cachedItemMRP, error) {
	if cached, ok := itemCache[itemCode]; ok {
		return cached, nil
	}

	// Item inexistente planeja com os padrões; falha de leitura é erro — antes
	// o item era planejado sem lote mínimo, fantasma nem ponto de pedido.
	var item *itementity.Item
	if code, err := valueobject.NewItemCode(itemCode); err == nil {
		item, err = s.ItemRepo.FindItemByCode(ctx, code)
		if err != nil && !domainerr.EhNaoEncontrado(err) {
			return &cachedItemMRP{}, fmt.Errorf("lendo o cadastro do item %d: %w", itemCode, err)
		}
	}

	cached := &cachedItemMRP{}
	if item != nil {
		cached.engineeringType = int(item.Engineering.Type)
		cached.typeMRP = int(item.Planning.TypeMRP)
		cached.ghost = item.Planning.Ghost
		cached.reorderPoint = item.Planning.ReorderPoint
		cached.minimumLot = item.Planning.MinimumLot
		cached.multipleLot = item.Planning.MultipleLot
	}
	itemCache[itemCode] = cached
	return cached, nil
}

// salesDivisionRef é uma versão leve de SalesDivision para uso no MRP.
type salesDivisionRef struct {
	code                  int64
	isTechnicalAssistance bool
}

// loadDivisionMap carrega divisões de vendas para os itens fornecidos.
func (s *MRPServiceImpl) loadDivisionMap(ctx context.Context, itemCodes []int64) map[int64]*salesDivisionRef {
	sdMap, err := s.MRPRepo.ListItemSalesDivisions(ctx, itemCodes)
	if err != nil || sdMap == nil {
		return nil
	}
	result := make(map[int64]*salesDivisionRef, len(sdMap))
	for code, sd := range sdMap {
		result[code] = &salesDivisionRef{
			code:                  code,
			isTechnicalAssistance: sd.IsTechnicalAssistance,
		}
	}
	return result
}

// =============================================================================
// Pure in-memory helpers (no DB calls)
// =============================================================================

// buildLLCFromBOM computes LLC values via DFS over the pre-loaded adjacency map.
func buildLLCFromBOM(bomMap map[int64][]*structentity.ItemStructure, rootItems []int64) map[int64]int {
	llcMap := make(map[int64]int)

	var assignLLC func(itemCode int64, level int)
	assignLLC = func(itemCode int64, level int) {
		if level > 20 {
			return
		}
		if current, exists := llcMap[itemCode]; exists && current >= level {
			return
		}
		llcMap[itemCode] = level
		for _, child := range structentity.SelectPrimarySubstituteComponents(bomMap[itemCode]) {
			// Co-products are outputs, not lower-level components — skip so they
			// don't get a misleading (deeper) low-level code.
			if child.IsCoproduct {
				continue
			}
			assignLLC(child.ChildCode, level+1)
		}
	}

	for _, itemCode := range rootItems {
		assignLLC(itemCode, 0)
	}
	return llcMap
}

// explodeFromBOM expands one BOM level using the pre-loaded adjacency map.
// Mantém compatibilidade com código existente (usa fórmula default 1).
func explodeFromBOM(
	bomMap map[int64][]*structentity.ItemStructure,
	parentCode int64,
	mask string,
	quantity decimal.Decimal,
	level int,
) []*entity.MRPInput {
	return explodeFromBOMWithFormula(bomMap, parentCode, mask, quantity, level, 1)
}

// explodeFromBOMWithVars expande um nível da estrutura avaliando também a
// fórmula de quantidade de cada componente com as variáveis da configuração.
func explodeFromBOMWithVars(
	bomMap map[int64][]*structentity.ItemStructure,
	parentCode int64,
	mask string,
	quantity decimal.Decimal,
	level int,
	formula int,
	vars map[string]float64,
	necessidade time.Time,
) []*entity.MRPInput {
	return explode(bomMap, parentCode, mask, quantity, level, formula, vars, necessidade)
}

// explodeFromBOMWithFormula expands one BOM level with configurable loss formula.
func explodeFromBOMWithFormula(
	bomMap map[int64][]*structentity.ItemStructure,
	parentCode int64,
	mask string,
	quantity decimal.Decimal,
	level int,
	formula int,
) []*entity.MRPInput {
	return explode(bomMap, parentCode, mask, quantity, level, formula, nil, time.Time{})
}

func explode(
	bomMap map[int64][]*structentity.ItemStructure,
	parentCode int64,
	mask string,
	quantity decimal.Decimal,
	level int,
	formula int,
	vars map[string]float64,
	necessidade time.Time,
) []*entity.MRPInput {
	if level > 20 {
		return nil
	}
	children := bomMap[parentCode]
	applicable := make([]*structentity.ItemStructure, 0, len(children))
	for _, child := range children {
		if child.ParentMask != nil && (mask == "" || *child.ParentMask != mask) {
			continue
		}
		// A vigência é avaliada na data em que o componente será consumido, e
		// antes da escolha do alternativo: se o material principal venceu, a
		// necessidade vai para o próximo da prioridade em vez de sumir. Data
		// zero = sem filtro (chamadas que não têm data de necessidade).
		if !necessidade.IsZero() && !child.VigenteEm(necessidade) {
			continue
		}
		applicable = append(applicable, child)
	}

	selectedChildren := structentity.SelectPrimarySubstituteComponents(applicable)
	inputs := make([]*entity.MRPInput, 0, len(selectedChildren))
	for _, child := range selectedChildren {
		// Co-products/by-products/scrap are OUTPUTS, not consumed inputs → no demand.
		if child.IsCoproduct {
			continue
		}
		// Fixed-quantity components are consumed once per order (per lot), not scaled
		// by the parent quantity; run the loss formula against a base of 1.
		base := quantity
		if child.IsFixedQty {
			base = decimal.NewFromInt(1)
		}
		// A quantidade por unidade do pai pode vir de fórmula (ex.:
		// "2*(COMPRIMENTO/1000)"); sem fórmula avaliável vale a quantidade fixa.
		perUnit, _ := child.ResolvedQuantity(vars)
		adjustedQty := applyLossFormula(base, perUnit, child.LossPercentage, formula)
		inputs = append(inputs, &entity.MRPInput{
			ItemCode: child.ChildCode,
			Quantity: adjustedQty,
			LLC:      level,
		})
	}
	return inputs
}

// applyLossFormula calcula a quantidade ajustada conforme a fórmula de perdas.
// Fórmula 1: Qty = QtdPai * QtdComponente * (1 + %Perda/100)
// Fórmula 2: Qty = QtdPai * QtdComponente / (1 - %Perda/100)
// Fórmula 3: Qty = QtdPai * QtdComponente (ignora perda)
func applyLossFormula(parentQty decimal.Decimal, childQtyPer, lossPercentage float64, formula int) decimal.Decimal {
	return quantidadeComPerda(parentQty.Mul(dq(childQtyPer)), lossPercentage, formula)
}

func collectAllItemCodes(bomMap map[int64][]*structentity.ItemStructure, roots []int64) []int64 {
	seen := make(map[int64]bool, len(roots)+len(bomMap))
	for _, code := range roots {
		seen[code] = true
	}
	for parent, children := range bomMap {
		seen[parent] = true
		for _, child := range children {
			seen[child.ChildCode] = true
		}
	}
	codes := make([]int64, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	return codes
}

func firmSupplyForItem(supplyMap map[int64][]ports.SupplyEntry, itemCode int64, needDate time.Time) decimal.Decimal {
	total := decimal.Zero
	for _, entry := range supplyMap[itemCode] {
		if !entry.ArrivalDate.After(needDate) {
			total = total.Add(entry.Quantity)
		}
	}
	return total
}

func shouldPlanTechnicalAssistance(input *entity.MRPInput, params *entity.TypedPlanningParams) bool {
	return input != nil && params != nil && input.TechnicalAssistance &&
		input.DemandType == "SALES_ORDER" && input.ParentItemCode == nil &&
		input.WarehouseCode != nil && params.TrataAssistenciaTecnica &&
		params.ObrigarControleEstoqueTerceiros
}

func (s *MRPServiceImpl) createNumberedSuggestion(ctx context.Context, suggestion *entity.PlannedOrderSuggestion, nextOrderNumber *int64) (*entity.PlannedOrderSuggestion, error) {
	if nextOrderNumber == nil {
		// Cálculo de cenário: sugestão sem número de ordem.
		suggestion.OrderNumber = nil
		return s.MRPRepo.CreatePlannedOrderSuggestion(ctx, suggestion)
	}
	if *nextOrderNumber <= 0 {
		return nil, domainerr.NewValidation("próximo número de ordem inválido")
	}
	// Número ocupado (ordem planejada ativa ou sugestão de outro plano) não é
	// erro do cálculo: a sugestão pega o próximo livre. Antes ela se perdia e o
	// plano terminava "com erros" sem nenhuma ordem quando o número inicial
	// repetia o de uma rodada anterior.
	const tentativas = 10000
	for i := 0; i < tentativas; i++ {
		number := *nextOrderNumber
		suggestion.OrderNumber = &number
		*nextOrderNumber++
		created, err := s.MRPRepo.CreatePlannedOrderSuggestion(ctx, suggestion)
		if errors.Is(err, mrprepo.ErrOrderNumberInUse) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return created, nil
	}
	return nil, domainerr.NewValidation(fmt.Sprintf("nenhum número de ordem livre em %d tentativas a partir de %d", tentativas, *nextOrderNumber-tentativas))
}

// suppliesItem diz se a sugestão traz o item para o estoque. A ordem de
// SERVIÇO é a compra da operação externa do MESMO item que a ordem de
// fabricação já cobre: não é item novo nem consome componentes. Antes ela era
// somada como suprimento (a necessidade seguinte do item saía menor) e
// explodia a estrutura de novo (os componentes eram pedidos em dobro).
func suppliesItem(s *entity.PlannedOrderSuggestion) bool {
	return s != nil && s.OrderType != "SERVICO"
}

// advanceNettingState prevents stock and firm receipts already consumed by an
// earlier demand from being reused by the next demand of the same item.
func advanceNettingState(input *entity.MRPInput, output *entity.MRPOutput, snapshotMap map[int64]*entity.StockSnapshot, supplyMap map[int64][]ports.SupplyEntry) {
	planned := decimal.Zero
	for _, order := range output.PlannedOrders {
		if suppliesItem(order) {
			planned = planned.Add(order.Quantity)
		}
	}
	snapshotMap[input.ItemCode] = &entity.StockSnapshot{
		ItemCode: input.ItemCode, Quantity: decimal.Max(decimal.Zero, output.StockProjected.Add(planned)), SnapshotDate: input.NeedDate,
	}
	remaining := supplyMap[input.ItemCode][:0]
	for _, supply := range supplyMap[input.ItemCode] {
		if supply.ArrivalDate.After(input.NeedDate) {
			remaining = append(remaining, supply)
		}
	}
	supplyMap[input.ItemCode] = remaining
}

func maxTime() time.Time {
	return time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
}

func maxLLC(llcMap map[int64]int) int {
	max := 0
	for _, v := range llcMap {
		if v > max {
			max = v
		}
	}
	return max
}

// aggregateInputs only merges entries sharing item, mask and need date when
// the production plan explicitly requests same-date grouping.
func aggregateInputs(inputs []*entity.MRPInput, groupSameDate bool) []*entity.MRPInput {
	if !groupSameDate {
		result := make([]*entity.MRPInput, 0, len(inputs))
		for _, input := range inputs {
			cp := *input
			result = append(result, &cp)
		}
		sort.SliceStable(result, func(i, j int) bool { return result[i].NeedDate.Before(result[j].NeedDate) })
		return result
	}
	type key struct {
		itemCode             int64
		mask                 string
		needDate             string
		warehouseCode        int64
		technicalAssistance  bool
		sourceEnterpriseCode int64
	}
	agg := make(map[key]*entity.MRPInput, len(inputs))
	for _, inp := range inputs {
		warehouseCode := int64(0)
		if inp.TechnicalAssistance && inp.WarehouseCode != nil {
			warehouseCode = *inp.WarehouseCode
		}
		sourceEnterpriseCode := int64(0)
		if inp.InterFactory && inp.SourceEnterpriseCode != nil {
			sourceEnterpriseCode = *inp.SourceEnterpriseCode
		}
		k := key{inp.ItemCode, inp.Mask, inp.NeedDate.Format("2006-01-02"), warehouseCode, inp.TechnicalAssistance, sourceEnterpriseCode}
		if existing, ok := agg[k]; ok {
			existing.Quantity = existing.Quantity.Add(inp.Quantity)
		} else {
			cp := *inp
			agg[k] = &cp
		}
	}
	result := make([]*entity.MRPInput, 0, len(agg))
	for _, v := range agg {
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].NeedDate.Before(result[j].NeedDate)
	})
	return result
}

// ---------- Production Plan helpers ----------

func (s *MRPServiceImpl) filterDemandsFromDate(
	demands []*inddemandentity.IndependentDemand,
	params map[string]interface{},
) []*inddemandentity.IndependentDemand {
	raw, ok := params["from_date"]
	if !ok {
		return demands
	}
	fromStr, _ := raw.(string)
	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return demands
	}
	filtered := demands[:0]
	for _, d := range demands {
		if !d.DemandDate.Before(from) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (s *MRPServiceImpl) resolveAllowedItemSet(ctx context.Context, plan *planentity.ProductionPlan) (map[int64]struct{}, error) {
	if plan.Classification == nil || plan.ClassItemCodes == nil || strings.TrimSpace(*plan.ClassItemCodes) == "" {
		return nil, nil
	}
	parts := strings.Split(*plan.ClassItemCodes, ",")
	classCodes := make([]string, 0, len(parts))
	for _, part := range parts {
		if code := strings.TrimSpace(part); code != "" {
			classCodes = append(classCodes, code)
		}
	}
	itemCodes, err := s.MRPRepo.ResolveClassificationItemCodes(ctx, strings.TrimSpace(*plan.Classification), classCodes)
	if err != nil {
		return nil, err
	}
	set := make(map[int64]struct{}, len(itemCodes))
	for _, code := range itemCodes {
		set[code] = struct{}{}
	}
	return set, nil
}

func (s *MRPServiceImpl) itemAllowed(allowed map[int64]struct{}, itemCode int64) bool {
	if allowed == nil {
		return true
	}
	_, ok := allowed[itemCode]
	return ok
}

// ---------- Sales Forecast → MRP demand ----------

func (s *MRPServiceImpl) loadForecastDemands(ctx context.Context, planCode int64) ([]*entity.MRPInput, error) {
	if s.ForecastRepo == nil {
		return nil, nil
	}
	year := time.Now().Year()
	forecasts, err := s.ForecastRepo.ListForecasts(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("lendo a previsão de vendas de %d: %w", year, err)
	}
	nextYear, err := s.ForecastRepo.ListForecasts(ctx, year+1)
	if err != nil {
		return nil, fmt.Errorf("lendo a previsão de vendas de %d: %w", year+1, err)
	}
	forecasts = append(forecasts, nextYear...)

	result := make([]*entity.MRPInput, 0, len(forecasts))
	for _, f := range forecasts {
		needDate := mrpWeekToDate(f.Year, f.Week)
		mask := ""
		if f.Mask != nil {
			mask = *f.Mask
		}
		result = append(result, &entity.MRPInput{
			PlanCode:   planCode,
			ItemCode:   f.ItemCode,
			Mask:       mask,
			Quantity:   dq(f.Quantity),
			NeedDate:   needDate,
			DemandType: "FORECAST",
		})
	}
	return result, nil
}

func mrpWeekToDate(year, week int) time.Time {
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, -weekday+1)
	return monday.AddDate(0, 0, (week-1)*7)
}

func mpsPeriodToDate(periodType string, periodValue, year int) time.Time {
	switch strings.ToUpper(periodType) {
	case "WEEK":
		return mrpWeekToDate(year, periodValue)
	case "MONTH":
		return time.Date(year, time.Month(periodValue), 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
}

// aplicaPoliticaDeLote arredonda a necessidade líquida para a política de lote
// do item: nunca abaixo do lote mínimo e sempre num múltiplo do lote múltiplo.
//
// É a ordem que SAP e FoccoERP usam — primeiro o mínimo, depois o múltiplo —
// porque arredondar para o múltiplo antes poderia devolver uma quantidade
// abaixo do mínimo.
func aplicaPoliticaDeLote(necessidade, loteMinimo, loteMultiplo decimal.Decimal) decimal.Decimal {
	if !necessidade.IsPositive() {
		return necessidade
	}
	if loteMinimo.IsPositive() && necessidade.LessThan(loteMinimo) {
		necessidade = loteMinimo
	}
	if loteMultiplo.IsPositive() {
		necessidade = necessidade.Div(loteMultiplo).Ceil().Mul(loteMultiplo)
	}
	return necessidade
}

// registrarErro guarda a falha no log do cálculo — a primeira mensagem de cada
// tipo e quantas vezes ocorreu. O cálculo segue e termina COMPLETED_WITH_ERRORS:
// a falha aparece em vez de sumir.
func registrarErro(errs map[string]interface{}, chave string, err error) {
	if err == nil {
		return
	}
	if _, ja := errs[chave]; !ja {
		errs[chave] = err.Error()
	}
	n, _ := errs[chave+"_ocorrencias"].(int)
	errs[chave+"_ocorrencias"] = n + 1
}
