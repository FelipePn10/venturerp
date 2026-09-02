package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
	filter, err := parseSalesOrderFilter(r, true)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	results, err := h.listAdvancedUC.Execute(r.Context(), filter)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, results)
}

func (h *SalesOrderHandler) Report(w http.ResponseWriter, r *http.Request) {
	filter, err := parseSalesOrderFilter(r, false)
	if err != nil {
		security.RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	result, err := h.reportUC.Execute(r.Context(), filter)
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
		security.RespondError(w, http.StatusBadRequest, "código do cliente inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código do item inválido")
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
		security.RespondError(w, http.StatusBadRequest, "código inválido")
		return 0, false
	}
	return code, true
}

func parseSalesOrderFilter(r *http.Request, paginate bool) (orderrepo.SalesOrderFilter, error) {
	q := r.URL.Query()
	filter := orderrepo.SalesOrderFilter{Search: strings.TrimSpace(q.Get("search"))}
	var err error
	if filter.CustomerCode, err = parsePositiveInt64Query(q.Get("customer_code"), "customer_code"); err != nil {
		return filter, err
	}
	if filter.ItemCode, err = parsePositiveInt64Query(q.Get("item_code"), "item_code"); err != nil {
		return filter, err
	}
	if filter.RepresentativeCode, err = parsePositiveInt64Query(q.Get("representative_code"), "representative_code"); err != nil {
		return filter, err
	}
	if filter.PaymentTermCode, err = parsePositiveInt64Query(q.Get("payment_term_code"), "payment_term_code"); err != nil {
		return filter, err
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
	if raw := strings.ToUpper(strings.TrimSpace(q.Get("workflow_status"))); raw != "" {
		switch raw {
		case "ANALISADO", "ATENDIDO", "CONFERIDO", "ATRASADO", "FATURADO":
			filter.WorkflowStatus = &raw
		default:
			return filter, fmt.Errorf("workflow_status deve ser ANALISADO, ATENDIDO, CONFERIDO, ATRASADO ou FATURADO")
		}
	}
	if raw := q.Get("is_blocked"); raw != "" {
		v, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return filter, fmt.Errorf("o campo is_blocked deve ser verdadeiro ou falso")
		}
		filter.IsBlocked = &v
	}
	if filter.EmissionFrom, err = parseISODateQuery(q.Get("emission_from"), "emission_from"); err != nil {
		return filter, err
	}
	if filter.EmissionTo, err = parseISODateQuery(q.Get("emission_to"), "emission_to"); err != nil {
		return filter, err
	}
	if filter.DeliveryFrom, err = parseISODateQuery(q.Get("delivery_from"), "delivery_from"); err != nil {
		return filter, err
	}
	if filter.DeliveryTo, err = parseISODateQuery(q.Get("delivery_to"), "delivery_to"); err != nil {
		return filter, err
	}
	if filter.EmissionFrom != nil && filter.EmissionTo != nil && filter.EmissionFrom.After(*filter.EmissionTo) {
		return filter, fmt.Errorf("emission_from não pode ser posterior a emission_to")
	}
	if filter.DeliveryFrom != nil && filter.DeliveryTo != nil && filter.DeliveryFrom.After(*filter.DeliveryTo) {
		return filter, fmt.Errorf("delivery_from não pode ser posterior a delivery_to")
	}
	if paginate {
		filter.Limit = 100
		if raw := q.Get("limit"); raw != "" {
			v, parseErr := strconv.ParseInt(raw, 10, 32)
			if parseErr != nil || v < 1 || v > 500 {
				return filter, fmt.Errorf("o campo limit deve estar entre 1 e 500")
			}
			filter.Limit = int32(v)
		}
		if raw := q.Get("offset"); raw != "" {
			v, parseErr := strconv.ParseInt(raw, 10, 32)
			if parseErr != nil || v < 0 {
				return filter, fmt.Errorf("o campo offset deve ser um número inteiro maior ou igual a zero")
			}
			filter.Offset = int32(v)
		}
	}
	return filter, nil
}

func parsePositiveInt64Query(raw, field string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return nil, fmt.Errorf("o campo %s deve ser um código inteiro positivo", field)
	}
	return &v, nil
}

func parseISODateQuery(raw, field string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("o campo %s deve usar o formato AAAA-MM-DD", field)
	}
	return &v, nil
}
