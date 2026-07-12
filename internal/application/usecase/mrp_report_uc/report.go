package mrp_report_uc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

type Filter struct {
	PlanCode               *int64
	ItemCode               *int64
	Planner                *int64
	Warehouse              *int64
	From                   *time.Time
	To                     *time.Time
	ItemType               string
	Position               string
	OnlyAvailable          bool
	ClassificationMaskCode *int64
	ClassificationCode     string
	Layout                 string
	OrderBy1               string
	OrderBy2               string
	BreakBy                string
	IncludeDrawings        bool
	IncludeSalesOrders     bool
	OnlyWithMessage        bool
	OnlyStockWithoutReason bool
	SalesOrderCodes        []int64
	Quantity               decimal.Decimal
	PlanningType           string
	OrderPosition          string
	Periods                []DateRange
	ExplosionOption        string
	ListMode               string
	DescriptionType        string
	ConsiderItemWarehouses bool
	ProductionOrderCodes   []int64
	LoadCodes              []int64
}

type DateRange struct{ From, To time.Time }

type ReportRow struct {
	ItemCode       int64             `json:"item_code"`
	Mask           string            `json:"mask,omitempty"`
	Date           *time.Time        `json:"date,omitempty"`
	Level          *int              `json:"level,omitempty"`
	ParentItemCode *int64            `json:"parent_item_code,omitempty"`
	OrderType      string            `json:"order_type,omitempty"`
	Demand         decimal.Decimal   `json:"demand"`
	PlannedSupply  decimal.Decimal   `json:"planned_supply"`
	FirmSupply     decimal.Decimal   `json:"firm_supply"`
	Stock          decimal.Decimal   `json:"stock"`
	ProjectedStock decimal.Decimal   `json:"projected_stock"`
	Available      decimal.Decimal   `json:"available"`
	Required       decimal.Decimal   `json:"required"`
	AverageMonthly decimal.Decimal   `json:"average_monthly"`
	SafetyStock    decimal.Decimal   `json:"safety_stock"`
	MaximumStock   decimal.Decimal   `json:"maximum_stock"`
	Planner        *int64            `json:"planner,omitempty"`
	Classification string            `json:"classification,omitempty"`
	DrawingCodes   []string          `json:"drawing_codes,omitempty"`
	SalesOrders    []int64           `json:"sales_orders,omitempty"`
	BreakKey       string            `json:"break_key,omitempty"`
	RowType        string            `json:"row_type,omitempty"`
	SourceType     string            `json:"source_type,omitempty"`
	SourceCode     *int64            `json:"source_code,omitempty"`
	Description    string            `json:"description,omitempty"`
	UOM            string            `json:"uom,omitempty"`
	PurchaseSupply decimal.Decimal   `json:"purchase_supply"`
	Cost           decimal.Decimal   `json:"cost"`
	PeriodValues   []decimal.Decimal `json:"period_values,omitempty"`
}

type Reader interface {
	Profile(context.Context, Filter) ([]ReportRow, error)
	Availability(context.Context, Filter) ([]ReportRow, error)
	GroupedNeeds(context.Context, Filter) ([]ReportRow, error)
	Explosion(context.Context, int64, decimal.Decimal, *time.Time, Filter) ([]ReportRow, error)
	ReorderPoint(context.Context, Filter) ([]ReportRow, error)
}

type UseCase struct {
	Reader Reader
	Auth   ports.AuthService
}

func (uc *UseCase) authorize(ctx context.Context) error {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

func validateFilter(filter *Filter, report string) error {
	filter.ItemType = strings.ToUpper(strings.TrimSpace(filter.ItemType))
	filter.Position = strings.ToUpper(strings.TrimSpace(filter.Position))
	filter.Layout = strings.ToUpper(strings.TrimSpace(filter.Layout))
	filter.OrderBy1 = strings.ToUpper(strings.TrimSpace(filter.OrderBy1))
	filter.OrderBy2 = strings.ToUpper(strings.TrimSpace(filter.OrderBy2))
	filter.BreakBy = strings.ToUpper(strings.TrimSpace(filter.BreakBy))
	filter.PlanningType = strings.ToUpper(strings.TrimSpace(filter.PlanningType))
	filter.OrderPosition = strings.ToUpper(strings.TrimSpace(filter.OrderPosition))
	filter.ExplosionOption = strings.ToUpper(strings.TrimSpace(filter.ExplosionOption))
	filter.ListMode = strings.ToUpper(strings.TrimSpace(filter.ListMode))
	filter.DescriptionType = strings.ToUpper(strings.TrimSpace(filter.DescriptionType))
	if filter.From != nil && filter.To != nil && filter.To.Before(*filter.From) {
		return errorsuc.NewValidationError("from cannot be after to")
	}
	if (filter.ClassificationMaskCode == nil) != (strings.TrimSpace(filter.ClassificationCode) == "") {
		return errorsuc.NewValidationError("classification_mask_code and classification_code must be informed together")
	}
	allowedItemType := map[string]bool{"": true, "TODOS": true, "FABRICADO": true, "COMPRADO": true, "DE_TERCEIRO": true, "TERCEIRIZADO": true}
	if !allowedItemType[filter.ItemType] {
		return errorsuc.NewValidationError("invalid item_type")
	}
	allowedBreak := map[string]bool{"": true, "NENHUM": true, "PLANEJADOR": true, "CLASSIFICACAO": true, "ITEM": true}
	if !allowedBreak[filter.BreakBy] {
		return errorsuc.NewValidationError("invalid break_by")
	}
	allowedOrder := map[string]bool{"": true, "NENHUM": true, "PLANEJADOR": true, "CLASSIFICACAO": true, "ITEM": true, "CODIGO": true, "DESCRICAO": true, "DATA": true}
	if !allowedOrder[filter.OrderBy1] || !allowedOrder[filter.OrderBy2] {
		return errorsuc.NewValidationError("invalid report ordering")
	}
	if !map[string]bool{"": true, "CALCULATION": true, "CURRENT": true}[filter.Position] {
		return errorsuc.NewValidationError("invalid position")
	}
	if (report == "profile" || report == "grouped") && filter.PlanCode == nil {
		return errorsuc.NewValidationError("plan_code is required")
	}
	for _, p := range filter.Periods {
		if p.To.Before(p.From) {
			return errorsuc.NewValidationError("period start cannot be after period end")
		}
	}
	layouts := map[string]map[string]bool{
		"profile":      {"": true, "ANALITICO": true, "SINTETICO": true},
		"availability": {"": true, "AMBOS": true, "NECESSIDADES": true, "ITENS_PEDIDO": true},
	}
	if set, ok := layouts[report]; ok && !set[filter.Layout] {
		return errorsuc.NewValidationError(fmt.Sprintf("invalid layout for %s report", report))
	}
	if report == "availability" && len(filter.SalesOrderCodes) == 0 && (filter.ItemCode == nil || !filter.Quantity.IsPositive()) {
		return errorsuc.NewValidationError("sales_orders or item_code with positive quantity are required")
	}
	if report == "grouped" && len(filter.Periods) > 0 && len(filter.Periods) != 6 {
		return errorsuc.NewValidationError("grouped needs requires exactly six periods when periods are informed")
	}
	if report == "reorder" && !map[string]bool{"": true, "TODOS": true, "REORDER_POINT": true, "KANBAN": true}[filter.PlanningType] {
		return errorsuc.NewValidationError("invalid planning_type")
	}
	if report == "reorder" && !map[string]bool{"": true, "LIBERADOS": true, "LIBERADOS_E_BLOQUEADOS": true}[filter.OrderPosition] {
		return errorsuc.NewValidationError("invalid order_position")
	}
	if report == "explosion" {
		if !map[string]bool{"": true, "SIMPLES": true, "CUSTO": true, "SALDO": true, "SALDO_DEM": true}[filter.ExplosionOption] {
			return errorsuc.NewValidationError("invalid explosion_option")
		}
		if !map[string]bool{"": true, "TODOS": true, "FILHOS_IMEDIATOS": true}[filter.ListMode] {
			return errorsuc.NewValidationError("invalid list_mode")
		}
	}
	return nil
}

func finalizeRows(rows []ReportRow, filter Filter) []ReportRow {
	value := func(row ReportRow, key string) string {
		switch key {
		case "PLANEJADOR":
			if row.Planner != nil {
				return fmt.Sprintf("%020d", *row.Planner)
			}
		case "CLASSIFICACAO":
			return row.Classification
		case "DESCRICAO":
			return row.Mask
		case "DATA":
			if row.Date != nil {
				return row.Date.Format("2006-01-02")
			}
		case "ITEM", "CODIGO":
			return fmt.Sprintf("%020d", row.ItemCode)
		}
		return ""
	}
	if filter.OrderBy1 != "" || filter.OrderBy2 != "" {
		sort.SliceStable(rows, func(i, j int) bool {
			for _, key := range []string{filter.OrderBy1, filter.OrderBy2, "ITEM", "DATA"} {
				a, b := value(rows[i], key), value(rows[j], key)
				if a != b {
					return a < b
				}
			}
			return false
		})
	}
	for i := range rows {
		switch filter.BreakBy {
		case "PLANEJADOR":
			if rows[i].Planner != nil {
				rows[i].BreakKey = strconv.FormatInt(*rows[i].Planner, 10)
			}
		case "CLASSIFICACAO":
			rows[i].BreakKey = rows[i].Classification
		case "ITEM":
			rows[i].BreakKey = strconv.FormatInt(rows[i].ItemCode, 10)
		}
	}
	return rows
}

func aggregatePeriods(rows []ReportRow, periods []DateRange) []ReportRow {
	if len(periods) == 0 {
		return rows
	}
	byItem := make(map[int64]*ReportRow)
	order := make([]int64, 0)
	for _, row := range rows {
		target := byItem[row.ItemCode]
		if target == nil {
			copyRow := row
			copyRow.Date = nil
			copyRow.Demand, copyRow.PlannedSupply, copyRow.FirmSupply, copyRow.Required = decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
			copyRow.PeriodValues = make([]decimal.Decimal, len(periods))
			target = &copyRow
			byItem[row.ItemCode] = target
			order = append(order, row.ItemCode)
		}
		target.Demand = target.Demand.Add(row.Demand)
		target.PlannedSupply = target.PlannedSupply.Add(row.PlannedSupply)
		target.FirmSupply = target.FirmSupply.Add(row.FirmSupply)
		target.Required = target.Required.Add(row.Required)
		if row.Date != nil {
			for i, period := range periods {
				if !row.Date.Before(period.From) && !row.Date.After(period.To) {
					target.PeriodValues[i] = target.PeriodValues[i].Add(row.Required)
				}
			}
		}
	}
	result := make([]ReportRow, 0, len(order))
	for _, item := range order {
		result = append(result, *byItem[item])
	}
	return result
}

func (uc *UseCase) Profile(ctx context.Context, filter Filter) ([]ReportRow, error) {
	if err := uc.authorize(ctx); err != nil {
		return nil, err
	}
	if err := validateFilter(&filter, "profile"); err != nil {
		return nil, err
	}
	rows, err := uc.Reader.Profile(ctx, filter)
	return finalizeRows(rows, filter), err
}
func (uc *UseCase) Availability(ctx context.Context, filter Filter) ([]ReportRow, error) {
	if err := uc.authorize(ctx); err != nil {
		return nil, err
	}
	if err := validateFilter(&filter, "availability"); err != nil {
		return nil, err
	}
	rows, err := uc.Reader.Availability(ctx, filter)
	return finalizeRows(rows, filter), err
}
func (uc *UseCase) GroupedNeeds(ctx context.Context, filter Filter) ([]ReportRow, error) {
	if err := uc.authorize(ctx); err != nil {
		return nil, err
	}
	if err := validateFilter(&filter, "grouped"); err != nil {
		return nil, err
	}
	rows, err := uc.Reader.GroupedNeeds(ctx, filter)
	if err != nil {
		return nil, err
	}
	return finalizeRows(aggregatePeriods(rows, filter.Periods), filter), nil
}
func (uc *UseCase) Explosion(ctx context.Context, itemCode int64, quantity decimal.Decimal, at *time.Time, filter Filter) ([]ReportRow, error) {
	if err := uc.authorize(ctx); err != nil {
		return nil, err
	}
	if len(filter.ProductionOrderCodes) == 0 && len(filter.LoadCodes) == 0 && (itemCode == 0 || !quantity.IsPositive()) {
		return nil, errorsuc.NewValidationError("item_code and positive quantity, production_orders or loads are required")
	}
	if err := validateFilter(&filter, "explosion"); err != nil {
		return nil, err
	}
	rows, err := uc.Reader.Explosion(ctx, itemCode, quantity, at, filter)
	return finalizeRows(rows, filter), err
}
func (uc *UseCase) ReorderPoint(ctx context.Context, filter Filter) ([]ReportRow, error) {
	if err := uc.authorize(ctx); err != nil {
		return nil, err
	}
	if err := validateFilter(&filter, "reorder"); err != nil {
		return nil, err
	}
	rows, err := uc.Reader.ReorderPoint(ctx, filter)
	return finalizeRows(rows, filter), err
}
