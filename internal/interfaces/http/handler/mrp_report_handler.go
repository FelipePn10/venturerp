package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
)

type MRPReportHandler struct{ uc *mrp_report_uc.UseCase }

func NewMRPReportHandler(uc *mrp_report_uc.UseCase) *MRPReportHandler {
	return &MRPReportHandler{uc: uc}
}

func reportInt64(value string) *int64 {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}
func reportDate(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &parsed
}
func reportDateStrict(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("invalid date, expected YYYY-MM-DD")
	}
	return &parsed, nil
}
func reportInt64List(value string) ([]int64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer list")
		}
		result = append(result, parsed)
	}
	return result, nil
}
func reportPeriods(value string) ([]mrp_report_uc.DateRange, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) > 6 {
		return nil, fmt.Errorf("at most six periods are allowed")
	}
	result := make([]mrp_report_uc.DateRange, 0, len(parts))
	for _, part := range parts {
		bounds := strings.Split(strings.TrimSpace(part), "|")
		if len(bounds) != 2 {
			return nil, fmt.Errorf("period must use YYYY-MM-DD|YYYY-MM-DD")
		}
		from, err := time.Parse("2006-01-02", bounds[0])
		if err != nil {
			return nil, fmt.Errorf("invalid period start")
		}
		to, err := time.Parse("2006-01-02", bounds[1])
		if err != nil {
			return nil, fmt.Errorf("invalid period end")
		}
		result = append(result, mrp_report_uc.DateRange{From: from, To: to})
	}
	return result, nil
}
func reportFilter(r *http.Request) (mrp_report_uc.Filter, error) {
	q := r.URL.Query()
	from, err := reportDateStrict(q.Get("from"))
	if err != nil {
		return mrp_report_uc.Filter{}, err
	}
	to, err := reportDateStrict(q.Get("to"))
	if err != nil {
		return mrp_report_uc.Filter{}, err
	}
	orders, err := reportInt64List(q.Get("sales_orders"))
	if err != nil {
		return mrp_report_uc.Filter{}, err
	}
	periods, err := reportPeriods(q.Get("periods"))
	if err != nil {
		return mrp_report_uc.Filter{}, err
	}
	productionOrders, err := reportInt64List(q.Get("production_orders"))
	if err != nil {
		return mrp_report_uc.Filter{}, err
	}
	loads, err := reportInt64List(q.Get("loads"))
	if err != nil {
		return mrp_report_uc.Filter{}, err
	}
	quantity := decimal.Zero
	if raw := q.Get("quantity"); raw != "" {
		quantity, err = decimal.NewFromString(raw)
		if err != nil {
			return mrp_report_uc.Filter{}, fmt.Errorf("invalid quantity")
		}
	}
	return mrp_report_uc.Filter{
		PlanCode: reportInt64(q.Get("plan_code")), ItemCode: reportInt64(q.Get("item_code")), Planner: reportInt64(q.Get("planner")),
		Warehouse: reportInt64(q.Get("warehouse_id")), From: from, To: to,
		ItemType: q.Get("item_type"), Position: q.Get("position"), OnlyAvailable: q.Get("only_available") == "true",
		ClassificationMaskCode: reportInt64(q.Get("classification_mask_code")), ClassificationCode: q.Get("classification_code"),
		Layout: q.Get("layout"), OrderBy1: q.Get("order_by_1"), OrderBy2: q.Get("order_by_2"), BreakBy: q.Get("break_by"),
		IncludeDrawings: q.Get("include_drawings") == "true", IncludeSalesOrders: q.Get("include_sales_orders") == "true",
		OnlyWithMessage: q.Get("only_with_message") == "true", OnlyStockWithoutReason: q.Get("only_stock_without_reason") == "true",
		SalesOrderCodes: orders, Quantity: quantity, PlanningType: q.Get("planning_type"), OrderPosition: q.Get("order_position"), Periods: periods,
		ExplosionOption: q.Get("explosion_option"), ListMode: q.Get("list_mode"), DescriptionType: q.Get("description_type"),
		ConsiderItemWarehouses: q.Get("consider_item_warehouses") == "true", ProductionOrderCodes: productionOrders, LoadCodes: loads}, nil
}
func reportResult(w http.ResponseWriter, result []mrp_report_uc.ReportRow, err error) {
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *MRPReportHandler) Profile(w http.ResponseWriter, r *http.Request) {
	filter, err := reportFilter(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Profile(r.Context(), filter)
	reportResult(w, result, err)
}
func (h *MRPReportHandler) Availability(w http.ResponseWriter, r *http.Request) {
	filter, err := reportFilter(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Availability(r.Context(), filter)
	reportResult(w, result, err)
}
func (h *MRPReportHandler) GroupedNeeds(w http.ResponseWriter, r *http.Request) {
	filter, err := reportFilter(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.GroupedNeeds(r.Context(), filter)
	reportResult(w, result, err)
}
func (h *MRPReportHandler) ReorderPoint(w http.ResponseWriter, r *http.Request) {
	filter, err := reportFilter(r)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.ReorderPoint(r.Context(), filter)
	reportResult(w, result, err)
}
func (h *MRPReportHandler) Explosion(w http.ResponseWriter, r *http.Request) {
	filter, filterErr := reportFilter(r)
	if filterErr != nil {
		security.RespondError(w, http.StatusBadRequest, filterErr.Error())
		return
	}
	item, err := strconv.ParseInt(chi.URLParam(r, "itemCode"), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	quantity := decimal.Zero
	if raw := r.URL.Query().Get("quantity"); raw != "" {
		quantity, err = decimal.NewFromString(raw)
		if err != nil {
			security.RespondError(w, http.StatusBadRequest, "invalid quantity")
			return
		}
	}
	result, err := h.uc.Explosion(r.Context(), item, quantity, reportDate(r.URL.Query().Get("at")), filter)
	reportResult(w, result, err)
}
