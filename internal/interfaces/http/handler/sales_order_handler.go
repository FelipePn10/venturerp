package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"
)

type SalesOrderHandler struct {
	createUC         *sales_order_uc.CreateSalesOrderUseCase
	updateUC         *sales_order_uc.UpdateSalesOrderUseCase
	getUC            *sales_order_uc.GetSalesOrderUseCase
	listUC           *sales_order_uc.ListSalesOrdersUseCase
	listByCustomerUC *sales_order_uc.ListSalesOrdersByCustomerUseCase
	listByStatusUC   *sales_order_uc.ListSalesOrdersByStatusUseCase
	cancelUC         *sales_order_uc.CancelSalesOrderUseCase
	blockUC          *sales_order_uc.BlockSalesOrderUseCase
	unblockUC        *sales_order_uc.UnblockSalesOrderUseCase
	changeStatusUC   *sales_order_uc.ChangeStatusSalesOrderUseCase
	listAdvancedUC   *sales_order_uc.ListSalesOrdersAdvancedUseCase
	reportUC         *sales_order_uc.SalesOrderReportUseCase
	analyzeUC        *sales_order_uc.AnalyzeSalesOrderUseCase
	releaseUC        *sales_order_uc.ReleaseSalesOrderUseCase
	attendUC         *sales_order_uc.AttendSalesOrderUseCase
	conferUC         *sales_order_uc.ConferSalesOrderUseCase
	delayReasonUC    *sales_order_uc.SaveSalesOrderDelayReasonUseCase

	createItemUC *sales_order_uc.CreateSalesOrderItemUseCase
	updateItemUC *sales_order_uc.UpdateSalesOrderItemUseCase
	listItemsUC  *sales_order_uc.ListSalesOrderItemsUseCase
	cancelItemUC *sales_order_uc.CancelSalesOrderItemUseCase
}

func (h *SalesOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesOrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.UpdateSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.updateUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesOrderHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	result, err := h.getUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusNotFound, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	results, err := h.listUC.Execute(r.Context())
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) Search(w http.ResponseWriter, r *http.Request) {
	results, err := h.listAdvancedUC.Execute(r.Context(), parseSalesOrderFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) Report(w http.ResponseWriter, r *http.Request) {
	result, err := h.reportUC.Execute(r.Context(), parseSalesOrderFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesOrderHandler) ListByCustomer(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "customerCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid customer code")
		return
	}
	results, err := h.listByCustomerUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	results, err := h.listByStatusUC.Execute(r.Context(), status)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.CancelSalesOrderDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Code = code
	if err := h.cancelUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesOrderCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.AnalyzeSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.analyzeUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Release(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesOrderCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ReleaseSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.releaseUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Attend(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesOrderCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.AttendSalesOrderDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Code = code
	if err := h.attendUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Confer(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesOrderCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ConferSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.conferUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) SaveDelayReason(w http.ResponseWriter, r *http.Request) {
	code, ok := parseSalesOrderCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.SaveSalesOrderDelayReasonDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.delayReasonUC.Execute(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Block(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.BlockSalesOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.blockUC.Execute(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	if err := h.unblockUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesOrderHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	var dto request.ChangeStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.changeStatusUC.Execute(r.Context(), dto); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Item handlers

func (h *SalesOrderHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesOrderItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.createItemUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesOrderHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	var dto request.UpdateSalesOrderItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.updateItemUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesOrderHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "code")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return
	}
	results, err := h.listItemsUC.Execute(r.Context(), code)
	if err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) CancelItem(w http.ResponseWriter, r *http.Request) {
	codeStr := chi.URLParam(r, "itemCode")
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid item code")
		return
	}
	if err := h.cancelItemUC.Execute(r.Context(), code); err != nil {
		security.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseSalesOrderCode(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	codeStr := chi.URLParam(r, name)
	code, err := strconv.ParseInt(codeStr, 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return 0, false
	}
	return code, true
}

func parseSalesOrderFilter(r *http.Request) orderrepo.SalesOrderFilter {
	q := r.URL.Query()
	var filter orderrepo.SalesOrderFilter
	if raw := q.Get("customer_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.CustomerCode = &v
		}
	}
	if raw := q.Get("representative_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.RepresentativeCode = &v
		}
	}
	if raw := q.Get("payment_term_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.PaymentTermCode = &v
		}
	}
	if raw := q.Get("status"); raw != "" {
		status := orderentity.SalesOrderStatus(raw)
		filter.Status = &status
	}
	if raw := q.Get("commercial_analysis_status"); raw != "" {
		status := orderentity.SalesOrderAnalysisStatus(raw)
		filter.CommercialAnalysisStatus = &status
	}
	if raw := q.Get("financial_analysis_status"); raw != "" {
		status := orderentity.SalesOrderAnalysisStatus(raw)
		filter.FinancialAnalysisStatus = &status
	}
	if raw := q.Get("release_status"); raw != "" {
		status := orderentity.SalesOrderReleaseStatus(raw)
		filter.ReleaseStatus = &status
	}
	if raw := q.Get("conference_status"); raw != "" {
		status := orderentity.SalesOrderConferenceStatus(raw)
		filter.ConferenceStatus = &status
	}
	if raw := q.Get("is_blocked"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			filter.IsBlocked = &v
		}
	}
	if raw := q.Get("emission_from"); raw != "" {
		filter.EmissionFrom = datetime.ParseDatePtr(&raw)
	}
	if raw := q.Get("emission_to"); raw != "" {
		filter.EmissionTo = datetime.ParseDatePtr(&raw)
	}
	if raw := q.Get("delivery_from"); raw != "" {
		filter.DeliveryFrom = datetime.ParseDatePtr(&raw)
	}
	if raw := q.Get("delivery_to"); raw != "" {
		filter.DeliveryTo = datetime.ParseDatePtr(&raw)
	}
	return filter
}
