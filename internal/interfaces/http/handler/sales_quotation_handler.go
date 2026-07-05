package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc"
	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/go-chi/chi/v5"
)

type SalesQuotationHandler struct {
	uc        *sales_quotation_uc.UseCase
	convertUC *sales_quotation_uc.ConvertUseCase
}

func NewSalesQuotationHandler(uc *sales_quotation_uc.UseCase, convertUC *sales_quotation_uc.ConvertUseCase) *SalesQuotationHandler {
	return &SalesQuotationHandler{uc: uc, convertUC: convertUC}
}

func (h *SalesQuotationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) Update(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UpdateSalesQuotationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.uc.Update(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) Get(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.Get(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := parseQuotationFilter(r)
	result, err := h.uc.List(r.Context(), filter)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.CancelSalesQuotationDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Code = code
	if err := h.uc.Cancel(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) Attend(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.AttendSalesQuotationDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Code = code
	if err := h.uc.Attend(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) Uncancel(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.UncancelSalesQuotationDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Code = code
	if err := h.uc.Uncancel(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ChangeSalesQuotationStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	if err := h.uc.ChangeStatus(r.Context(), dto); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SalesQuotationHandler) Report(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.Report(r.Context(), parseQuotationFilter(r))
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) Convert(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	var dto request.ConvertSalesQuotationDTO
	_ = json.NewDecoder(r.Body).Decode(&dto)
	dto.Code = code
	result, err := h.convertUC.Execute(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateSalesQuotationItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.CreateItem(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusCreated, result)
}

func (h *SalesQuotationHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "itemCode")
	if !ok {
		return
	}
	var dto request.UpdateSalesQuotationItemDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		security.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dto.Code = code
	result, err := h.uc.UpdateItem(r.Context(), dto)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "code")
	if !ok {
		return
	}
	result, err := h.uc.ListItems(r.Context(), code)
	if err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	security.RespondJSON(w, http.StatusOK, result)
}

func (h *SalesQuotationHandler) CancelItem(w http.ResponseWriter, r *http.Request) {
	code, ok := parseQuotationCode(w, r, "itemCode")
	if !ok {
		return
	}
	if err := h.uc.CancelItem(r.Context(), code); err != nil {
		security.RespondUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseQuotationCode(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	code, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		security.RespondError(w, http.StatusBadRequest, "invalid code")
		return 0, false
	}
	return code, true
}

func parseQuotationFilter(r *http.Request) quoterepo.SalesQuotationFilter {
	q := r.URL.Query()
	var filter quoterepo.SalesQuotationFilter
	if raw := q.Get("customer_code"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.CustomerCode = &v
		}
	}
	if raw := q.Get("status"); raw != "" {
		status := quoteentity.SalesQuotationStatus(raw)
		filter.Status = &status
	}
	if from := q.Get("from"); from != "" {
		t := datetime.ParseDatePtr(&from)
		filter.From = t
	}
	if to := q.Get("to"); to != "" {
		t := datetime.ParseDatePtr(&to)
		filter.To = t
	}
	if purchaseOrder := q.Get("purchase_order_number"); purchaseOrder != "" {
		filter.PurchaseOrderNumber = &purchaseOrder
	}
	if freightType := q.Get("freight_type"); freightType != "" {
		filter.FreightType = &freightType
	}
	return filter
}
