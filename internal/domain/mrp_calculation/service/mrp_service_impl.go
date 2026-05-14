package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	inddemandrepo "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	calrepo "github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	planentity "github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	planrepo "github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
	restrictionrepo "github.com/FelipePn10/panossoerp/internal/domain/restriction/repository"
	forecastrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	structrepo "github.com/FelipePn10/panossoerp/internal/domain/structure/repository"
)

// tooEarlyDays is the threshold (in calendar days) beyond which a firm supply
// arriving before the need date triggers a RESCHEDULE_OUT exception.
const tooEarlyDays = 30

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
	}
}

// cachedItemMRP holds the item fields the MRP needs, looked up once per item.
type cachedItemMRP struct {
	engineeringType int
	typeMRP         int
}

// Calculate orchestrates the full MRP run. Strategy:
//  1. One recursive CTE loads the entire BOM tree (no N+1).
//  2. Two bulk queries load all snapshots and configured rules.
//  3. If SupplyPort is wired: one query loads all firm supply for time-phased netting.
//  4. The main loop processes items level by level; each item costs zero extra queries
//     (item-type lookups are cached; workday subtraction is a single PG function call).
//  5. After the main loop: exception messages are generated comparing firm supply
//     against the computed net requirements.
func (s *MRPServiceImpl) Calculate(ctx context.Context, planCode int64, generateLLC bool) (*entity.MRPCalculationLog, error) {
	log, err := s.MRPRepo.StartCalculation(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("starting calculation log: %w", err)
	}

	errs := make(map[string]interface{})

	_ = s.MRPRepo.DeleteSuggestionsByPlan(ctx, planCode)
	_ = s.MRPRepo.DeleteProfilesByPlan(ctx, planCode)
	_ = s.MRPRepo.DeleteExceptionsByPlan(ctx, planCode)

	// Load the production plan to drive demand filtering and item scope.
	plan, err := s.PlanRepo.GetByCode(ctx, planCode)
	if err != nil {
		errs["load_plan"] = err.Error()
		return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
	}

	// Load independent demands unless the plan disables them.
	var demands []*inddemandentity.IndependentDemand
	if plan.IndependentDemands != planentity.IndependentDemandsNo {
		demands, err = s.DemandRepo.List(ctx)
		if err != nil {
			errs["load_demands"] = err.Error()
			return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
		}
		if plan.IndependentDemands == planentity.IndependentDemandsFromDate {
			demands = s.filterDemandsFromDate(demands, plan.Parameters)
		}
	}

	// Load sales forecasts and merge as MRP inputs.
	forecastInputs := s.loadForecastDemands(ctx, planCode)

	if len(demands) == 0 && len(forecastInputs) == 0 {
		return s.MRPRepo.FinishCalculation(ctx, log.Code, "COMPLETED", nil, 0, 0)
	}

	// Build allowed item set from plan classification / class item codes.
	allowedItems := s.buildAllowedItemSet(plan)

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
	for _, fi := range forecastInputs {
		if !s.itemAllowed(allowedItems, fi.ItemCode) {
			continue
		}
		if !seen[fi.ItemCode] {
			seen[fi.ItemCode] = true
			rootItems = append(rootItems, fi.ItemCode)
		}
	}

	// Bulk load 1: entire BOM tree in one recursive CTE — used for LLC and explosion.
	bomMap, err := s.StructRepo.LoadBOMForRoots(ctx, rootItems)
	if err != nil {
		errs["load_bom"] = err.Error()
		return s.MRPRepo.FinishCalculation(ctx, log.Code, "ERROR", errs, 0, 0)
	}

	llcMap := buildLLCFromBOM(bomMap, rootItems)

	// Bulk load 2 & 3: snapshots and configured rules — one query each.
	snapshotMap, err := s.MRPRepo.ListAllStockSnapshots(ctx)
	if err != nil {
		snapshotMap = make(map[int64]*entity.StockSnapshot)
	}
	rulesMap, err := s.MRPRepo.ListAllConfiguredRules(ctx)
	if err != nil {
		rulesMap = make(map[int64][]*entity.ConfiguredItemRule)
	}

	// Bulk load 4 (optional): firm supply from planned_order for time-phased netting.
	allCodes := collectAllItemCodes(bomMap, rootItems)
	var supplyMap map[int64][]ports.SupplyEntry
	if s.SupplyPort != nil {
		supplyMap, _ = s.SupplyPort.ListFirmSupplyForItems(ctx, allCodes)
	}
	if supplyMap == nil {
		supplyMap = make(map[int64][]ports.SupplyEntry)
	}

	// Bulk load 5: restricted item codes — skip planning suggestions for these.
	restrictedItems := make(map[int64]struct{})
	if s.RestrictionRepo != nil {
		restrictedItems, _ = s.RestrictionRepo.ListRestrictedItemCodes(ctx, allCodes)
	}

	// Lazy item-type cache: one ItemRepo call per unique item across the whole run.
	itemCache := make(map[int64]*cachedItemMRP)

	// Accumulators for exception-message generation after the main loop.
	netReqByItem := make(map[int64]float64)
	needDateByItem := make(map[int64]time.Time)

	// Seed level 0 from independent demands.
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
		levelQueues[llc] = append(levelQueues[llc], &entity.MRPInput{
			PlanCode: planCode,
			ItemCode: d.ItemCode,
			Mask:     mask,
			Quantity: d.Quantity,
			NeedDate: d.DemandDate,
			LLC:      llc,
		})
	}
	// Seed from sales forecast inputs.
	for _, fi := range forecastInputs {
		if !s.itemAllowed(allowedItems, fi.ItemCode) {
			continue
		}
		llc := llcMap[fi.ItemCode]
		fi.LLC = llc
		levelQueues[llc] = append(levelQueues[llc], fi)
	}

	maxLevel := maxLLC(llcMap)
	totalItems := 0
	totalOrders := 0

	for level := 0; level <= maxLevel; level++ {
		inputs, ok := levelQueues[level]
		if !ok {
			continue
		}

		for _, input := range aggregateInputs(inputs) {
			input.PlanCode = planCode
			input.LLC = level

			// Skip items blocked by an active restriction.
			if _, blocked := restrictedItems[input.ItemCode]; blocked {
				errs[fmt.Sprintf("item_%d_restricted", input.ItemCode)] = "item has active restriction — skipped"
				continue
			}

			output, err := s.calcNetReqFast(ctx, input, snapshotMap, rulesMap, supplyMap, itemCache)
			if err != nil {
				errs[fmt.Sprintf("item_%d", input.ItemCode)] = err.Error()
				continue
			}

			// Accumulate for exception generation.
			netReqByItem[input.ItemCode] += output.NetRequirement
			if existing, ok := needDateByItem[input.ItemCode]; !ok || input.NeedDate.Before(existing) {
				needDateByItem[input.ItemCode] = input.NeedDate
			}

			_, _ = s.MRPRepo.CreateProfile(ctx, &entity.MRPItemProfile{
				ItemCode:        input.ItemCode,
				PlanCode:        planCode,
				CalculationDate: time.Now(),
				Demand:          output.Demand,
				OrdersPlanned:   output.NetRequirement,
				OrdersFirm:      firmSupplyForItem(supplyMap, input.ItemCode, input.NeedDate),
				StockProjected:  output.StockProjected,
				LLC:             level,
				NeedDate:        input.NeedDate,
			})
			totalItems++

			for _, suggestion := range output.PlannedOrders {
				suggestion.PlanCode = planCode
				_, _ = s.MRPRepo.CreatePlannedOrderSuggestion(ctx, suggestion)
				totalOrders++

				if suggestion.StartDate == nil {
					continue
				}
				children := explodeFromBOM(bomMap, input.ItemCode, input.Mask, suggestion.Quantity, level+1)
				for _, child := range children {
					child.PlanCode = planCode
					child.NeedDate = *suggestion.StartDate
					child.ParentItemCode = &input.ItemCode
					levelQueues[level+1] = append(levelQueues[level+1], child)
				}
			}
		}
	}

	// Generate exception messages comparing firm supply against computed net requirements.
	if s.SupplyPort != nil {
		s.generateExceptionMessages(ctx, planCode, supplyMap, netReqByItem, needDateByItem)
	}

	status := "COMPLETED"
	if len(errs) > 0 {
		status = "COMPLETED_WITH_ERRORS"
	}

	return s.MRPRepo.FinishCalculation(ctx, log.Code, status, errs, totalItems, totalOrders)
}

// CalculateNetRequirements satisfies the MRPService interface for external callers.
// Does individual DB calls — for the optimised path inside Calculate, use calcNetReqFast.
func (s *MRPServiceImpl) CalculateNetRequirements(ctx context.Context, input *entity.MRPInput) (*entity.MRPOutput, error) {
	snapshotMap := make(map[int64]*entity.StockSnapshot)
	if snapshot, err := s.MRPRepo.GetStockSnapshot(ctx, input.ItemCode); err == nil && snapshot != nil {
		snapshotMap[input.ItemCode] = snapshot
	}

	rulesMap := make(map[int64][]*entity.ConfiguredItemRule)
	if rules, err := s.MRPRepo.GetConfiguredItemRules(ctx, input.ItemCode); err == nil {
		rulesMap[input.ItemCode] = rules
	}

	return s.calcNetReqFast(ctx, input, snapshotMap, rulesMap, nil, make(map[int64]*cachedItemMRP))
}

// ExplodeStructure satisfies the MRPService interface for external callers.
// Does a DB call — for the optimised path inside Calculate, use explodeFromBOM.
func (s *MRPServiceImpl) ExplodeStructure(ctx context.Context, parentCode int64, mask string, quantity float64, level int) ([]*entity.MRPInput, error) {
	if level > 20 {
		return nil, nil
	}

	children, err := s.StructRepo.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("exploding structure for item %d: %w", parentCode, err)
	}

	inputs := make([]*entity.MRPInput, 0, len(children))
	for _, child := range children {
		if !child.IsActive {
			continue
		}
		if child.ParentMask != nil && (mask == "" || *child.ParentMask != mask) {
			continue
		}
		adjustedQty := quantity * child.Quantity
		if child.LossPercentage > 0 {
			adjustedQty *= 1 + child.LossPercentage/100
		}
		inputs = append(inputs, &entity.MRPInput{
			ItemCode: child.ChildCode,
			Quantity: adjustedQty,
			LLC:      level,
		})
	}
	return inputs, nil
}

// CalculateItemLLC returns the LLC for a single item (external caller).
func (s *MRPServiceImpl) CalculateItemLLC(ctx context.Context, itemCode int64) (int, error) {
	llcMap, err := s.buildLLCMap(ctx, []int64{itemCode})
	if err != nil {
		return 0, err
	}
	return llcMap[itemCode], nil
}

// GenerateLLC is a no-op; LLC is computed in-memory per run inside Calculate.
func (s *MRPServiceImpl) GenerateLLC(ctx context.Context) error {
	return nil
}

// buildLLCMap loads the BOM and computes LLC via in-memory DFS.
// Used by CalculateItemLLC. Inside Calculate, buildLLCFromBOM is called directly.
func (s *MRPServiceImpl) buildLLCMap(ctx context.Context, rootItems []int64) (map[int64]int, error) {
	bomMap, err := s.StructRepo.LoadBOMForRoots(ctx, rootItems)
	if err != nil {
		return nil, err
	}
	return buildLLCFromBOM(bomMap, rootItems), nil
}

// subtractWorkdays delegates to the DB function subtract_workdays (migration 000080).
// One round-trip regardless of how many days are subtracted.
func (s *MRPServiceImpl) subtractWorkdays(ctx context.Context, from time.Time, days int) (time.Time, error) {
	if days <= 0 {
		return from, nil
	}
	return s.CalRepo.SubtractWorkdays(ctx, from, days)
}

// =============================================================================
// Private helpers — optimised Calculate path
// =============================================================================

// calcNetReqFast computes net requirements using pre-loaded in-memory maps.
// Time-phased netting: firm supply entries with ArrivalDate <= NeedDate are
// subtracted from the gross requirement before generating a new planned order.
func (s *MRPServiceImpl) calcNetReqFast(
	ctx context.Context,
	input *entity.MRPInput,
	snapshotMap map[int64]*entity.StockSnapshot,
	rulesMap map[int64][]*entity.ConfiguredItemRule,
	supplyMap map[int64][]ports.SupplyEntry, // nil = no netting
	itemCache map[int64]*cachedItemMRP,
) (*entity.MRPOutput, error) {

	// 1. Available stock from snapshot.
	availableStock := 0.0
	if snapshot, ok := snapshotMap[input.ItemCode]; ok && snapshot != nil {
		avail := snapshot.Quantity - snapshot.ReservedQty - snapshot.SafetyStock
		if avail > 0 {
			availableStock = avail
		}
	}

	// 2. Time-phased netting: firm supply arriving by the need date reduces gross demand.
	firmSupply := 0.0
	if supplyMap != nil {
		for _, entry := range supplyMap[input.ItemCode] {
			if !entry.ArrivalDate.After(input.NeedDate) {
				firmSupply += entry.Quantity
			}
		}
	}

	// 3. Configured rules (lead time, minimum lot).
	leadTimeDays := 0
	minLot := 0.0
	for _, rule := range rulesMap[input.ItemCode] {
		if rule.RuleType != "EQUAL" {
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
	demandType := "INDEPENDENTE"
	if input.ParentItemCode != nil {
		demandType = "DEPENDENTE"
	}

	// 5. Item-type lookup — cached across the whole run.
	cached, alreadyFetched := itemCache[input.ItemCode]
	if !alreadyFetched {
		if code, err := valueobject.NewItemCode(input.ItemCode); err == nil {
			if item, err := s.ItemRepo.FindItemByCode(ctx, code); err == nil {
				cached = &cachedItemMRP{
					engineeringType: int(item.Engineering.Type),
					typeMRP:         int(item.Planning.TypeMRP),
				}
			}
		}
		if cached == nil {
			cached = &cachedItemMRP{}
		}
		itemCache[input.ItemCode] = cached
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

	// 6. Start date = need date minus lead time (single PG function call).
	startDate, err := s.subtractWorkdays(ctx, input.NeedDate, leadTimeDays)
	if err != nil {
		startDate = input.NeedDate.AddDate(0, 0, -leadTimeDays)
	}

	output.NetRequirement = netReq
	output.PlannedOrders = []*entity.PlannedOrderSuggestion{
		{
			PlanCode:       input.PlanCode,
			ItemCode:       input.ItemCode,
			Quantity:       netReq,
			NeedDate:       input.NeedDate,
			StartDate:      &startDate,
			OrderType:      orderType,
			DemandType:     demandType,
			ParentItemCode: input.ParentItemCode,
			LLC:            input.LLC,
		},
	}

	return output, nil
}

// generateExceptionMessages analyses existing firm supply versus what the MRP
// actually computed and persists actionable exception messages.
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

		// Per-order: RESCHEDULE_IN (late) or RESCHEDULE_OUT (too early).
		for _, e := range entries {
			eCode := e.SourceCode
			eType := string(e.SourceType)

			if e.ArrivalDate.After(needDate) {
				// Order arrives after the demand needs it — expedite.
				days := int(e.ArrivalDate.Sub(needDate).Hours() / 24)
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
				continue
			}

			// Order arrives too early — ties up capital and storage.
			earlyDays := int(needDate.Sub(e.ArrivalDate).Hours() / 24)
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
		for _, child := range bomMap[itemCode] {
			assignLLC(child.ChildCode, level+1)
		}
	}

	for _, itemCode := range rootItems {
		assignLLC(itemCode, 0)
	}
	return llcMap
}

// explodeFromBOM expands one BOM level using the pre-loaded adjacency map.
func explodeFromBOM(
	bomMap map[int64][]*structentity.ItemStructure,
	parentCode int64,
	mask string,
	quantity float64,
	level int,
) []*entity.MRPInput {
	if level > 20 {
		return nil
	}
	children := bomMap[parentCode]
	inputs := make([]*entity.MRPInput, 0, len(children))
	for _, child := range children {
		if child.ParentMask != nil && (mask == "" || *child.ParentMask != mask) {
			continue
		}
		adjustedQty := quantity * child.Quantity
		if child.LossPercentage > 0 {
			adjustedQty *= 1 + child.LossPercentage/100
		}
		inputs = append(inputs, &entity.MRPInput{
			ItemCode: child.ChildCode,
			Quantity: adjustedQty,
			LLC:      level,
		})
	}
	return inputs
}

// collectAllItemCodes returns every item code reachable from roots via the BOM.
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

// firmSupplyForItem sums firm supply entries arriving by needDate for an item.
// Used to populate OrdersFirm in the MRP profile.
func firmSupplyForItem(supplyMap map[int64][]ports.SupplyEntry, itemCode int64, needDate time.Time) float64 {
	total := 0.0
	for _, entry := range supplyMap[itemCode] {
		if !entry.ArrivalDate.After(needDate) {
			total += entry.Quantity
		}
	}
	return total
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

// aggregateInputs merges MRPInput entries for the same item+mask, summing
// quantities and keeping the earliest need date.
func aggregateInputs(inputs []*entity.MRPInput) []*entity.MRPInput {
	type key struct {
		itemCode int64
		mask     string
	}
	agg := make(map[key]*entity.MRPInput, len(inputs))
	for _, inp := range inputs {
		k := key{inp.ItemCode, inp.Mask}
		if existing, ok := agg[k]; ok {
			existing.Quantity += inp.Quantity
			if inp.NeedDate.Before(existing.NeedDate) {
				existing.NeedDate = inp.NeedDate
			}
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

// filterDemandsFromDate removes independent demands whose DemandDate is before
// the "from_date" stored in plan Parameters. Falls back to no filtering if the
// key is absent or unparseable.
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

// buildAllowedItemSet builds a set of permitted item codes based on
// plan.ClassItemCodes (comma-separated). If nil/empty, all items are allowed
// (nil map returned = no filter).
func (s *MRPServiceImpl) buildAllowedItemSet(plan *planentity.ProductionPlan) map[int64]struct{} {
	if plan.ClassItemCodes == nil || *plan.ClassItemCodes == "" {
		return nil
	}
	parts := strings.Split(*plan.ClassItemCodes, ",")
	set := make(map[int64]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if code, err := strconv.ParseInt(p, 10, 64); err == nil {
			set[code] = struct{}{}
		}
	}
	return set
}

// itemAllowed returns true when the item is within the plan scope.
// A nil allowed map means all items are permitted.
func (s *MRPServiceImpl) itemAllowed(allowed map[int64]struct{}, itemCode int64) bool {
	if allowed == nil {
		return true
	}
	_, ok := allowed[itemCode]
	return ok
}

// ---------- Sales Forecast → MRP demand ----------

// loadForecastDemands converts sales forecasts for the current year into
// MRPInput entries, summing quantities per item and using Monday of each
// ISO week as the need date.
func (s *MRPServiceImpl) loadForecastDemands(ctx context.Context, planCode int64) []*entity.MRPInput {
	if s.ForecastRepo == nil {
		return nil
	}
	year := time.Now().Year()
	forecasts, err := s.ForecastRepo.ListForecasts(ctx, year)
	if err != nil {
		return nil
	}
	// Also include next year's forecasts for long-horizon plans.
	nextYear, _ := s.ForecastRepo.ListForecasts(ctx, year+1)
	forecasts = append(forecasts, nextYear...)

	type key struct {
		itemCode int64
		mask     string
	}
	agg := make(map[key]*entity.MRPInput)
	for _, f := range forecasts {
		needDate := mrpWeekToDate(f.Year, f.Week)
		mask := ""
		if f.Mask != nil {
			mask = *f.Mask
		}
		k := key{f.ItemCode, mask}
		if existing, ok := agg[k]; ok {
			existing.Quantity += f.Quantity
			if needDate.Before(existing.NeedDate) {
				existing.NeedDate = needDate
			}
		} else {
			agg[k] = &entity.MRPInput{
				PlanCode: planCode,
				ItemCode: f.ItemCode,
				Mask:     mask,
				Quantity: f.Quantity,
				NeedDate: needDate,
			}
		}
	}
	result := make([]*entity.MRPInput, 0, len(agg))
	for _, v := range agg {
		result = append(result, v)
	}
	return result
}

// mrpWeekToDate returns the Monday of the given ISO week/year.
func mrpWeekToDate(year, week int) time.Time {
	// Jan 4 is always in ISO week 1.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	// Weekday of Jan 4: Monday=0 offset.
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, -weekday+1)
	return monday.AddDate(0, 0, (week-1)*7)
}
