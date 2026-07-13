package service

import (
	"context"
	"fmt"
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
	RoutingRepo     routingrepo.RoutingRepository // nil = fallback to configured lead time
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
) MRPService {
	return &MRPServiceImpl{
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
}

// cachedItemMRP holds the item fields the MRP needs, looked up once per item.
type cachedItemMRP struct {
	engineeringType int
	typeMRP         int
	ghost           bool
	reorderPoint    *valueobject.ReorderPoint
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

	_ = s.MRPRepo.DeleteSuggestionsByPlan(ctx, planCode)
	_ = s.MRPRepo.DeleteProfilesByPlan(ctx, planCode)
	_ = s.MRPRepo.DeleteProfileDetailsByPlan(ctx, planCode)
	_ = s.MRPRepo.DeleteExceptionsByPlan(ctx, planCode)

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
		demandInputs = append(demandInputs, s.loadForecastDemands(ctx, planCode)...)
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
		supplyMap, _ = s.SupplyPort.ListFirmSupplyForItems(ctx, allCodeSlice)
	}
	if supplyMap == nil {
		supplyMap = make(map[int64][]ports.SupplyEntry)
	}

	// Bulk load 5: restricted items.
	restrictedItems := make(map[int64]struct{})
	if s.RestrictionRepo != nil {
		restrictedItems, _ = s.RestrictionRepo.ListRestrictedItemCodes(ctx, allCodeSlice)
	}

	// Item cache — lazy load from DB.
	itemCache := make(map[int64]*cachedItemMRP)

	// Accumuladores pós-loop.
	netReqByItem := make(map[int64]float64)
	needDateByItem := make(map[int64]time.Time)
	totalItems := 0
	totalOrders := 0
	nextOrderNumber := initialOrderNumber

	// Dispara modos conforme PlanningTypes do plano.
	if len(plan.PlanningTypes) == 0 {
		plan.PlanningTypes = []string{"MRP"}
	}

	for _, pt := range plan.PlanningTypes {
		switch strings.ToUpper(strings.TrimSpace(pt)) {
		case "MIN_MAX":
			tItems, tOrders := s.calculateMinMax(ctx, planCode, params, snapshotMap, supplyMap, rulesMap, llcMap, itemCache, allowedItems, restrictedItems, &nextOrderNumber, errs)
			totalItems += tItems
			totalOrders += tOrders
		case "REORDER_POINT":
			tItems, tOrders := s.calculateReorderPoint(ctx, planCode, params, snapshotMap, supplyMap, rulesMap, llcMap, itemCache, allowedItems, restrictedItems, &nextOrderNumber, errs)
			totalItems += tItems
			totalOrders += tOrders
		case "KANBAN":
			tItems, tOrders := s.calculateKanban(ctx, planCode, params, snapshotMap, supplyMap, llcMap, itemCache, allowedItems, restrictedItems, &nextOrderNumber, errs)
			totalItems += tItems
			totalOrders += tOrders
		case "MPS":
			if !noMRPDemand {
				tItems, tOrders := s.calculateMPS(ctx, planCode, params, plan, snapshotMap, rulesMap, supplyMap, llcMap, bomMap, itemCache, allowedItems, restrictedItems, demands, demandInputs, &nextOrderNumber, &netReqByItem, &needDateByItem, errs)
				totalItems += tItems
				totalOrders += tOrders
			}
		default: // MRP
			if !noMRPDemand {
				tItems, tOrders := s.calculateMRP(ctx, planCode, params, plan, snapshotMap, rulesMap, supplyMap, llcMap, bomMap, itemCache, allowedItems, restrictedItems, demands, demandInputs, &nextOrderNumber, &netReqByItem, &needDateByItem, errs)
				totalItems += tItems
				totalOrders += tOrders
			}
		}
	}

	// Pós-processamento: integração com máquinas.
	s.processMachineIntegration(ctx, planCode, allCodeSlice)

	// Pós-processamento: geração de prioridades automáticas.
	if params.GerarPrioridadesOrdens {
		s.processAutoPriority(ctx, planCode, params)
	}

	// Geração de mensagens de exceção.
	if s.SupplyPort != nil {
		s.generateExceptionMessages(ctx, planCode, supplyMap, netReqByItem, needDateByItem)
	}
	if generateLLC {
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
	params, _ := s.MRPRepo.LoadTypedPlanningParams(ctx)
	if params == nil {
		params = entity.DefaultTypedPlanningParams()
	}
	snapshotMap := make(map[int64]*entity.StockSnapshot)
	if snapshot, err := s.MRPRepo.GetStockSnapshot(ctx, input.ItemCode); err == nil && snapshot != nil {
		snapshotMap[input.ItemCode] = snapshot
	}
	rulesMap := make(map[int64][]*entity.ConfiguredItemRule)
	if rules, err := s.MRPRepo.GetConfiguredItemRules(ctx, input.ItemCode); err == nil {
		rulesMap[input.ItemCode] = rules
	}
	return s.calcNetReqFast(ctx, input, snapshotMap, rulesMap, nil, make(map[int64]*cachedItemMRP), params)
}

func (s *MRPServiceImpl) ExplodeStructure(ctx context.Context, parentCode int64, mask string, quantity float64, level int) ([]*entity.MRPInput, error) {
	if level > 20 {
		return nil, nil
	}
	children, err := s.StructRepo.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("exploding structure for item %d: %w", parentCode, err)
	}
	params, _ := s.MRPRepo.LoadTypedPlanningParams(ctx)
	if params == nil {
		params = entity.DefaultTypedPlanningParams()
	}
	inputs := make([]*entity.MRPInput, 0, len(children))
	for _, child := range children {
		if !child.IsActive {
			continue
		}
		if child.ParentMask != nil && (mask == "" || *child.ParentMask != mask) {
			continue
		}
		adjustedQty := applyLossFormula(quantity, child.Quantity, child.LossPercentage, params.FormulaPerdasEstrutura)
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
	netReqByItem *map[int64]float64,
	needDateByItem *map[int64]time.Time,
	errs map[string]interface{},
) (int, int) {
	totalItems := 0
	totalOrders := 0

	// An order-item calculation must not introduce any additional independent source.
	var safetyInputs []*entity.MRPInput
	if plan.OrderItemCode == nil {
		safetyInputs = s.generateSafetyStockDemands(ctx, params, snapshotMap, supplyMap, llcMap, itemCache, allowedItems, restrictedItems)
	}
	for _, si := range safetyInputs {
		if params.AgrupaDemandaEstoque {
			// Agrega à primeira demanda existente do item.
			merged := false
			for _, fi := range forecastInputs {
				if fi.ItemCode == si.ItemCode && fi.Mask == si.Mask && !fi.TechnicalAssistance {
					fi.Quantity += si.Quantity
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

			(*netReqByItem)[input.ItemCode] += output.NetRequirement
			if existing, ok := (*needDateByItem)[input.ItemCode]; !ok || input.NeedDate.Before(existing) {
				(*needDateByItem)[input.ItemCode] = input.NeedDate
			}

			_, _ = s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        input.ItemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          output.Demand,
				OrdersPlanned:   output.NetRequirement,
				OrdersFirm:      firmSupplyUsed,
				StockProjected:  output.StockProjected,
				LLC:             level,
				NeedDate:        input.NeedDate,
			})
			totalItems++

			for _, suggestion := range output.PlannedOrders {
				suggestion.PlanCode = planCode

				cached := s.ensureItemCache(ctx, itemCache, input.ItemCode)

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
				}

				if suggestion.StartDate == nil {
					continue
				}

				if cached.ghost && !params.ItensFantasmasGravar {
					children := explodeFromBOMWithFormula(bomMap, input.ItemCode, input.Mask, suggestion.Quantity, level+1, params.FormulaPerdasEstrutura)
					for _, child := range children {
						child.PlanCode = planCode
						child.NeedDate = *suggestion.StartDate
						child.ParentItemCode = &input.ItemCode
						levelQueues[level+1] = append(levelQueues[level+1], child)
					}
					continue
				}

				children := explodeFromBOMWithFormula(bomMap, input.ItemCode, input.Mask, suggestion.Quantity, level+1, params.FormulaPerdasEstrutura)
				for _, child := range children {
					child.PlanCode = planCode
					child.NeedDate = *suggestion.StartDate
					child.ParentItemCode = &input.ItemCode
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

	planningExtras, _ := s.MRPRepo.ListItemPlanningExtras(ctx)
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
		if lmi <= 0 {
			continue
		}

		// LMA = maximum_stock (ou minimum_stock * 3 como default)
		lma := lmi * 3
		if extra, ok := planningExtras[itemCode]; ok && extra.MaximumStock > 0 {
			lma = extra.MaximumStock
		}

		// QTDE = stock disponível (quantity - reserved)
		qtde := snapshot.Quantity - snapshot.ReservedQty
		if qtde < 0 {
			qtde = 0
		}

		// QTDP = firm supply (ordens firmes de compra)
		qtdp := firmSupplyForItem(supplyMap, itemCode, maxTime())

		qtdeTotal := qtde + qtdp

		if qtdeTotal <= lmi {
			qtc := lma - qtdeTotal
			if qtc <= 0 {
				continue
			}

			cached := s.ensureItemCache(ctx, itemCache, itemCode)
			if cached == nil || cached.engineeringType == 2 { // DE_TERCEIRO
				continue
			}

			leadTimeDays := s.getLeadTimeDays(rulesMap, itemCode)
			needDate := today().AddDate(0, 0, leadTimeDays)
			startDate, _ := s.subtractWorkdays(ctx, needDate, leadTimeDays)
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

			_, _ = s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        itemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          qtc,
				OrdersPlanned:   qtc,
				OrdersFirm:      qtdp,
				StockProjected:  qtdeTotal - qtc,
				LLC:             llc,
				NeedDate:        needDate,
			})
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

		cached := s.ensureItemCache(ctx, itemCache, itemCode)
		if cached == nil || cached.engineeringType == 2 {
			continue
		}
		if cached.reorderPoint == nil {
			continue
		}

		pr, err := cached.reorderPoint.Calculate()
		if err != nil || pr <= 0 {
			continue
		}

		qtde := snapshot.Quantity - snapshot.ReservedQty
		if qtde < 0 {
			qtde = 0
		}

		if qtde <= float64(pr) {
			loteEconomico := float64(pr * 2)
			leadTimeDays := s.getLeadTimeDays(rulesMap, itemCode)
			needDate := today().AddDate(0, 0, leadTimeDays)
			startDate, _ := s.subtractWorkdays(ctx, needDate, leadTimeDays)
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

			_, _ = s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        itemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          loteEconomico,
				OrdersPlanned:   loteEconomico,
				OrdersFirm:      0,
				StockProjected:  qtde - loteEconomico,
				LLC:             llc,
				NeedDate:        needDate,
			})
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

		qtde := 0.0
		if snapshot, ok := snapshotMap[itemCode]; ok && snapshot != nil {
			qtde = snapshot.Quantity - snapshot.ReservedQty
			if qtde < 0 {
				qtde = 0
			}
		}

		for _, card := range group.cards {
			if qtde > card.ReorderPoint {
				continue
			}

			cached := s.ensureItemCache(ctx, itemCache, itemCode)
			if cached == nil || cached.engineeringType == 2 {
				continue
			}

			qtc := card.QuantityPerCard * float64(card.CardCount)
			leadTimeDays := s.getLeadTimeDays(nil, itemCode)
			needDate := today().AddDate(0, 0, leadTimeDays)
			startDate, _ := s.subtractWorkdays(ctx, needDate, leadTimeDays)
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

			_, _ = s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        itemCode,
				PlanCode:        planCode,
				CalculationDate: today(),
				Demand:          qtc,
				OrdersPlanned:   qtc,
				OrdersFirm:      0,
				StockProjected:  qtde - qtc,
				LLC:             llc,
				NeedDate:        needDate,
			})
			totalItems++
			qtde -= qtc
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
	netReqByItem *map[int64]float64,
	needDateByItem *map[int64]time.Time,
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
		})
	}

	return s.calculateMRP(ctx, planCode, params, plan, snapshotMap, rulesMap, supplyMap,
		llcMap, bomMap, itemCache, allowedItems, restrictedItems,
		demands, forecastInputs, nextOrderNumber, netReqByItem, needDateByItem, errs)
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
	availableStock := 0.0
	if snapshot, ok := snapshotMap[input.ItemCode]; ok && snapshot != nil {
		avail := snapshot.Quantity - snapshot.ReservedQty - snapshot.SafetyStock
		if avail > 0 {
			availableStock = avail
		}
	}

	// 2. Time-phased netting.
	firmSupply := 0.0
	if supplyMap != nil {
		for _, entry := range supplyMap[input.ItemCode] {
			if !entry.ArrivalDate.After(input.NeedDate) {
				firmSupply += entry.Quantity
			}
		}
	}

	// 3. Configured rules (lead time, minimum lot) — suporta EQUAL, DIFFERENT, RANGE.
	leadTimeDays := 0
	minLot := 0.0
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
			if v, err := strconv.ParseFloat(rule.RuleValue, 64); err == nil {
				minLot = v
			}
		}
	}

	// 4. Net requirement.
	totalCoverage := availableStock + firmSupply
	netReq := input.Quantity - totalCoverage
	stockProjected := totalCoverage - input.Quantity

	output := &entity.MRPOutput{
		ItemCode:       input.ItemCode,
		LLC:            input.LLC,
		Demand:         input.Quantity,
		StockProjected: stockProjected,
		NetRequirement: netReq,
	}

	if netReq <= 0 {
		return output, nil
	}

	if minLot > 0 && netReq < minLot {
		netReq = minLot
	}

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

	// 5. Item-type lookup — cached.
	cached := s.ensureItemCache(ctx, itemCache, input.ItemCode)
	if cached == nil {
		cached = &cachedItemMRP{}
	}

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
		extOps, _ := s.RoutingRepo.GetExternalOpsByItem(ctx, input.ItemCode)
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

// generateExceptionMessages analyses existing firm supply versus what the MRP
// actually computed and persists actionable exception messages.
// RescheduleIn is split into EXPEDITE (<=5 days late) and RESCHEDULE_IN (>5 days).
func (s *MRPServiceImpl) generateExceptionMessages(
	ctx context.Context,
	planCode int64,
	supplyMap map[int64][]ports.SupplyEntry,
	netReqByItem map[int64]float64,
	needDateByItem map[int64]time.Time,
) {
	for itemCode, entries := range supplyMap {
		if len(entries) == 0 {
			continue
		}

		totalFirmSupply := 0.0
		for _, e := range entries {
			totalFirmSupply += e.Quantity
		}

		netReq := netReqByItem[itemCode]
		needDate, hasNeedDate := needDateByItem[itemCode]

		// CANCEL: firm order exists but the item has no demand in this plan.
		if !hasNeedDate || netReq <= 0 {
			for _, e := range entries {
				eCode := e.SourceCode
				eType := string(e.SourceType)
				desc := fmt.Sprintf(
					"Item %d possui ordem firme de %.2f unidades (código %d) sem demanda neste plano. Considere cancelar.",
					itemCode, e.Quantity, e.SourceCode,
				)
				_ = s.MRPRepo.CreateExceptionMessage(ctx, &entity.MRPExceptionMessage{
					PlanCode:    planCode,
					ItemCode:    itemCode,
					MessageType: entity.ExceptionCancel,
					SourceCode:  &eCode,
					SourceType:  &eType,
					Description: desc,
				})
			}
			continue
		}

		// EXCESS_PROJECTED: total firm supply significantly exceeds net requirement.
		if totalFirmSupply > netReq*(1+excessThreshold) {
			excess := totalFirmSupply - netReq
			desc := fmt.Sprintf(
				"Item %d: suprimento firme total (%.2f un.) excede a necessidade líquida (%.2f un.) em %.2f unidades. Estoque excedente projetado.",
				itemCode, totalFirmSupply, netReq, excess,
			)
			_ = s.MRPRepo.CreateExceptionMessage(ctx, &entity.MRPExceptionMessage{
				PlanCode:    planCode,
				ItemCode:    itemCode,
				MessageType: entity.ExceptionExcess,
				Description: desc,
			})
		}

		// Per-order: EXPEDITE (<= 5 days) / RESCHEDULE_IN (> 5 days) or RESCHEDULE_OUT.
		for _, e := range entries {
			eCode := e.SourceCode
			eType := string(e.SourceType)

			if e.ArrivalDate.After(needDate) {
				days := int(math.Ceil(e.ArrivalDate.Sub(needDate).Hours() / 24))
				if days <= expediteThreshold {
					desc := fmt.Sprintf(
						"Item %d: ordem %d (%.2f un.) chega em %s, %d dia(s) após necessidade %s. ACELERAR urgentemente.",
						itemCode, e.SourceCode, e.Quantity,
						e.ArrivalDate.Format("02/01/2006"), days, needDate.Format("02/01/2006"),
					)
					_ = s.MRPRepo.CreateExceptionMessage(ctx, &entity.MRPExceptionMessage{
						PlanCode:    planCode,
						ItemCode:    itemCode,
						MessageType: entity.ExceptionExpedite,
						SourceCode:  &eCode,
						SourceType:  &eType,
						Description: desc,
					})
				} else {
					desc := fmt.Sprintf(
						"Item %d: ordem %d (%.2f un.) chega em %s, mas a necessidade é %s (%d dia(s) de atraso). Antecipar.",
						itemCode, e.SourceCode, e.Quantity,
						e.ArrivalDate.Format("02/01/2006"), needDate.Format("02/01/2006"), days,
					)
					_ = s.MRPRepo.CreateExceptionMessage(ctx, &entity.MRPExceptionMessage{
						PlanCode:    planCode,
						ItemCode:    itemCode,
						MessageType: entity.ExceptionRescheduleIn,
						SourceCode:  &eCode,
						SourceType:  &eType,
						Description: desc,
					})
				}
				continue
			}

			earlyDays := int(math.Ceil(needDate.Sub(e.ArrivalDate).Hours() / 24))
			if earlyDays > tooEarlyDays {
				desc := fmt.Sprintf(
					"Item %d: ordem %d (%.2f un.) chega em %s, %d dia(s) antes da necessidade (%s). Atrasar para liberar capital.",
					itemCode, e.SourceCode, e.Quantity,
					e.ArrivalDate.Format("02/01/2006"), earlyDays, needDate.Format("02/01/2006"),
				)
				_ = s.MRPRepo.CreateExceptionMessage(ctx, &entity.MRPExceptionMessage{
					PlanCode:    planCode,
					ItemCode:    itemCode,
					MessageType: entity.ExceptionRescheduleOut,
					SourceCode:  &eCode,
					SourceType:  &eType,
					Description: desc,
				})
			}
		}
	}
}

// =============================================================================
// Pós-processamento: integração com máquinas e prioridades
// =============================================================================

// processMachineIntegration assigns machines and creates machine_schedule
// entries for FABRICACAO-type planned orders.
func (s *MRPServiceImpl) processMachineIntegration(ctx context.Context, planCode int64, allCodes []int64) {
	machineTimes, err := s.MRPRepo.ListItemMachineTimes(ctx, allCodes)
	if err != nil || len(machineTimes) == 0 {
		return
	}

	suggestions, err := s.MRPRepo.ListSuggestionsByPlan(ctx, planCode)
	if err != nil || len(suggestions) == 0 {
		return
	}

	for _, sug := range suggestions {
		if sug.OrderType != "FABRICACAO" {
			continue
		}
		mtList, ok := machineTimes[sug.ItemCode]
		if !ok || len(mtList) == 0 {
			continue
		}

		// Máquina com maior prioridade (menor número).
		bestMT := mtList[0]
		for _, mt := range mtList[1:] {
			if mt.Priority < bestMT.Priority {
				bestMT = mt
			}
		}

		productionTime := sug.Quantity * bestMT.ProductionTime

		_ = s.MRPRepo.UpdatePlannedOrderMachine(ctx, sug.Code, bestMT.MachineID, productionTime)

		scheduleDate := sug.NeedDate
		if sug.StartDate != nil {
			scheduleDate = *sug.StartDate
		}
		_ = s.MRPRepo.CreateMachineSchedule(ctx, &entity.MachineScheduleInfo{
			PlanCode:         planCode,
			PlannedOrderCode: sug.Code,
			MachineID:        bestMT.MachineID,
			ScheduleDate:     scheduleDate,
			ProductionTime:   productionTime,
		})
	}
}

// processAutoPriority assigns priority codes to planned orders based on
// order_priorities rules and the DiasPrioridades window.
func (s *MRPServiceImpl) processAutoPriority(ctx context.Context, planCode int64, params *entity.TypedPlanningParams) {
	priorities, err := s.MRPRepo.ListAllOrderPriorities(ctx)
	if err != nil || len(priorities) == 0 {
		return
	}

	suggestions, err := s.MRPRepo.ListSuggestionsByPlan(ctx, planCode)
	if err != nil || len(suggestions) == 0 {
		return
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
			_ = s.MRPRepo.UpdatePlannedOrderPriority(ctx, sug.Code, priority)
		}
	}
}

func findPriorityForQuantity(priorities []*orderpriority.OrderPriority, quantity float64) string {
	for _, p := range priorities {
		if quantity >= p.IntervalStart && quantity <= p.IntervalEnd {
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
func (s *MRPServiceImpl) getLeadTimeDays(rulesMap map[int64][]*entity.ConfiguredItemRule, itemCode int64) int {
	// 1. Try routing lead time (critical-path through operation network).
	if s.RoutingRepo != nil {
		route, err := s.RoutingRepo.GetRouteForItem(context.Background(), itemCode, "")
		if err == nil && route != nil {
			ops, err := s.RoutingRepo.GetRouteOperations(context.Background(), route.ID)
			if err == nil && len(ops) > 0 {
				edges, _ := s.RoutingRepo.GetNetworkEdges(context.Background(), route.ID)
				// qty=1: item-level planning lead time (per unit / per base batch).
				// Shared CPM implementation with the routing lead-time use case.
				totalHours := routingentity.CriticalPath(ops, edges, 1.0).TotalHours
				if totalHours > 0 {
					days := int(math.Ceil(totalHours / 8.0)) // 8h = 1 dia útil
					if days > 0 {
						return days
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
	return maxLead
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
) []*entity.MRPInput {
	var result []*entity.MRPInput
	baseDate := today()

	for itemCode, snapshot := range snapshotMap {
		if !s.itemAllowed(allowedItems, itemCode) {
			continue
		}
		if _, blocked := restrictedItems[itemCode]; blocked {
			continue
		}
		if snapshot.SafetyStock <= 0 {
			continue
		}

		if !params.GerarDemandaSegurancaTodos {
			// Only for items with movimentação: verifica consumo médio mensal.
			cached := s.ensureItemCache(ctx, itemCache, itemCode)
			if cached == nil {
				continue
			}
		}

		// Calcula data da necessidade.
		cached := s.ensureItemCache(ctx, itemCache, itemCode)
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
		})
	}

	return result
}

// =============================================================================
// Helpers — item cache e divisões
// =============================================================================

// ensureItemCache garante que o item esteja no cache, carregando do DB se necessário.
func (s *MRPServiceImpl) ensureItemCache(ctx context.Context, itemCache map[int64]*cachedItemMRP, itemCode int64) *cachedItemMRP {
	if cached, ok := itemCache[itemCode]; ok {
		return cached
	}

	var item *itementity.Item
	if code, err := valueobject.NewItemCode(itemCode); err == nil {
		item, _ = s.ItemRepo.FindItemByCode(ctx, code)
	}

	cached := &cachedItemMRP{}
	if item != nil {
		cached.engineeringType = int(item.Engineering.Type)
		cached.typeMRP = int(item.Planning.TypeMRP)
		cached.ghost = item.Planning.Ghost
		cached.reorderPoint = item.Planning.ReorderPoint
	}
	itemCache[itemCode] = cached
	return cached
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
	quantity float64,
	level int,
) []*entity.MRPInput {
	return explodeFromBOMWithFormula(bomMap, parentCode, mask, quantity, level, 1)
}

// explodeFromBOMWithFormula expands one BOM level with configurable loss formula.
func explodeFromBOMWithFormula(
	bomMap map[int64][]*structentity.ItemStructure,
	parentCode int64,
	mask string,
	quantity float64,
	level int,
	formula int,
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
			base = 1
		}
		adjustedQty := applyLossFormula(base, child.Quantity, child.LossPercentage, formula)
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
func applyLossFormula(parentQty, childQtyPer, lossPercentage float64, formula int) float64 {
	base := parentQty * childQtyPer
	if lossPercentage <= 0 {
		return base
	}
	switch formula {
	case 1:
		return base * (1 + lossPercentage/100)
	case 2:
		denominator := 1 - lossPercentage/100
		if denominator > 0 {
			return base / denominator
		}
		return base
	case 3:
		return base
	default:
		return base * (1 + lossPercentage/100)
	}
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

func firmSupplyForItem(supplyMap map[int64][]ports.SupplyEntry, itemCode int64, needDate time.Time) float64 {
	total := 0.0
	for _, entry := range supplyMap[itemCode] {
		if !entry.ArrivalDate.After(needDate) {
			total += entry.Quantity
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
	if nextOrderNumber == nil || *nextOrderNumber <= 0 {
		return nil, fmt.Errorf("invalid next order number")
	}
	number := *nextOrderNumber
	suggestion.OrderNumber = &number
	*nextOrderNumber++
	created, err := s.MRPRepo.CreatePlannedOrderSuggestion(ctx, suggestion)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// advanceNettingState prevents stock and firm receipts already consumed by an
// earlier demand from being reused by the next demand of the same item.
func advanceNettingState(input *entity.MRPInput, output *entity.MRPOutput, snapshotMap map[int64]*entity.StockSnapshot, supplyMap map[int64][]ports.SupplyEntry) {
	planned := 0.0
	for _, order := range output.PlannedOrders {
		planned += order.Quantity
	}
	snapshotMap[input.ItemCode] = &entity.StockSnapshot{
		ItemCode: input.ItemCode, Quantity: math.Max(0, output.StockProjected+planned), SnapshotDate: input.NeedDate,
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
			existing.Quantity += inp.Quantity
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

func (s *MRPServiceImpl) loadForecastDemands(ctx context.Context, planCode int64) []*entity.MRPInput {
	if s.ForecastRepo == nil {
		return nil
	}
	year := time.Now().Year()
	forecasts, err := s.ForecastRepo.ListForecasts(ctx, year)
	if err != nil {
		return nil
	}
	nextYear, _ := s.ForecastRepo.ListForecasts(ctx, year+1)
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
			Quantity:   f.Quantity,
			NeedDate:   needDate,
			DemandType: "FORECAST",
		})
	}
	return result
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
