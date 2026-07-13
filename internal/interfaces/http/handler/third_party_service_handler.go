package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	thirdpartyuc "github.com/FelipePn10/panossoerp/internal/application/usecase/third_party_service_uc"
	domain "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ThirdPartyServiceHandler struct{ uc *thirdpartyuc.UseCase }

func NewThirdPartyServiceHandler(uc *thirdpartyuc.UseCase) *ThirdPartyServiceHandler {
	return &ThirdPartyServiceHandler{uc: uc}
}
func actor(r *http.Request) (uuid.UUID, bool) {
	u, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser)
	if !ok || u == nil {
		return uuid.Nil, false
	}
	id, e := uuid.Parse(u.ID)
	return id, e == nil
}
func body(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if e := json.NewDecoder(r.Body).Decode(v); e != nil {
		jsonError(w, 400, "invalid payload")
		return false
	}
	return true
}
func pathInt(r *http.Request, name string) (int64, bool) {
	v, e := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	return v, e == nil && v > 0
}
func optInt(q string) *int64 {
	if q == "" {
		return nil
	}
	v, e := strconv.ParseInt(q, 10, 64)
	if e != nil {
		return nil
	}
	return &v
}
func optDate(q string) *time.Time {
	if q == "" {
		return nil
	}
	v, e := time.Parse("2006-01-02", q)
	if e != nil {
		return nil
	}
	return &v
}
func (h *ThirdPartyServiceHandler) CreatePrice(w http.ResponseWriter, r *http.Request) {
	var d request.ThirdPartyPriceDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.CreatePrice(r.Context(), d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *ThirdPartyServiceHandler) UpdatePrice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	var d request.ThirdPartyPriceDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.UpdatePrice(r.Context(), id, d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) DeletePrice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	if e := h.uc.DeletePrice(r.Context(), id, r.URL.Query().Get("reason"), by); e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *ThirdPartyServiceHandler) GetPrice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	v, e := h.uc.GetPrice(r.Context(), id)
	if e != nil {
		jsonError(w, 404, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) ListPrices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.PriceFilter{ItemFrom: optInt(q.Get("item_from")), ItemTo: optInt(q.Get("item_to")), SupplierFrom: optInt(q.Get("supplier_from")), SupplierTo: optInt(q.Get("supplier_to")), OperationID: optInt(q.Get("operation_id")), ReferenceDate: optDate(q.Get("reference_date")), PriceType: q.Get("price_type"), OrderBy: q.Get("order_by"), ItemSearch: q.Get("item_search"), SupplierSearch: q.Get("supplier_search"), ClassificationMaskCode: optInt(q.Get("classification_mask_code")), ClassificationCodes: split(q.Get("classification_codes"))}
	if v := q.Get("mask"); v != "" {
		f.Mask = &v
	}
	if v := q.Get("preferred"); v != "" {
		b, e := strconv.ParseBool(v)
		if e != nil {
			jsonError(w, 400, "invalid preferred")
			return
		}
		f.Preferred = &b
	}
	f.Limit, _ = strconv.Atoi(q.Get("limit"))
	f.Offset, _ = strconv.Atoi(q.Get("offset"))
	v, e := h.uc.ListPrices(r.Context(), f)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) ResolvePrice(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	item := optInt(q.Get("item_code"))
	op := optInt(q.Get("operation_id"))
	if item == nil || op == nil {
		jsonError(w, 400, "item_code and operation_id are required")
		return
	}
	supplier := int64(0)
	if v := optInt(q.Get("supplier_code")); v != nil {
		supplier = *v
	}
	at := time.Now()
	if v := optDate(q.Get("reference_date")); v != nil {
		at = *v
	}
	attrs := map[string]string{}
	if raw := q.Get("attributes"); raw != "" {
		if e := json.Unmarshal([]byte(raw), &attrs); e != nil {
			jsonError(w, 400, "attributes must be a JSON object")
			return
		}
	}
	v, e := h.uc.ResolvePrice(r.Context(), *item, q.Get("mask"), supplier, *op, at, attrs)
	if e != nil {
		jsonError(w, 404, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) ResolveCost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	item, op := optInt(q.Get("item_code")), optInt(q.Get("operation_id"))
	if item == nil || op == nil {
		jsonError(w, 400, "item_code and operation_id are required")
		return
	}
	at := time.Now()
	if v := optDate(q.Get("reference_date")); v != nil {
		at = *v
	}
	mode := strings.ToUpper(q.Get("mode"))
	if mode == "" {
		mode = "STANDARD"
	}
	v, e := h.uc.CostPerUnit(r.Context(), *item, q.Get("mask"), *op, at, mode)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, response.ThirdPartyCostResponse{Mode: mode, ItemCode: *item, OperationID: *op, GrossUnitCost: v.GrossUnitCost.StringFixed(6), Freight: v.Freight.StringFixed(6), RecoverableTaxes: v.RecoverableTaxes.StringFixed(6), ConversionFactor: v.ConversionFactor.StringFixed(8), EffectiveUnitCost: v.EffectiveUnitCost.StringFixed(6)})
}
func (h *ThirdPartyServiceHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	v, e := h.uc.History(r.Context(), id)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) Readjust(w http.ResponseWriter, r *http.Request) {
	var d request.ThirdPartyReadjustDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.Readjust(r.Context(), d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *ThirdPartyServiceHandler) CopyMove(w http.ResponseWriter, r *http.Request) {
	var d request.ThirdPartyCopyMoveDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.CopyMove(r.Context(), d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *ThirdPartyServiceHandler) CreateOrders(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "productionOrderID")
	if !ok {
		jsonError(w, 400, "invalid production order id")
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.CreateOrders(r.Context(), id, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *ThirdPartyServiceHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.OrderFilter{PlanCode: optInt(q.Get("plan_code")), ItemFrom: optInt(q.Get("item_from")), ItemTo: optInt(q.Get("item_to")), ProductionOrderID: optInt(q.Get("production_order_id")), ServiceOrderCode: optInt(q.Get("service_order_code")), OperationID: optInt(q.Get("operation_id")), SupplierCode: optInt(q.Get("supplier_code")), PurchaseOrderCode: optInt(q.Get("purchase_order_code")), ProductionOrderIDs: splitInt64(q.Get("production_order_ids")), ServiceOrderCodes: splitInt64(q.Get("service_order_codes")), OperationIDs: splitInt64(q.Get("operation_ids")), SupplierCodes: splitInt64(q.Get("supplier_codes")), PurchaseOrderCodes: splitInt64(q.Get("purchase_order_codes")), From: optDate(q.Get("from")), To: optDate(q.Get("to")), EmittedFrom: optDate(q.Get("emitted_from")), EmittedTo: optDate(q.Get("emitted_to")), DeliveryFrom: optDate(q.Get("delivery_from")), DeliveryTo: optDate(q.Get("delivery_to")), Position: q.Get("position"), OnlyKanban: q.Get("only_kanban") == "true", Statuses: split(q.Get("statuses")), ItemSearch: q.Get("item_search"), SupplierSearch: q.Get("supplier_search"), OrderBy: q.Get("order_by"), ClassificationMaskCode: optInt(q.Get("classification_mask_code")), ClassificationCodes: split(q.Get("classification_codes"))}
	f.Limit, _ = strconv.Atoi(q.Get("limit"))
	f.Offset, _ = strconv.Atoi(q.Get("offset"))
	v, e := h.uc.ListOrders(r.Context(), f)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	if q.Get("include_history") == "true" {
		for index := range v {
			if v[index].ID == 0 {
				continue
			}
			history, historyErr := h.uc.OrderHistory(r.Context(), v[index].ID)
			if historyErr != nil {
				jsonError(w, 422, historyErr.Error())
				return
			}
			v[index].History = history
		}
	}
	if exported, exportErr := export.WriteSlice(w, r, "Ordens de serviços de terceiros", "ordens_servicos_terceiros", v); exported {
		if exportErr != nil {
			return
		}
		return
	}
	jsonResponse(w, 200, v)
}
func splitInt64(v string) []int64 {
	parts := split(v)
	out := make([]int64, 0, len(parts))
	for _, raw := range parts {
		if n, e := strconv.ParseInt(raw, 10, 64); e == nil && n > 0 {
			out = append(out, n)
		}
	}
	return out
}
func split(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(strings.ToUpper(v), ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
func (h *ThirdPartyServiceHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	v, e := h.uc.GetOrder(r.Context(), id)
	if e != nil {
		jsonError(w, 404, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	var d request.ThirdPartyOrderStatusDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.UpdateOrderStatus(r.Context(), id, d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) AddMovement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	var d request.ThirdPartyMovementDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.AddMovement(r.Context(), id, d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *ThirdPartyServiceHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	v, e := h.uc.ListMovements(r.Context(), id)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) UpsertGlobalConversion(w http.ResponseWriter, r *http.Request) {
	var d request.GlobalUnitConversionDTO
	if !body(w, r, &d) {
		return
	}
	by, ok := actor(r)
	if !ok {
		jsonError(w, 401, "invalid authenticated user")
		return
	}
	v, e := h.uc.UpsertGlobalConversion(r.Context(), d, by)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 201, v)
}
func (h *ThirdPartyServiceHandler) ListGlobalConversions(w http.ResponseWriter, r *http.Request) {
	v, e := h.uc.ListGlobalConversions(r.Context())
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
func (h *ThirdPartyServiceHandler) DeleteGlobalConversion(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	if e := h.uc.DeleteGlobalConversion(r.Context(), id); e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	w.WriteHeader(204)
}
func (h *ThirdPartyServiceHandler) OrderHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		jsonError(w, 400, "invalid id")
		return
	}
	v, e := h.uc.OrderHistory(r.Context(), id)
	if e != nil {
		jsonError(w, 422, e.Error())
		return
	}
	jsonResponse(w, 200, v)
}
